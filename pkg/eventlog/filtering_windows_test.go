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

	"github.com/stretchr/testify/require"
)

func TestNewUsesUnfilteredWindowsQuery(t *testing.T) {
	api, err := New(Config{
		Name:        "ForwardedEvents",
		IgnoreOlder: 2 * time.Hour,
		EventID:     "100,200-205,-203",
		Level:       "error,warning",
		Provider:    []string{"Example-Provider"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, api.Close()) })

	reader := api.(*winEventLog)
	require.Equal(t, "*", reader.query)
	require.NotNil(t, reader.filter)
}

func TestNewPreservesCustomXMLQuery(t *testing.T) {
	const customQuery = `<QueryList><Query Id="0"><Select Path="ForwardedEvents">*</Select></Query></QueryList>`

	api, err := New(Config{Name: "ForwardedEvents", Query: customQuery})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, api.Close()) })

	reader := api.(*winEventLog)
	require.Equal(t, customQuery, reader.query)
	require.Nil(t, reader.filter)
}
