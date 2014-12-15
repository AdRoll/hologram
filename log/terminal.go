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
	"github.com/aybabtme/rgbterm"
	"time"
)

/*
terminalSink reports log messages at the specified level from Hologram.
*/
type terminalSink struct{}

/*
Return a working logger that colourises output to the terminal according to level.
*/
func NewColourisedTerminalSink() *terminalSink {
	return &terminalSink{}
}

func (ss *terminalSink) Info(message string) {
	messageTime := time.Now().Format(time.RFC3339)
	leveledMessage := fmt.Sprintf("[INFO   ] %s %s", messageTime, message)
	fmt.Println(leveledMessage)
}

func (ss *terminalSink) Debug(message string) {
	messageTime := time.Now().Format(time.RFC3339)
	leveledMessage := fmt.Sprintf("[DEBUG  ] %s %s", messageTime, message)
	colouredMessage := rgbterm.String(leveledMessage, 42, 161, 152)
	fmt.Println(colouredMessage)
}

func (ss *terminalSink) Warning(message string) {
	messageTime := time.Now().Format(time.RFC3339)
	leveledMessage := fmt.Sprintf("[WARNING] %s %s", messageTime, message)
	colouredMessage := rgbterm.String(leveledMessage, 181, 137, 0)
	fmt.Println(colouredMessage)
}

func (ss *terminalSink) Error(message string) {
	messageTime := time.Now().Format(time.RFC3339)
	leveledMessage := fmt.Sprintf("[ERROR  ] %s %s", messageTime, message)
	colouredMessage := rgbterm.String(leveledMessage, 220, 50, 47)
	fmt.Println(colouredMessage)
}
