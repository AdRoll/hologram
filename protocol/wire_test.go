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
	"bytes"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"math/rand"
	"testing"
	"time"
)

// WimpyWriter is a hampered Writer that can only process 1 byte at a time.
type WimpyWriter struct {
	bytesWritten int
}

func (w *WimpyWriter) Write(p []byte) (n int, err error) {
	w.bytesWritten += 1
	return 1, nil
}

// CorruptingPipe subtly leaves out data as it writes in chunks,
// which should trigger the checksum warning.
type CorruptingPipe struct {
	internalBuffer *bytes.Buffer
	headerWritten  bool
}

func (cp *CorruptingPipe) Write(p []byte) (int, error) {
	// Silently change data inside of the buffer, as long as it's not the header.
	if cp.headerWritten {
		l := len(p)
		for i := 0; i < 10; i++ {
			pos := rand.Intn(l)
			p[pos] = '/'
		}
	} else {
		cp.headerWritten = true
	}

	cp.internalBuffer.Write(p)
	return len(p), nil
}

func (cp *CorruptingPipe) Read(p []byte) (int, error) {
	return cp.internalBuffer.Read(p)
}

type readWriteWrapper struct {
	io.Reader
	io.Writer
}

func ReadWriter(reader io.Reader, writer io.Writer) io.ReadWriter {
	return readWriteWrapper{reader, writer}
}

func TestWire(t *testing.T) {
	Convey("Given a scratch buffer", t, func() {
		buffer := new(bytes.Buffer)

		Convey("A message written should read back okay", func() {
			source := Message_OTHER
			msg := &Message{Source: &source, Ping: &Ping{}}

			So(Write(buffer, msg), ShouldBeNil)

			reader := bytes.NewReader(buffer.Bytes())
			readMsg, err := Read(reader)
			So(err, ShouldBeNil)
			So(readMsg, ShouldNotBeNil)
		})

		Convey("A mesage written in several parts should read back", func() {
			source := Message_OTHER
			pingType := Ping_REQUEST
			msg := &Message{Source: &source, Ping: &Ping{Type: &pingType}}

			Write(buffer, msg)

			r, w := io.Pipe()
			go func() {
				w.Write(buffer.Bytes()[0:1])
				time.Sleep(time.Duration(1) * time.Millisecond)
				w.Write(buffer.Bytes()[1:5])
				time.Sleep(time.Duration(1) * time.Millisecond)
				w.Write(buffer.Bytes()[5:])
			}()

			readMsg, err := Read(r)
			So(err, ShouldBeNil)
			So(readMsg, ShouldNotBeNil)
		})

		Convey("Partial writes should work", func() {
			source := Message_OTHER
			pingType := Ping_REQUEST
			msg := &Message{Source: &source, Ping: &Ping{Type: &pingType}}

			wimp := &WimpyWriter{}

			err := Write(wimp, msg)
			So(err, ShouldBeNil)
			So(wimp.bytesWritten, ShouldBeGreaterThan, 0)
		})

		Convey("Data corruption should be detected", func() {
			source := Message_OTHER
			pingType := Ping_REQUEST
			msg := &Message{Source: &source, Ping: &Ping{Type: &pingType}}

			cp := &CorruptingPipe{
				internalBuffer: new(bytes.Buffer),
				headerWritten:  false,
			}

			err := Write(cp, msg)
			So(err, ShouldBeNil)

			// Trying to read this should fail.
			rMsg, err := Read(cp)
			So(err, ShouldEqual, ErrCorruptedMessage)
			So(rMsg, ShouldBeNil)
		})

		Convey("Massive messages should throw an error", func() {
			source := Message_OTHER
			pingType := Ping_REQUEST
			msg := &Message{Source: &source, Ping: &Ping{Type: &pingType}}

			Write(buffer, msg)
			bufferBytes := buffer.Bytes()
			bufferBytes[3] = 128 // massive image

			msg, err := Read(bytes.NewReader(bufferBytes))
			So(err, ShouldNotBeNil)
			So(msg, ShouldBeNil)
		})
	})

	Convey("Test Channelize", t, func() {
		r, w := io.Pipe()

		rw := ReadWriter(r, w)

		receive, send, errors := Channelize(rw)

		Convey("Writing a messages should see it come out the other side", func() {
			msg := &Message{ServerRequest: &ServerRequest{}}
			send <- msg

			receivedMsg := <-receive

			So(receivedMsg.GetServerRequest(), ShouldNotBeNil)

			var err error
			select {
			case err = <-errors:
				break
			default:
				break
			}
			So(err, ShouldBeNil)
		})
	})
}
