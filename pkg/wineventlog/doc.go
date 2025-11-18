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

// Package wineventlog provides a low-level wrapper around the Windows Event Log API.
//
// This package encapsulates all direct interactions with the Windows Event Log APIs,
// including:
//   - Event iteration and querying
//   - Event rendering (converting handles to structured data)
//   - Bookmark management for resuming reads
//   - Metadata caching for efficient rendering
//
// Files to extract from beats:
//   - winlogbeat/sys/wineventlog/wineventlog_windows.go  -> api.go
//   - winlogbeat/sys/wineventlog/iterator.go             -> iterator.go
//   - winlogbeat/sys/wineventlog/bookmark.go             -> bookmark.go
//   - winlogbeat/sys/wineventlog/query.go                -> query.go
//   - winlogbeat/sys/wineventlog/renderer.go             -> renderer.go
//   - winlogbeat/sys/wineventlog/format_message.go       -> format_message.go
//   - winlogbeat/sys/wineventlog/publisher_metadata.go   -> publisher_metadata.go
//   - winlogbeat/sys/wineventlog/metadata_store.go       -> metadata_store.go
//   - winlogbeat/sys/wineventlog/template.go             -> template.go
//   - winlogbeat/sys/wineventlog/stringinserts.go        -> stringinserts.go
//   - winlogbeat/sys/wineventlog/syscall_windows.go      -> syscall_windows.go
//   - winlogbeat/sys/wineventlog/zsyscall_windows.go     -> zsyscall_windows.go
package wineventlog
