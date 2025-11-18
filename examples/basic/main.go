// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows

// Package main demonstrates basic usage of the go-windows-eventlog library.
//
// This example shows how to:
//   - Configure an event log reader
//   - Read events from the Application channel
//   - Process events with a callback function
//   - Handle graceful shutdown
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tianlin/go-windows-eventlog/pkg/checkpoint"
	"github.com/tianlin/go-windows-eventlog/pkg/wineventlog"
)

func main() {
	// Parse command-line flags
	channel := flag.String("channel", "Application", "Event log channel to read")
	checkpointFile := flag.String("checkpoint", ".checkpoint.yml", "Checkpoint file path")
	batchSize := flag.Int("batch", 100, "Batch size for reading events")
	query := flag.String("query", "*", "XPath query for filtering events")
	flag.Parse()

	log.Printf("Reading from channel: %s", *channel)
	log.Printf("Using checkpoint file: %s", *checkpointFile)
	log.Printf("Batch size: %d", *batchSize)
	if *query != "*" {
		log.Printf("Query filter: %s", *query)
	}

	// Create checkpoint
	cp, err := checkpoint.NewCheckpoint(*checkpointFile, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create checkpoint: %v", err)
	}
	defer cp.Shutdown()

	// Get last state
	states := cp.States()
	var recordNum uint64
	if state, exists := states[*channel]; exists {
		recordNum = state.RecordNumber
		log.Printf("Resuming from record number: %d", recordNum)
	}

	// Build query string
	queryStr := *query
	if queryStr == "*" {
		q := wineventlog.Query{
			Log: *channel,
		}
		queryStr, err = q.Build()
		if err != nil {
			log.Fatalf("Failed to build query: %v", err)
		}
	}

	// Open event log query
	handle, err := wineventlog.EvtQuery(
		wineventlog.NilHandle,
		*channel,
		queryStr,
		wineventlog.EvtQueryChannelPath|wineventlog.EvtQueryForwardDirection,
	)
	if err != nil {
		log.Fatalf("Failed to open event log: %v", err)
	}
	defer handle.Close()

	// Seek to bookmark if we have one
	if recordNum > 0 {
		bookmark, err := wineventlog.CreateBookmarkFromRecordID(*channel, recordNum)
		if err != nil {
			log.Printf("Warning: Failed to create bookmark: %v", err)
		} else {
			defer bookmark.Close()
			err = wineventlog.EvtSeek(handle, 0, bookmark, wineventlog.EvtSeekRelativeToBookmark)
			if err != nil {
				log.Printf("Warning: Failed to seek to bookmark: %v", err)
			}
		}
	}

	// Create event iterator
	iterator, err := wineventlog.NewEventIterator(
		wineventlog.WithSubscription(handle),
		wineventlog.WithBatchSize(*batchSize),
	)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iterator.Close()

	// Create renderer
	renderer, err := wineventlog.NewRenderer(0, wineventlog.NilHandle)
	if err != nil {
		log.Fatalf("Failed to create renderer: %v", err)
	}
	defer renderer.Close()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Event counter
	var eventCount int
	startTime := time.Now()

	// Read and process events
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
		}

		eventHandle, ok := iterator.Next()
		if !ok {
			err := iterator.Err()
			if err != nil {
				// Check if this is ERROR_NO_MORE_ITEMS
				if errno, ok := err.(syscall.Errno); ok && errno == 259 {
					// No more events
					break loop
				}
				log.Printf("Iterator error: %v", err)
				break loop
			}
			// No more events
			break loop
		}

		// Render the event
		event, _, err := renderer.Render(eventHandle)
		eventHandle.Close()

		if err != nil {
			log.Printf("Failed to render event: %v", err)
			continue
		}

		eventCount++

		// Print event summary
		fmt.Printf("[%d] Event ID: %d, Provider: %s, Time: %s\n",
			eventCount,
			event.EventIdentifier.ID,
			event.Provider.Name,
			event.TimeCreated.SystemTime.Format(time.RFC3339))

		// Print message if available
		if event.Message != "" {
			fmt.Printf("    Message: %s\n", truncate(event.Message, 100))
		}

		fmt.Println()

		// Update checkpoint
		cp.PersistState(checkpoint.EventLogState{
			Name:         *channel,
			RecordNumber: event.RecordID,
			Timestamp:    event.TimeCreated.SystemTime,
		})
	}

	// Print statistics
	duration := time.Since(startTime)
	log.Printf("Read %d events in %v (%.2f events/sec)",
		eventCount,
		duration,
		float64(eventCount)/duration.Seconds())
}

// truncate truncates a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
