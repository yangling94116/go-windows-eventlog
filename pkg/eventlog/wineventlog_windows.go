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

package eventlog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"

	"github.com/tianlin/go-windows-eventlog/pkg/checkpoint"
	"github.com/tianlin/go-windows-eventlog/pkg/wineventlog"
)

// winEventLog implements the EventLog interface for reading from the Windows
// Event Log API.
type winEventLog struct {
	config      Config
	query       string
	id          string // Identifier of this event log.
	channelName string // Name of the channel from which to read.
	file        bool   // Reading from file rather than channel.
	maxRead     int    // Maximum number returned in one Read.
	lastRead    checkpoint.EventLogState
	log         Logger

	iterator *wineventlog.EventIterator
	renderer wineventlog.EventRenderer
}

// New creates and returns a new EventLog instance based on the given config.
func New(config Config) (EventLog, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	id := config.ID
	if id == "" {
		id = config.Name
	}

	l := &winEventLog{
		config:      config,
		id:          id,
		channelName: config.Name,
		maxRead:     config.BatchSize,
		log:         &noopLogger{},
	}

	// Build the query
	if config.Query != "" {
		l.query = config.Query
	} else {
		queryLog := config.Name
		if info, err := os.Stat(config.Name); err == nil && info.Mode().IsRegular() {
			path, err := filepath.Abs(config.Name)
			if err != nil {
				return nil, err
			}
			l.file = true
			queryLog = "file://" + path
		}

		winQuery := wineventlog.Query{
			Log:         queryLog,
			IgnoreOlder: config.IgnoreOlder,
			Level:       config.Level,
			EventID:     config.EventID,
			Provider:    config.Provider,
		}

		var err error
		l.query, err = winQuery.Build()
		if err != nil {
			return nil, err
		}
	}

	// Create the renderer
	var err error
	if config.IncludeXML || l.isForwarded() {
		l.renderer = wineventlog.NewXMLRenderer(
			config.Locale,
			l.isForwarded(),
			wineventlog.NilHandle)
	} else {
		l.renderer, err = wineventlog.NewRenderer(
			config.Locale,
			wineventlog.NilHandle)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

func (l *winEventLog) isForwarded() bool {
	c := l.config
	return (c.Forwarded != nil && *c.Forwarded) || (c.Forwarded == nil && c.Name == "ForwardedEvents")
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (l *winEventLog) Name() string {
	return l.id
}

// Channel returns the event log's channel name.
func (l *winEventLog) Channel() string {
	return l.channelName
}

// IsFile returns true if the event log is an evtx file.
func (l *winEventLog) IsFile() bool {
	return l.file
}

func (l *winEventLog) Open(state checkpoint.EventLogState) error {
	l.lastRead = state

	var err error
	l.iterator, err = wineventlog.NewEventIterator(
		wineventlog.WithSubscriptionFactory(func() (handle wineventlog.EvtHandle, err error) {
			return l.open(l.lastRead)
		}),
		wineventlog.WithBatchSize(l.maxRead))
	return err
}

func (l *winEventLog) open(state checkpoint.EventLogState) (wineventlog.EvtHandle, error) {
	var bookmark wineventlog.Bookmark
	if len(state.Bookmark) > 0 {
		var err error
		bookmark, err = wineventlog.NewBookmarkFromXML(state.Bookmark)
		if err != nil {
			return wineventlog.NilHandle, err
		}
		defer bookmark.Close()
	}

	if l.file {
		return l.openFile(state, bookmark)
	}
	return l.openChannel(bookmark)
}

func (l *winEventLog) openFile(state checkpoint.EventLogState, bookmark wineventlog.Bookmark) (wineventlog.EvtHandle, error) {
	path := l.channelName

	h, err := wineventlog.EvtQuery(0, path, l.query, wineventlog.EvtQueryFilePath|wineventlog.EvtQueryForwardDirection)
	if err != nil {
		return wineventlog.NilHandle, fmt.Errorf("failed to get handle to event log file %v: %w", path, err)
	}

	if bookmark > 0 {
		if l.log != nil {
			l.log.Debug("Seeking to bookmark", "timestamp", state.Timestamp, "bookmark", state.Bookmark)
		}

		// This seeks to the last read event and strictly validates that the
		// bookmarked record number exists.
		if err = wineventlog.EvtSeek(h, 0, wineventlog.EvtHandle(bookmark), wineventlog.EvtSeekRelativeToBookmark|wineventlog.EvtSeekStrict); err == nil {
			// Then we advance past the last read event to avoid sending that
			// event again. This won't fail if we're at the end of the file.
			if seekErr := wineventlog.EvtSeek(h, 1, wineventlog.EvtHandle(bookmark), wineventlog.EvtSeekRelativeToBookmark); seekErr != nil {
				err = fmt.Errorf("failed to seek past bookmarked position: %w", seekErr)
			}
		} else {
			if l.log != nil {
				l.log.Warn("Failed to seek to bookmarked location, recovering by reading from beginning",
					"path", path, "error", err)
			}
			if seekErr := wineventlog.EvtSeek(h, 0, 0, wineventlog.EvtSeekRelativeToFirst); seekErr != nil {
				err = fmt.Errorf("failed to seek to beginning of log: %w", seekErr)
			}
		}

		if err != nil {
			return wineventlog.NilHandle, err
		}
	}

	return h, err
}

func (l *winEventLog) openChannel(bookmark wineventlog.Bookmark) (wineventlog.EvtHandle, error) {
	// Using a pull subscription to receive events. See:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385771(v=vs.85).aspx#pull
	signalEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return wineventlog.NilHandle, err
	}
	defer windows.CloseHandle(signalEvent) //nolint:errcheck // This is just a resource release.

	var flags wineventlog.EvtSubscribeFlag
	if bookmark > 0 {
		flags = wineventlog.EvtSubscribeStartAfterBookmark
		if !l.isForwarded() {
			// Use EvtSubscribeStrict to detect when the bookmark is missing and be able to
			// subscribe again from the beginning.
			flags |= wineventlog.EvtSubscribeStrict
		}
	} else {
		flags = wineventlog.EvtSubscribeStartAtOldestRecord
	}

	if l.log != nil {
		l.log.Debug("Using subscription query", "query", l.query)
	}

	h, err := wineventlog.Subscribe(
		0, // Session - nil for localhost
		signalEvent,
		"",                                  // Channel - empty b/c channel is in the query
		l.query,                             // Query - nil means all events
		wineventlog.EvtHandle(bookmark),     // Bookmark - for resuming from a specific event
		flags)

	switch err { //nolint:errorlint // This is an errno or nil.
	case nil:
		return h, nil
	case wineventlog.ERROR_NOT_FOUND, wineventlog.ERROR_EVT_QUERY_RESULT_STALE, wineventlog.ERROR_EVT_QUERY_RESULT_INVALID_POSITION:
		// The bookmarked event was not found, we retry the subscription from the start.
		return wineventlog.Subscribe(0, signalEvent, "", l.query, 0, wineventlog.EvtSubscribeStartAtOldestRecord)
	default:
		return 0, err
	}
}

func (l *winEventLog) Read() ([]Record, error) {
	var records []Record

	for h, ok := l.iterator.Next(); ok; h, ok = l.iterator.Next() {
		record, err := l.processHandle(h)
		if err != nil {
			if l.log != nil {
				l.log.Warn("Dropping event due to rendering error", "error", err)
			}
			continue
		}
		records = append(records, *record)

		// It has read the maximum requested number of events.
		if len(records) >= l.maxRead {
			return records, nil
		}
	}

	// An error occurred while retrieving more events.
	if err := l.iterator.Err(); err != nil {
		return records, err
	}

	// Reader is configured to stop when there are no more events.
	if l.config.NoMoreEvents == "stop" {
		return records, io.EOF
	}

	return records, nil
}

func (l *winEventLog) processHandle(h wineventlog.EvtHandle) (*Record, error) {
	defer h.Close()

	// NOTE: Render can return an error and a partial event.
	evt, xml, err := l.renderer.Render(h)
	if evt == nil {
		return nil, err
	}
	if err != nil {
		evt.RenderErr = append(evt.RenderErr, err.Error())
	}

	r := &Record{
		Event: *evt,
	}

	if l.config.IncludeXML {
		r.XML = xml
	}

	if l.file {
		r.File = l.id
	}

	r.Offset = checkpoint.EventLogState{
		Name:         l.id,
		RecordNumber: r.RecordID,
		Timestamp:    r.TimeCreated.SystemTime,
	}

	if r.Offset.Bookmark, err = l.createBookmarkFromEvent(h); err != nil {
		if l.log != nil {
			l.log.Warn("Failed creating bookmark", "error", err)
		}
	}
	l.lastRead = r.Offset
	return r, nil
}

func (l *winEventLog) createBookmarkFromEvent(evtHandle wineventlog.EvtHandle) (string, error) {
	bookmark, err := wineventlog.NewBookmarkFromEvent(evtHandle)
	if err != nil {
		return "", fmt.Errorf("failed to create new bookmark from event handle: %w", err)
	}
	defer bookmark.Close()

	return bookmark.XML()
}

func (l *winEventLog) Reset() error {
	if l.log != nil {
		l.log.Debug("Closing event log reader handles for reset")
	}
	// Only close the iterator, keep the renderer alive to avoid
	// unnecessarily recreating render contexts. The renderer's
	// systemContext and userContext should remain valid across
	// session resets since they were created independently.
	if l.iterator == nil {
		return nil
	}
	err := l.iterator.Close()
	l.iterator = nil
	return err
}

func (l *winEventLog) Close() error {
	if l.log != nil {
		l.log.Debug("Closing event log reader handles")
	}
	return l.close()
}

func (l *winEventLog) close() error {
	if l.iterator == nil {
		return l.renderer.Close()
	}
	return errors.Join(
		l.iterator.Close(),
		l.renderer.Close(),
	)
}

// SetLogger sets the logger for this event log reader.
func (l *winEventLog) SetLogger(logger Logger) {
	if logger != nil {
		l.log = logger
	}
}
