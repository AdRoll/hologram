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

package remote_test

import (
	"testing"

	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/transport/remote"
	. "github.com/smartystreets/goconvey/convey"
)

func testHandler(msc protocol.MessageReadWriteCloser) {
	for {
		msg, _ := msc.Read()

		if pingReq := msg.GetPing(); pingReq != nil {
			pingResp := protocol.Ping_RESPONSE
			msc.Write(&protocol.Message{
				Ping: &protocol.Ping{
					Type: &pingResp,
				},
			})
		}

	}
}

// Test that self-signed certificates work properly when the CA cert
// included.
func TestSSLWithSelfSignedRootCA(t *testing.T) {
	Convey("Given a test server with self-signed SSL certificates", t, func() {
		// TODO(silversupreme): Make this select random ports for testing.
		tlsServer, err := remote.NewServer("127.0.0.1:10101", testHandler)
		So(err, ShouldBeNil)

		Reset(func() {
			tlsServer.Close()
		})

		Convey("When a client connects and pings", func() {
			testPingClient, err := remote.NewClient("127.0.0.1:10101")
			So(err, ShouldBeNil)

			pingReq := protocol.Ping_REQUEST
			pingMsg := &protocol.Message{
				Ping: &protocol.Ping{
					Type: &pingReq,
				},
			}
			err = testPingClient.Write(pingMsg)
			So(err, ShouldBeNil)

			Convey("Then it should get a pong response", func() {
				resp, err := testPingClient.Read()
				So(err, ShouldBeNil)
				So(resp.GetPing(), ShouldNotBeNil)
			})
		})

	})
}
