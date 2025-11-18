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

// Package main demonstrates reading from .evtx files.
//
// This example shows how to:
//   - Read archived event log files
//   - Parse .evtx file format
//   - Extract and display events
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tianlin/go-windows-eventlog/pkg/winevent"
	"github.com/tianlin/go-windows-eventlog/pkg/wineventlog"
)

func main() {
	file := flag.String("file", "", "Path to .evtx file (required)")
	output := flag.String("output", "text", "Output format: text, json")
	limit := flag.Int("limit", 0, "Maximum number of events to read (0 = all)")
	flag.Parse()

	if *file == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Verify file exists
	if _, err := os.Stat(*file); os.IsNotExist(err) {
		log.Fatalf("File does not exist: %s", *file)
	}

	fmt.Printf("Reading events from: %s\n", *file)
	fmt.Printf("Output format: %s\n", *output)
	if *limit > 0 {
		fmt.Printf("Limit: %d events\n", *limit)
	}
	fmt.Println()

	// Get absolute path and convert to file:// URI
	absPath, err := filepath.Abs(*file)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	fileURI := "file://" + absPath

	// Build query
	q := wineventlog.Query{Log: fileURI}
	queryStr, err := q.Build()
	if err != nil {
		log.Fatalf("Failed to build query: %v", err)
	}

	// Open the .evtx file
	handle, err := wineventlog.EvtQuery(
		wineventlog.NilHandle,
		fileURI,
		queryStr,
		wineventlog.EvtQueryFilePath|wineventlog.EvtQueryForwardDirection,
	)
	if err != nil {
		log.Fatalf("Failed to open .evtx file: %v", err)
	}
	defer handle.Close()

	// Create event iterator
	iterator, err := wineventlog.NewEventIterator(
		wineventlog.WithSubscription(handle),
		wineventlog.WithBatchSize(100),
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

	var count int

loop:
	for {
		if *limit > 0 && count >= *limit {
			break
		}

		eventHandle, ok := iterator.Next()
		if !ok {
			err := iterator.Err()
			if err != nil {
				// Check if this is ERROR_NO_MORE_ITEMS
				if errno, ok := err.(syscall.Errno); ok && errno == 259 {
					break loop
				}
				log.Printf("Iterator error: %v", err)
				break loop
			}
			break loop
		}

		// Render the event
		event, _, err := renderer.Render(eventHandle)
		eventHandle.Close()

		if err != nil {
			log.Printf("Failed to render event: %v", err)
			continue
		}

		count++

		switch *output {
		case "json":
			printJSON(event)
		default:
			printText(event)
		}
	}

	fmt.Printf("\nTotal events read: %d\n", count)
}

func printText(event *winevent.Event) {
	fmt.Printf("Event ID: %d\n", event.EventIdentifier.ID)
	fmt.Printf("Provider: %s\n", event.Provider.Name)
	fmt.Printf("Time: %s\n", event.TimeCreated.SystemTime)
	if event.Level != "" {
		fmt.Printf("Level: %s\n", event.Level)
	}
	if event.Message != "" {
		fmt.Printf("Message: %s\n", event.Message)
	}
	fmt.Println(strings.Repeat("-", 80))
}

func printJSON(event *winevent.Event) {
	fields := event.Fields()
	data, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}
	fmt.Println(string(data))
}
