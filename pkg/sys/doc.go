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

// Package sys provides low-level system utilities for working with Windows.
//
// This package includes:
//   - Byte buffer pooling for efficient memory management
//   - UTF-16 to UTF-8 string conversion
//   - Error handling utilities
//
// Files to extract from beats:
//   - winlogbeat/sys/buffer.go         -> buffer.go
//   - winlogbeat/sys/bufferpool.go     -> bufferpool.go
//   - winlogbeat/sys/strings.go        -> strings.go
//   - winlogbeat/sys/strings_windows.go -> strings_windows.go
//   - winlogbeat/sys/errors.go         -> errors.go
//
// Note: The UTF-16 conversion functions need to be updated to use
// golang.org/x/text/encoding/utf16 instead of libbeat dependencies.
package sys
