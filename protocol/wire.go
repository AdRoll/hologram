// Package protocol implements the wire protocol spoken by all parts
// of a running Hologram system.
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

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"

	"code.google.com/p/goprotobuf/proto"
)

// Common errors we'll run into.
var (
	ErrCorruptedMessage = errors.New("Message did not pass checksum.")
)

// We restrict messages to 1Mb in size as a precaution against a single
// message just filling up all available memory on the server.
const MaximumMessageSize uint32 = 1024 * 1024

// header comprises a structured dataset for all parties involved
// in message-passing to verify received data.
type header struct {
	// Length of the proceeding data.
	ContentLength uint32

	// crc32 checksum of the proceeding data, used for verification
	// of correct transmission.
	Checksum uint32

	// Reserved space for future protocol updates without breaking
	// existing clients.
	Reserved uint64
}

func Channelize(c io.ReadWriter) (receive chan *Message, send chan *Message, errors chan error) {
	receive = make(chan *Message)
	send = make(chan *Message)
	errors = make(chan error)

	go func() {
		for {
			msg, err := Read(c)
			if err != nil {
				errors <- err
			}
			if msg != nil {
				receive <- msg
			}
		}
	}()

	go func() {
		var msg *Message
		for {
			select {
			case msg = <-send:
				writeErr := Write(c, msg)
				if writeErr != nil {
					errors <- writeErr
				}
			}
		}
	}()

	return
}

func Read(r io.Reader) (*Message, error) {
	var incomingHeader header

	err := binary.Read(r, binary.LittleEndian, &incomingHeader)
	if err != nil {
		return nil, err
	}

	if incomingHeader.ContentLength > MaximumMessageSize {
		return nil, fmt.Errorf("message too large: requested %d bytes but max is %d", incomingHeader.ContentLength, MaximumMessageSize)
	}

	msg := new(Message)
	data := make([]byte, incomingHeader.ContentLength)

	// Keep reading so we are resilient to IP fragmentation along the path.
	for n := uint32(0); n < incomingHeader.ContentLength; {
		nRead, err := r.Read(data[n:])
		if err != nil {
			return nil, err
		}
		n += uint32(nRead)
	}

	// Checksum the incoming data so that transmission errors can be dealt with.
	check := crc32.ChecksumIEEE(data)
	if check != incomingHeader.Checksum {
		return nil, ErrCorruptedMessage
	}

	err = proto.Unmarshal(data[0:incomingHeader.ContentLength], msg)
	return msg, err
}

// Write marshals a Message into the proper on-wire format and sends it
// to the remote system.
func Write(w io.Writer, msg *Message) error {
	buf, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	// For now we just compute the length as we don't have tests.
	bufHeader := &header{
		ContentLength: uint32(len(buf)),
		Checksum:      crc32.ChecksumIEEE(buf),
		Reserved:      0,
	}

	err = binary.Write(w, binary.LittleEndian, bufHeader)
	if err != nil {
		return err
	}

	for totalWritten := 0; totalWritten < len(buf); {
		nWritten, err := w.Write(buf)
		if err != nil {
			return err
		}
		totalWritten += nWritten
	}
	return nil
}
