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
package protocol

//go:generate protoc --go_out=. hologram.proto

import "io"

/*
MessageReadWriteCloser implementers provide a wrapper around the Hologram
protocol that servers can use in their connection handlers.
*/
type MessageReadWriteCloser interface {
	Read() (*Message, error)
	Write(*Message) error
	Close() error
}

/*
messageConnection encapsulates the protocol-level operations
required for a connection handler to speak the Hologram protocol.
*/
type messageConnection struct {
	internalConn io.ReadWriteCloser
}

func (smc *messageConnection) Read() (*Message, error) {
	return Read(smc.internalConn)
}

func (smc *messageConnection) Write(msg *Message) error {
	return Write(smc.internalConn, msg)
}

func (smc *messageConnection) Close() error {
	return smc.internalConn.Close()
}

/*
NewmessageConnection is a convenience function to create a
properly-initialized messageConnection.
*/
func NewMessageConnection(c io.ReadWriteCloser) *messageConnection {
	return &messageConnection{
		internalConn: c,
	}
}

/*
ConnectionHandlerFunc implementers are called as goroutines by the
server when a message is received.
*/
type ConnectionHandlerFunc func(MessageReadWriteCloser)
