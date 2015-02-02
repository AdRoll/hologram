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
	"encoding/base64"
	"github.com/AdRoll/hologram/server"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh/agent"
	"math/rand"
	"net"
	"os"
	"testing"
)

/*
StubLDAPServer exists to test Hologram's LDAP integration without
requiring an actual LDAP server.
*/
type StubLDAPServer struct {
	Key string
}

func (sls *StubLDAPServer) Search(s *ldap.SearchRequest) (*ldap.SearchResult, error) {
	return &ldap.SearchResult{
		Entries: []*ldap.Entry{
			&ldap.Entry{
				Attributes: []*ldap.EntryAttribute{
					&ldap.EntryAttribute{
						Name:   "cn",
						Values: []string{"testuser"},
					},
					&ldap.EntryAttribute{
						Name:   "sshPublicKey",
						Values: []string{sls.Key},
					},
					&ldap.EntryAttribute{
						Name:   "awsAccessKey",
						Values: []string{"AKIATESTSTRING"},
					},
					&ldap.EntryAttribute{
						Name:   "awsSecretKey",
						Values: []string{"TESTSTRINGSECRET"},
					},
				},
			},
		},
	}, nil
}

func randomBytes(length int) []byte {
	buf := make([]byte, length)

	for i := 0; i < length; i++ {
		buf[i] = byte(rand.Int() % 256)
	}

	return buf
}

func TestLDAPUserCache(t *testing.T) {
	Convey("Given an LDAP user cache connected to our server", t, func() {
		// The SSH agent stuff was moved up here so that we can use it to
		// dynamically create the LDAP result object.
		sshSock := os.Getenv("SSH_AUTH_SOCK")
		if sshSock == "" {
			t.Skip()
		}

		c, err := net.Dial("unix", sshSock)
		if err != nil {
			t.Fatal(err)
		}
		agent := agent.NewClient(c)
		keys, err := agent.List()
		if err != nil {
			t.Fatal(err)
		}

		s := &StubLDAPServer{
			Key: base64.StdEncoding.EncodeToString(keys[0].Blob),
		}
		lc, err := server.NewLDAPUserCache(s, g2s.Noop())
		So(err, ShouldBeNil)
		So(lc, ShouldNotBeNil)

		Convey("It should retrieve users from LDAP", func() {
			So(lc.Users(), ShouldNotBeEmpty)
		})

		Convey("It should verify the current user positively.", func() {
			success := false

			for i := 0; i < len(keys); i++ {
				challenge := randomBytes(64)
				sig, err := agent.Sign(keys[i], challenge)
				if err != nil {
					t.Fatal(err)
				}
				verifiedUser, err := lc.Authenticate("ericallen", challenge, sig)
				success = success || (verifiedUser != nil)
			}

			So(success, ShouldEqual, true)
		})

		Convey("When a user is requested that cannot be found in the cache", func() {
			// Use an SSH key we're guaranteed to not have.
			oldKey := s.Key
			s.Key = "AAAAB3NzaC1yc2EAAAADAQABAAABAQCXeNFWaBDX89YpZupfbyfHCKRAXT58sUU/grCQWXAEeZcK+2xRmip3j8c+YfnjoV+D5u+Xl1wvaxttzvnwaK2qZ4tanlXHSHeYC9H9h1XnVmuxZuj53E8FVewpubnmyKDWmLk9CXwJ+q+DwvUEzQsny5JxEEKSJHk0pTu5gpv9dGkcF+o8f6ZAYRgtRo1sIHQNtHnxkaYcwiCBc2j2LtSTRkvpXJ59k27gIYtH0KtmQ5N4w3DQ1zw2Etz2Kio/KxuwG4DLabqGgtyhsOYDDpd3EOGjJIj6ySGJWANILhzVieijhtiyWjSxa+i8wHOvp4tcFPy9R1BS8LY76T9vTJT3"
			lc.Update()

			// Swap the key back and try verifying.
			// We should still get a result back.
			s.Key = oldKey
			success := false

			for i := 0; i < len(keys); i++ {
				challenge := randomBytes(64)
				sig, err := agent.Sign(keys[i], challenge)
				if err != nil {
					t.Fatal(err)
				}
				verifiedUser, err := lc.Authenticate("ericallen", challenge, sig)
				success = success || (verifiedUser != nil)
			}

			Convey("Then it should update LDAP again and find the user.", func() {
				So(success, ShouldEqual, true)
			})
		})

	})
}
