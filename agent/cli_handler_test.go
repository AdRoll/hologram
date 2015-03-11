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
package agent

import (
	"io"
	"testing"

	"github.com/AdRoll/hologram/protocol"
	. "github.com/smartystreets/goconvey/convey"
)

type dummyClient struct {
	callCount int
}

func (c *dummyClient) AssumeRole(user string, role string) error {
	c.callCount++
	return nil
}

func (c *dummyClient) GetUserCredentials() error {
	c.callCount++
	return nil
}

func TestCliHandler(t *testing.T) {
	Convey("AssumeRole", t, func() {
		ra := &dummyClient{}
		ch := NewCliHandler("", ra)

		conn := testConnection(ch.HandleConnection)

		user := "user"
		role := "role"
		req := &protocol.Message{
			AgentRequest: &protocol.AgentRequest{
				AssumeRole: &protocol.AssumeRole{
					User: &user,
					Role: &role,
				},
			},
		}
		conn.Write(req)

		response, err := conn.Read()
		So(err, ShouldBeNil)

		So(response.GetAgentResponse(), ShouldNotBeNil)
		So(response.GetAgentResponse().GetSuccess(), ShouldNotBeNil)

		So(ra.callCount, ShouldEqual, 1)
	})
}

func testConnection(handler protocol.ConnectionHandlerFunc) protocol.MessageReadWriteCloser {
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	serverConnection := ReadWriter(serverReader, serverWriter)
	go handler(protocol.NewMessageConnection(serverConnection))

	clientConnection := ReadWriter(clientReader, clientWriter)
	return protocol.NewMessageConnection(clientConnection)
}

/*
Define a wrapper around an io.Pipe that we can use for testing
without needing complicated SSL setup.
*/
type readWriteWrapper struct {
	io.Reader
	io.Writer
	io.Closer
}

func ReadWriter(reader io.Reader, writer io.WriteCloser) io.ReadWriteCloser {
	return readWriteWrapper{reader, writer, writer}
}
