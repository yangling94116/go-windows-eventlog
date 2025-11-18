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

package eventlog

import (
	"fmt"
	"log"
)

// noopLogger is a logger that does nothing.
type noopLogger struct{}

func (noopLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (noopLogger) Info(msg string, keysAndValues ...interface{})  {}
func (noopLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (noopLogger) Error(msg string, keysAndValues ...interface{}) {}

// NoopLogger returns a logger that discards all logs.
func NoopLogger() Logger {
	return noopLogger{}
}

// stdLogger wraps the standard library logger to implement the Logger interface.
type stdLogger struct {
	logger *log.Logger
}

// NewStdLogger creates a Logger from the standard library log.Logger.
func NewStdLogger(l *log.Logger) Logger {
	if l == nil {
		l = log.Default()
	}
	return &stdLogger{logger: l}
}

func (s *stdLogger) Debug(msg string, keysAndValues ...interface{}) {
	s.logger.Printf("[DEBUG] %s %s", msg, formatKeyValues(keysAndValues...))
}

func (s *stdLogger) Info(msg string, keysAndValues ...interface{}) {
	s.logger.Printf("[INFO] %s %s", msg, formatKeyValues(keysAndValues...))
}

func (s *stdLogger) Warn(msg string, keysAndValues ...interface{}) {
	s.logger.Printf("[WARN] %s %s", msg, formatKeyValues(keysAndValues...))
}

func (s *stdLogger) Error(msg string, keysAndValues ...interface{}) {
	s.logger.Printf("[ERROR] %s %s", msg, formatKeyValues(keysAndValues...))
}

func formatKeyValues(keysAndValues ...interface{}) string {
	if len(keysAndValues) == 0 {
		return ""
	}

	var result string
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			result += fmt.Sprintf("%v=%v ", keysAndValues[i], keysAndValues[i+1])
		} else {
			result += fmt.Sprintf("%v=<missing> ", keysAndValues[i])
		}
	}
	return result
}

// noopStatusReporter is a status reporter that does nothing.
type noopStatusReporter struct{}

func (noopStatusReporter) UpdateStatus(status Status, message string) {}

// NoopStatusReporter returns a status reporter that does nothing.
func NoopStatusReporter() StatusReporter {
	return noopStatusReporter{}
}
