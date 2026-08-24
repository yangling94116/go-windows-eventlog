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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tianlin/go-windows-eventlog/pkg/winevent"
)

func TestRecordFilterMatch(t *testing.T) {
	f, err := newRecordFilter(query{
		IgnoreOlder: time.Hour,
		Provider:    []string{"MyProvider"},
		Level:       "warning",
		EventID:     "100,200-210,-205",
	})
	require.NoError(t, err)

	assert.True(t, f.match(filterTestRecord(time.Now().Add(-30*time.Minute), "MyProvider", 3, 100)))
	assert.False(t, f.match(filterTestRecord(time.Now().Add(-2*time.Hour), "MyProvider", 3, 100)))
	assert.False(t, f.match(filterTestRecord(time.Now(), "OtherProvider", 3, 100)))
	assert.False(t, f.match(filterTestRecord(time.Now(), "MyProvider", 2, 100)))
	assert.False(t, f.match(filterTestRecord(time.Now(), "MyProvider", 3, 205)))
	assert.False(t, f.match(filterTestRecord(time.Now(), "MyProvider", 3, 300)))
}

func TestRecordFilterRejectsInvalidExpressions(t *testing.T) {
	_, err := newRecordFilter(query{Level: "potato"})
	require.Error(t, err)

	_, err = newRecordFilter(query{EventID: "7-3"})
	require.Error(t, err)
}

func filterTestRecord(ts time.Time, provider string, level uint8, id uint32) *Record {
	return &Record{Event: winevent.Event{
		Provider:        winevent.Provider{Name: provider},
		LevelRaw:        level,
		EventIdentifier: winevent.EventIdentifier{ID: id},
		TimeCreated:     winevent.TimeCreated{SystemTime: ts},
	}}
}
