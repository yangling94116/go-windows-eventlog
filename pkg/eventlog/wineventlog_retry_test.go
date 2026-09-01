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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tianlin/go-windows-eventlog/pkg/checkpoint"
	"github.com/tianlin/go-windows-eventlog/pkg/winevent"
	"github.com/tianlin/go-windows-eventlog/pkg/wineventlog"
)

type staticEventRenderer struct {
	event *winevent.Event
}

func (r staticEventRenderer) Render(wineventlog.EvtHandle) (*winevent.Event, string, error) {
	return r.event, "", nil
}

func (staticEventRenderer) Close() error { return nil }

func TestHandleProcessErrorSkipsRenderFailureAfterRetryLimit(t *testing.T) {
	reader := &winEventLog{
		channelName: "ForwardedEvents",
		lastRead:    checkpoint.EventLogState{RecordNumber: 100, Bookmark: "previous"},
		log:         noopLogger{},
	}
	failure := &renderNoEventError{cause: assert.AnError, bookmark: "poison"}

	for attempt := 1; attempt <= renderNoEventRetryLimit; attempt++ {
		require.ErrorIs(t, reader.handleProcessError(failure), errRenderNoEvent)
	}
	require.NoError(t, reader.handleProcessError(failure))
	assert.Equal(t, "poison", reader.lastRead.Bookmark)
	assert.Zero(t, reader.lastRead.RecordNumber)
}

func TestHandleProcessErrorAcceptsRecordGapAfterRetryLimit(t *testing.T) {
	reader := &winEventLog{
		channelName: "Security",
		lastRead:    checkpoint.EventLogState{RecordNumber: 100, Bookmark: "previous"},
		log:         noopLogger{},
	}
	failure := &gapDetectedError{
		channel:  "Security",
		previous: 100,
		current:  105,
		bookmark: "current",
	}

	for attempt := 1; attempt <= recordIDGapRetryLimit; attempt++ {
		require.ErrorIs(t, reader.handleProcessError(failure), errRecordIDGap)
	}
	require.NoError(t, reader.handleProcessError(failure))
	assert.Equal(t, "current", reader.lastRead.Bookmark)
	assert.Equal(t, uint64(105), reader.lastRead.RecordNumber)
	assert.Equal(t, reader.lastRead, reader.Checkpoint())
}

func TestCheckpointReturnsLatestSourceState(t *testing.T) {
	reader := &winEventLog{
		lastRead: checkpoint.EventLogState{
			Name:         "Security",
			RecordNumber: 105,
			Bookmark:     "current",
		},
	}

	assert.Equal(t, checkpoint.EventLogState{
		Name:         "Security",
		RecordNumber: 105,
		Bookmark:     "current",
	}, reader.Checkpoint())
}

func TestRecoveryCountersResetForDifferentFailures(t *testing.T) {
	reader := &winEventLog{log: noopLogger{}}

	first := &gapDetectedError{channel: "Security", previous: 100, current: 105}
	second := &gapDetectedError{channel: "Security", previous: 105, current: 110}
	require.Error(t, reader.handleProcessError(first))
	require.Error(t, reader.handleProcessError(first))
	require.Error(t, reader.handleProcessError(second))
	assert.Equal(t, 1, reader.gapRetryCount)
}

func TestCustomQueryAllowsNonContiguousRecordIDs(t *testing.T) {
	reader := &winEventLog{
		config:   Config{Query: `<QueryList><Query Id="0"><Select Path="Security">*</Select></Query></QueryList>`},
		lastRead: checkpoint.EventLogState{RecordNumber: 100},
		log:      noopLogger{},
		renderer: staticEventRenderer{event: &winevent.Event{RecordID: 105}},
	}

	record, err := reader.processHandle(wineventlog.NilHandle)
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, uint64(105), record.RecordID)
}

func TestForwardedEventsAllowsNonContiguousRecordIDs(t *testing.T) {
	reader := &winEventLog{
		config:      Config{Name: "ForwardedEvents"},
		channelName: "ForwardedEvents",
		lastRead:    checkpoint.EventLogState{RecordNumber: 100},
		log:         noopLogger{},
		renderer:    staticEventRenderer{event: &winevent.Event{RecordID: 105}},
	}

	record, err := reader.processHandle(wineventlog.NilHandle)
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, uint64(105), record.RecordID)
}

func TestExplicitForwardedReaderAllowsNonContiguousRecordIDs(t *testing.T) {
	forwarded := true
	reader := &winEventLog{
		config:   Config{Name: "Security", Forwarded: &forwarded},
		lastRead: checkpoint.EventLogState{RecordNumber: 100},
		log:      noopLogger{},
		renderer: staticEventRenderer{event: &winevent.Event{RecordID: 105}},
	}

	record, err := reader.processHandle(wineventlog.NilHandle)
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, uint64(105), record.RecordID)
}
