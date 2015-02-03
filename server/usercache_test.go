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
	cryptrand "crypto/rand"
	"encoding/base64"
	"github.com/AdRoll/hologram/server"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"math/rand"
	"net"
	"os"
	"testing"
)

// This key is pretty much guaranteed to be unique.
var testKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAsLS8C5biZsLZdZ50bPoWt5uc80wCjNEGmzS3vDYNrjO5Fuwv
+jCpV7SaITyWaHyKExsC1iegFS0lCY/cxW8sKtYd+EA1p86v28bt8T68CuSKMsfN
tU45IX69Fc/Pe8KoriToBPffYXUOEcJIrDGLZ5pDg1Oyl0DEPBuv4/BXRca/+z2x
VmqreGNsTG1HlF8FIHXagOj9KpmLIqg3NlA+Qpx6NIqv1jioQgLSUpdJx7Saaw/e
2jWhtiuYBVjps7/Hq8utP0FWdHTW04kYxAV/2/9ou9tXcQx4tPDUsxcqggkBJN5n
h8LgYhtaB/vNX+GEWvn+A3LXVJWaQtPElDIZKQIDAQABAoIBAA6N9Gcn+GHqbqrn
cEOBndllsdnASv16QgcKoo+YDCxrCjW/InyDAY+9ymwuZ10X1O+Z6/Pjs6XK4CAX
f2GrtIGavUEzWLgHqCh8DCEwv6BODqv8FQ937/C4Va60PSy+bdJaK9os6HNIhu4j
iITWV9sis6jfffhDV2Z0CVrG8wlGIGWQD/VR3pRTHwZc1Nqkk7uO3Z8ljxFrrN0R
Ptr+C5TQ0SaqPmLFCOexj2Y0uqNVnbuX/qWbga+QDLOkqTlNyMEHpTb1kkT8OMR9
fFd/H41y4rNpIQ433anUXzeW9SPrah4weCAiLnnB9sk08USSLNAHwtRyRT/EDabI
l9wztTkCgYEA5xh49Kla+ek/GN6xElybXvg/4KBc2CNVN1uKA3hX7aqsCMg9GQHS
/Jh7cgmp0X+nHQ3yC7JKNn+hZ+cpQjI+Dvgs6qTUCObT9U1ASCHwM0yuaAt2zWVS
fsXok/eVEI4YW9/P9lUcSoTKNXc8rlzQ3ZJ/tYK44QIiQln4W0o3XTcCgYEAw7+/
QijA+REZu8m/FyziLES7eAbn2rbkzMVnzzCZ0StRMQtlKC/D3zneTTsrFRC+PUhb
d0Pkn4T+RZGNzJgIgCHpZT4BfsEDiWAMQuF35KwJASY0VVcWpnrvKa+MgBCO8sfr
5uDH2U5DLJJ3fKbltrVKcCPpNj/MVxxxn4FkbJ8CgYEAvyspdAtc7PucbLBbfrsI
9GkcPm+qHkosRlz9MJ2u7zaOlb0/fZ5asQZaqB2CU4Hr9kcBAdf9OFQga1l4cgAq
AiwezASKOsrocDX1hTY+A9HdPMivAH5e3exN14mp0EYbtHTTDg2eF679r3jxw7OY
PJLh/n8i/U/Mk2Ll5m7gmcUCgYEAm61ulVZGCo9gEOolMHBAvAY5tf5//IDCPFyu
76duXVz+6GtwmuJJ+8lRE8j/vXQgaCqYm6SCOZ+SfY+B33n2ILlXnm4O0Fj+0A10
Euiv6kwrqR9SNaDaYbKZbGSx79O7bDg1U9vm9Nr6L4OYxakSPhm2RrM4sS1R/OGh
N8K3NG8CgYEArYm0fGucWB54qapCZ8FCqXWSTaYGR3oKtVQnEixgJJlg0oKl0E/X
vKCUz2qQ/gPmrh7TVYOVuLnR6sPe6TxCIwKLJVkvBuzBo83NNzpLcCrOJsGlOwh2
1JQOc8liilr0P0ajbnBR7h2g3Pr/hoNC2UyU5nUBwvOUaQfZeDtjzbs=
-----END RSA PRIVATE KEY-----`)

/*
StubLDAPServer exists to test Hologram's LDAP integration without
requiring an actual LDAP server.
*/
type StubLDAPServer struct {
	Key  string
	Key2 string
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
						Values: []string{sls.Key, sls.Key2},
					},
				},
			},
		},
	}, nil
}

func (*StubLDAPServer) Modify(*ldap.ModifyRequest) error {
	return nil
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

		keyValue := base64.StdEncoding.EncodeToString(keys[0].Blob)

		// Load in an additional key from the test data.
		privateKey, _ := ssh.ParsePrivateKey(testKey)
		testPublicKey := base64.StdEncoding.EncodeToString(privateKey.PublicKey().Marshal())

		s := &StubLDAPServer{
			Key:  keyValue,
			Key2: testPublicKey,
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
			s.Key = testPublicKey
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

		Convey("When a user with multiple SSH keys assigned tries to use Hologram", func() {
			Convey("The system should allow them to use any key.", func() {
				success := false

				for i := 0; i < len(keys); i++ {
					challenge := randomBytes(64)
					sig, err := privateKey.Sign(cryptrand.Reader, challenge)
					if err != nil {
						t.Fatal(err)
					}
					verifiedUser, err := lc.Authenticate("ericallen", challenge, sig)
					success = success || (verifiedUser != nil)
				}

				So(success, ShouldEqual, true)

			})
		})

	})
}
