/// Package log provides a useful logging subsystem for Hologram tools.
//
// By default, it will log INFO-level messages to the system log and standard out,
// but DEBUG-level messages can be output to these sinks as well. By defaut DEBUG
// messages are suppressed.
//
// Messages emitted to the terminal are colourised for easy visual parsing, if the
// terminal supports it. The following colours are used:
// 	* Info:			White
// 	* Warning:	Yellow
// 	* Error:		Red
// 	* Debug:		Cyan
//
// The log format is as follows:
//
// [WARNING] 06/11/2014 18:22:34Z Message text.
// [ERROR  ] 06/11/2014 18:22:56Z Time to fail.
//
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

var (
	internalLog *logMux
)

/*
Initialize some package-level setup so you can import and go.
*/
func init() {
	internalLog = NewMux()
	internalLog.Add(NewSyslogSink())
	internalLog.Add(NewColourisedTerminalSink())
}

/*
Sink implementers provide processing logic for messages emitted by
Hologram programs.
*/
type Sink interface {
	Info(message string)
	Warning(message string)
	Error(message string)
	Debug(message string)
}

/*
These functions delegate to the package-level logged automatically created
so that we have a very simple API to get started with.
*/

func Info(message string, v ...interface{}) {
	internalLog.Info(message, v...)
}

func Warning(message string, v ...interface{}) {
	internalLog.Warning(message, v...)
}

func Error(message string, v ...interface{}) {
	internalLog.Error(message, v...)
}

func Debug(message string, v ...interface{}) {
	internalLog.Debug(message, v...)
}

/*
DebugMode sets the debug mode option for this built-in logger.
*/
func DebugMode(status bool) {
	internalLog.DebugMode(status)
}
