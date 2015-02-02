// Package server_test implements tests for the connection state machine
// that powers the server.
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
package server_test

import (
	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/server"
	"github.com/goamz/goamz/sts"
	"github.com/peterbourgon/g2s"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
	"io"
	"testing"
	"time"
)

// Define a logger that does nothing, for these tests.
type nullLogger struct{}

func (nl *nullLogger) Write(b []byte) (n int, err error) {
	return len(b), nil
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

type DummyAuthenticator struct {
	user *server.User
}

func (d *DummyAuthenticator) Authenticate(username string, challenge []byte, sig *ssh.Signature) (user *server.User, err error) {
	return d.user, nil
}

type dummyCredentials struct{}

func (*dummyCredentials) GetSessionToken(user *server.User) (*sts.Credentials, error) {
	return &sts.Credentials{
		AccessKeyId:     "access_key",
		SecretAccessKey: "secret",
		SessionToken:    "token",
		Expiration:      time.Now().Add(5 * time.Minute),
	}, nil
}

func (*dummyCredentials) AssumeRole(user *server.User, role string) (*sts.Credentials, error) {
	return &sts.Credentials{
		AccessKeyId:     "access_key",
		SecretAccessKey: "secret",
		SessionToken:    "token",
		Expiration:      time.Now().Add(5 * time.Minute),
	}, nil
}

func TestServerStateMachine(t *testing.T) {
	Convey("Given a state machine setup with a null logger", t, func() {
		authenticator := &DummyAuthenticator{&server.User{Username: "words"}}
		testServer := server.New(authenticator, &dummyCredentials{}, "default", g2s.Noop())
		r, w := io.Pipe()

		testConnection := protocol.NewMessageConnection(ReadWriter(r, w))
		go testServer.HandleConnection(testConnection)
		Convey("When a ping message comes in", func() {
			testPing := &protocol.Message{Ping: &protocol.Ping{}}
			testConnection.Write(testPing)
			Convey("Then the server should respond with a pong response.", func() {
				recvMsg, recvErr := testConnection.Read()
				So(recvErr, ShouldBeNil)
				So(recvMsg.GetPing(), ShouldNotBeNil)
			})
		})

		Convey("After an AssumeRequest", func() {
			username := "testuser"
			role := "testrole"

			msg := &protocol.Message{
				ServerRequest: &protocol.ServerRequest{
					AssumeRole: &protocol.AssumeRole{
						User: &username,
						Role: &role,
					},
				},
			}

			testConnection.Write(msg)

			msg, err := testConnection.Read()
			if err != nil {
				t.Fatal(err)
			}

			Convey("it should challenge, then send credentials on success", func() {
				challenge := msg.GetServerResponse().GetChallenge().GetChallenge()

				So(len(challenge), ShouldEqual, 64)

				format := "test"
				sig := []byte("ssss")

				challengeResponseMsg := &protocol.Message{
					ServerRequest: &protocol.ServerRequest{
						ChallengeResponse: &protocol.SSHChallengeResponse{
							Format:    &format,
							Signature: sig,
						},
					},
				}

				testConnection.Write(challengeResponseMsg)

				credsMsg, err := testConnection.Read()
				if err != nil {
					t.Fatal(err)
				}

				So(credsMsg, ShouldNotBeNil)
				So(credsMsg.GetServerResponse(), ShouldNotBeNil)
				So(credsMsg.GetServerResponse().GetCredentials(), ShouldNotBeNil)

				creds := credsMsg.GetServerResponse().GetCredentials()
				So(creds.GetAccessKeyId(), ShouldEqual, "access_key")
				So(creds.GetSecretAccessKey(), ShouldEqual, "secret")
				So(creds.GetAccessToken(), ShouldEqual, "token")
				So(creds.GetExpiration(), ShouldBeGreaterThanOrEqualTo, time.Now().Unix())
			})

			Convey("it should then send failure message on failed key verification", func() {
				authenticator.user = nil

				challenge := msg.GetServerResponse().GetChallenge().GetChallenge()

				So(len(challenge), ShouldEqual, 64)

				format := "test"
				sig := []byte("ssss")

				challengeResponseMsg := &protocol.Message{
					ServerRequest: &protocol.ServerRequest{
						ChallengeResponse: &protocol.SSHChallengeResponse{
							Format:    &format,
							Signature: sig,
						},
					},
				}

				testConnection.Write(challengeResponseMsg)

				credsMsg, err := testConnection.Read()
				if err != nil {
					t.Fatal(err)
				}

				So(credsMsg, ShouldNotBeNil)
				So(credsMsg.GetServerResponse(), ShouldNotBeNil)
				So(credsMsg.GetServerResponse().GetVerificationFailure(), ShouldNotBeNil)
			})
		})
	})
}
