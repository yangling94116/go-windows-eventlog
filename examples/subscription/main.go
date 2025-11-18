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

// Package main demonstrates real-time event subscription.
//
// This example shows how to:
//   - Subscribe to new events as they arrive
//   - Process events asynchronously via channels
//   - Handle backpressure and error conditions
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

	"golang.org/x/sys/windows"

	"github.com/tianlin/go-windows-eventlog/pkg/checkpoint"
	"github.com/tianlin/go-windows-eventlog/pkg/winevent"
	"github.com/tianlin/go-windows-eventlog/pkg/wineventlog"
)

func main() {
	channel := flag.String("channel", "Application", "Event log channel to monitor")
	flag.Parse()

	fmt.Printf("Monitoring %s for new events (press Ctrl+C to stop)...\n\n", *channel)

	// Create checkpoint
	cp, err := checkpoint.NewCheckpoint(".subscription-checkpoint.yml", 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create checkpoint: %v", err)
	}
	defer cp.Shutdown()

	// Get last state
	states := cp.States()
	var bookmark wineventlog.EvtHandle
	if state, exists := states[*channel]; exists {
		log.Printf("Resuming from record number: %d", state.RecordNumber)
		bookmark, err = wineventlog.CreateBookmarkFromRecordID(*channel, state.RecordNumber)
		if err != nil {
			log.Printf("Warning: Failed to create bookmark: %v", err)
			bookmark = wineventlog.NilHandle
		}
	} else {
		bookmark = wineventlog.NilHandle
	}
	if bookmark != wineventlog.NilHandle {
		defer bookmark.Close()
	}

	// Build query
	q := wineventlog.Query{Log: *channel}
	queryStr, err := q.Build()
	if err != nil {
		log.Fatalf("Failed to build query: %v", err)
	}

	// Subscribe to events (this will monitor for new events)
	flags := wineventlog.EvtSubscribeToFutureEvents
	if bookmark != wineventlog.NilHandle {
		flags = wineventlog.EvtSubscribeStartAfterBookmark
	}

	handle, err := wineventlog.Subscribe(
		wineventlog.NilHandle,
		windows.InvalidHandle,
		*channel,
		queryStr,
		bookmark,
		flags,
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to event log: %v", err)
	}
	defer handle.Close()

	// Create event iterator
	iterator, err := wineventlog.NewEventIterator(
		wineventlog.WithSubscription(handle),
		wineventlog.WithBatchSize(10),
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("\nShutting down...")
		cancel()
	}()

	// Process events
	var count int
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
					// Wait a bit before checking again
					time.Sleep(100 * time.Millisecond)
					continue
				}
				log.Printf("Iterator error: %v", err)
				break loop
			}
			// Wait a bit before checking again
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Render the event
		event, _, err := renderer.Render(eventHandle)
		eventHandle.Close()

		if err != nil {
			log.Printf("Failed to render event: %v", err)
			continue
		}

		count++
		printEvent(event, count)

		// Update checkpoint
		cp.PersistState(checkpoint.EventLogState{
			Name:         *channel,
			RecordNumber: event.RecordID,
			Timestamp:    event.TimeCreated.SystemTime,
		})
	}

	log.Printf("Event channel closed. Total events: %d", count)
}

func printEvent(event *winevent.Event, count int) {
	timestamp := event.TimeCreated.SystemTime.Format("15:04:05")
	provider := event.Provider.Name
	code := event.EventIdentifier.ID
	level := event.Level

	// Color-code by level
	levelStr := ""
	if level != "" {
		levelStr = fmt.Sprintf("[%s]", level)
	}

	fmt.Printf("%s %s %s Event %d\n",
		timestamp,
		levelStr,
		provider,
		code)

	if event.Message != "" {
		fmt.Printf("  %s\n", truncate(event.Message, 120))
	}
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
