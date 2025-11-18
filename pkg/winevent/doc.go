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

// Package winevent provides data structures for representing Windows Event Log events.
//
// This package contains the core data model for Windows events, including:
//   - Event structure with all event properties
//   - Helper functions for converting events to maps
//   - SID (Security Identifier) handling
//
// Files to extract from beats:
//   - winlogbeat/sys/winevent/event.go      -> event.go
//   - winlogbeat/sys/winevent/maputil.go    -> maputil.go
//   - winlogbeat/sys/winevent/sid.go        -> sid.go
//   - winlogbeat/sys/winevent/sid_windows.go -> sid_windows.go
//   - winlogbeat/sys/winevent/winmeta.go    -> winmeta.go
package winevent
