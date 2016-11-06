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
	"log/syslog"
)

/*
syslogSink reports log messages at the specified level from Hologram.
*/
type syslogSink struct {
	writer *syslog.Writer
}

/*
Return a working logger to Syslog.
*/
func NewSyslogSink() *syslogSink {
	rs := &syslogSink{}
	rs.writer, _ = syslog.Dial("", "", syslog.LOG_EMERG, "hologram")

	return rs
}

func (ss *syslogSink) Info(message string) {
	ss.writer.Info(message)
}

func (ss *syslogSink) Debug(message string) {
	ss.writer.Debug(message)
}

func (ss *syslogSink) Warning(message string) {
	ss.writer.Warning(message)
}

func (ss *syslogSink) Error(message string) {
	ss.writer.Err(message)
}
