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
	"github.com/SemanticSugar/hologram/protocol"
	"github.com/SemanticSugar/hologram/transport/remote"
	"github.com/goamz/goamz/sts"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
)

type nullLogger struct{}

func (nl *nullLogger) Write(b []byte) (n int, err error) {
	return len(b), nil
}

type dummyCredentialsReceiver struct {
	creds *sts.Credentials
}

func (r *dummyCredentialsReceiver) SetCredentials(creds *sts.Credentials, user string, role string) {
	r.creds = creds
}

func (r *dummyCredentialsReceiver) SetClient(Client) {}

func TestAssumeRole(t *testing.T) {
	fixtureSSHKey, _ := Asset("test_ssh_key")
	SSHSetAgentSock(os.Getenv("SSH_AUTH_SOCK"), fixtureSSHKey)

	Convey("AssumeRole", t, func() {
		// TODO randomize port number
		server, err := remote.NewServer("127.0.0.1:3101", DummyServer)
		if err != nil {
			t.Fatal(err)
		}

		credentialsReceiver := &dummyCredentialsReceiver{}

		c := NewClient("127.0.0.1:3101", credentialsReceiver)

		Reset(func() {
			server.Close()
		})

		err = c.AssumeRole("test_user", "test_role")

		So(err, ShouldBeNil)
		So(credentialsReceiver.creds, ShouldNotBeNil)
	})
}

func DummyServer(c protocol.MessageReadWriteCloser) {
	for {
		msg, err := c.Read()
		if err != nil {
			return
		}

		if msg.GetServerRequest() != nil {
			serverRequest := msg.GetServerRequest()

			accessKey := "access"
			secret := "secret"
			token := "token"
			exp := int64(0)

			if serverRequest.GetAssumeRole() != nil {
				challenge := &protocol.Message{
					ServerResponse: &protocol.ServerResponse{
						Challenge: &protocol.SSHChallenge{
							Challenge: []byte("foo"),
						},
					},
				}
				err = c.Write(challenge)
			} else if serverRequest.GetChallengeResponse() != nil {
				creds := &protocol.Message{
					ServerResponse: &protocol.ServerResponse{
						Credentials: &protocol.STSCredentials{
							AccessKeyId:     &accessKey,
							SecretAccessKey: &secret,
							AccessToken:     &token,
							Expiration:      &exp,
						},
					},
				}
				err = c.Write(creds)
			}
		}
	}
}
