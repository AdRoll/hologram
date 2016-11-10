// Copyright 2014 AdRoll, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
)

/*
logMux implements a multiplexer for log messages, to fan the message out to
multiple Sinks at a time. This is done asynchronously, and no error reporting
from the underlying system is supported.
*/
type logMux struct {
	sinks     []Sink
	debugMode bool
}

func NewMux() *logMux {
	return &logMux{
		debugMode: false,
	}
}

/*
Add a sink. TODO(silversupreme): Write this comment better.
*/
func (m *logMux) Add(s Sink) {
	m.sinks = append(m.sinks, s)
}

/*
Fan-out messages to each sink.
*/
func (m *logMux) Info(message string, v ...interface{}) {
	actualMessage := fmt.Sprintf(message, v...)
	for _, sink := range m.sinks {
		sink.Info(actualMessage)
	}
}

func (m *logMux) Debug(message string, v ...interface{}) {
	if !m.debugMode {
		return
	}
	actualMessage := fmt.Sprintf(message, v...)

	for _, sink := range m.sinks {
		sink.Debug(actualMessage)
	}
}

func (m *logMux) Error(message string, v ...interface{}) {
	actualMessage := fmt.Sprintf(message, v...)

	for _, sink := range m.sinks {
		sink.Error(actualMessage)
	}
}

func (m *logMux) Warning(message string, v ...interface{}) {
	actualMessage := fmt.Sprintf(message, v...)

	for _, sink := range m.sinks {
		sink.Warning(actualMessage)
	}
}

/*
DebugMode sets whether debug logs are output.
*/
func (m *logMux) DebugMode(status bool) {
	m.debugMode = status
}
