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
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/server"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
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

func (d *DummyAuthenticator) Update() error { return nil }

type dummyCredentials struct{}

func (*dummyCredentials) GetSessionToken(user *server.User) (*sts.Credentials, error) {
	accessKey := "access_key"
	secretKey := "secret"
	token := "token"
	expiration := time.Now().Add(5 * time.Minute)
	return &sts.Credentials{
		AccessKeyId:     &accessKey,
		SecretAccessKey: &secretKey,
		SessionToken:    &token,
		Expiration:      &expiration,
	}, nil
}

func (*dummyCredentials) AssumeRole(user *server.User, role string, enableLDAPRoles bool) (*sts.Credentials, error) {
	accessKey := "access_key"
	secretKey := "secret"
	token := "token"
	expiration := time.Now().Add(5 * time.Minute)
	return &sts.Credentials{
		AccessKeyId:     &accessKey,
		SecretAccessKey: &secretKey,
		SessionToken:    &token,
		Expiration:      &expiration,
	}, nil
}

type DummyLDAP struct {
	username string
	password string
	sshKeys  []string
	req      *ldap.ModifyRequest
}

func (l *DummyLDAP) Search(*ldap.SearchRequest) (*ldap.SearchResult, error) {
	return &ldap.SearchResult{
		Entries: []*ldap.Entry{
			&ldap.Entry{DN: "something",
				Attributes: []*ldap.EntryAttribute{
					&ldap.EntryAttribute{
						Name:   "cn",
						Values: []string{l.username},
					},
					&ldap.EntryAttribute{
						Name:   "userPassword",
						Values: []string{l.password},
					},
					&ldap.EntryAttribute{
						Name:   "sshPublicKey",
						Values: l.sshKeys,
					},
				},
			},
		},
	}, nil
}

func (l *DummyLDAP) Modify(mr *ldap.ModifyRequest) error {
	if reflect.DeepEqual(mr, l.req) {
		l.sshKeys = []string{"test"}
	}
	return nil
}

func TestServerStateMachine(t *testing.T) {
	// This silly thing is needed for equality testing for the LDAP dummy.
	neededModifyRequest := ldap.NewModifyRequest("something")
	neededModifyRequest.Add("sshPublicKey", []string{"test"})

	Convey("Given a state machine setup with a null logger", t, func() {
		authenticator := &DummyAuthenticator{&server.User{Username: "words"}}
		ldap := &DummyLDAP{
			username: "ari.adair",
			password: "098f6bcd4621d373cade4e832627b4f6",
			sshKeys:  []string{},
			req:      neededModifyRequest,
		}
		testServer := server.New(authenticator, &dummyCredentials{}, "default", g2s.Noop(), ldap, "cn", "dc=testdn,dc=com", false, "")
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
			role := "testrole"

			msg := &protocol.Message{
				ServerRequest: &protocol.ServerRequest{
					AssumeRole: &protocol.AssumeRole{
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

		Convey("When a request to add an SSH key comes in", func() {
			user := "ari.adair"
			password := "098f6bcd4621d373cade4e832627b4f6"
			sshKey := "test"
			testMessage := &protocol.Message{
				ServerRequest: &protocol.ServerRequest{
					AddSSHkey: &protocol.AddSSHKey{
						Username:     &user,
						Passwordhash: &password,
						Sshkeybytes:  &sshKey,
					},
				},
			}

			testConnection.Write(testMessage)
			Convey("If this request is valid", func() {
				msg, err := testConnection.Read()
				if err != nil {
					t.Fatal(err)
				}

				if msg.GetSuccess() == nil {
					t.Fail()
				}
				Convey("It should add the SSH key to the user.", func() {
					So(ldap.sshKeys[0], ShouldEqual, sshKey)
					Convey("If the user tries to add the same SSH key", func() {
						testConnection.Write(testMessage)
						Convey("It should not insert the same key twice.", func() {
							So(len(ldap.sshKeys), ShouldEqual, 1)
						})
					})
				})
			})
		})
	})
}
