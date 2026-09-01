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

// Package main demonstrates using the EventLog interface implementation.
//
// This example shows how to:
//   - Create an EventLog reader using the New function
//   - Configure the reader with various options
//   - Read events in batches
//   - Handle checkpoints for resuming reads
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
	"github.com/tianlin/go-windows-eventlog/pkg/eventlog"
)

func main() {
	// Parse command-line flags
	channel := flag.String("channel", "Application", "Event log channel to read")
	checkpointFile := flag.String("checkpoint", ".checkpoint.yml", "Checkpoint file path")
	batchSize := flag.Int("batch", 100, "Batch size for reading events")
	includeXML := flag.Bool("xml", false, "Include XML representation")
	flag.Parse()

	log.Printf("Reading from channel: %s", *channel)
	log.Printf("Using checkpoint file: %s", *checkpointFile)
	log.Printf("Batch size: %d", *batchSize)

	// Create checkpoint manager
	cp, err := checkpoint.NewCheckpoint(*checkpointFile, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create checkpoint: %v", err)
	}
	defer cp.Shutdown()

	// Create event log configuration
	config := eventlog.Config{
		Name:       *channel,
		BatchSize:  *batchSize,
		IncludeXML: *includeXML,
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create event log reader
	reader, err := eventlog.New(config)
	if err != nil {
		log.Fatalf("Failed to create event log reader: %v", err)
	}
	defer reader.Close()
	sourceCheckpoint, ok := reader.(eventlog.CheckpointProvider)
	if !ok {
		log.Fatal("event log reader does not provide a source checkpoint")
	}

	// Set logger (optional)
	stdLogger := eventlog.NewStdLogger(log.Default())
	if setter, ok := reader.(interface{ SetLogger(eventlog.Logger) }); ok {
		setter.SetLogger(stdLogger)
	}

	// Get last checkpoint state
	states := cp.States()
	state := states[reader.Name()]

	if state.RecordNumber > 0 {
		log.Printf("Resuming from record number: %d", state.RecordNumber)
	}

	// Open the event log
	if err := reader.Open(state); err != nil {
		log.Fatalf("Failed to open event log: %v", err)
	}

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
	var totalEvents int
	startTime := time.Now()

	// Read and process events
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
		}

		// Read a batch of events
		records, err := reader.Read()
		state = sourceCheckpoint.Checkpoint()

		// Process records before handling a read error. Read may return a
		// partially filled batch together with a recoverable error.
		for _, record := range records {
			totalEvents++

			// Print event summary
			fmt.Printf("[%d] Event ID: %d, Provider: %s, Time: %s\n",
				totalEvents,
				record.EventIdentifier.ID,
				record.Provider.Name,
				record.TimeCreated.SystemTime.Format(time.RFC3339))

			// Print message if available
			if record.Message != "" {
				fmt.Printf("    Message: %s\n", truncate(record.Message, 100))
			}

			// Print XML if requested
			if *includeXML && record.XML != "" {
				fmt.Printf("    XML: %s\n", truncate(record.XML, 200))
			}

			fmt.Println()

			// Update checkpoint with the last event's offset
			cp.PersistState(record.Offset)
			state = record.Offset
		}

		if err != nil {
			if err.Error() == "EOF" {
				log.Println("Reached end of event log")
				break loop
			}
			log.Printf("Error reading events: %v", err)
			// Recovery must use the source cursor, which may have advanced
			// while a record was skipped or filtered.
			state = sourceCheckpoint.Checkpoint()
			if resetErr := reader.Reset(); resetErr != nil {
				log.Printf("Failed to reset reader: %v", resetErr)
				break loop
			}
			if openErr := reader.Open(state); openErr != nil {
				log.Printf("Failed to reopen reader: %v", openErr)
				break loop
			}
			continue
		}

		if len(records) == 0 {
			// No more events available, wait a bit
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("Processed batch of %d events", len(records))
	}

	// Print statistics
	duration := time.Since(startTime)
	log.Printf("Read %d events in %v (%.2f events/sec)",
		totalEvents,
		duration,
		float64(totalEvents)/duration.Seconds())
}

// truncate truncates a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
