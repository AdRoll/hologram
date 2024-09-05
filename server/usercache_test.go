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
	"math/rand"
	"net"
	"os"
	"testing"

	"github.com/AdRoll/hologram/server"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var testKeys = [][]byte{[]byte(`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`),
	[]byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBPAIBAAJBAMwioem7niDILlnWDIcTOJ6m2l/JiL08zfTVmGW9T2EAW95DETW4
Fcll6Az+0TJhQOZKjnKXfWZ4trwDuZp/DZkCAwEAAQJBAJzQqgMs7r+OKBU5GqyV
NnSiBrV400NUN38yqnzVneoMBJPWf7MPEZt/rXgdisC9lYVjTTn5xohj5VrtcPFN
hVECIQD8ZuBZKvNdBfPduQ1BvaSr38GZJmP1tuQozCknOHNZ/QIhAM8LnEhRjoSr
s9rDwXZ81c/ospcg9D7k/CaTp7ksCVbNAiAN+fBgX6V8ODEpzO5z/nlY3xoMTfjp
CUiXDb8VoeWZTQIhALXXVagycQBWqTzOxuBg3YyfrBKNr9Z5WHgtIJbCdWVVAiEA
rfD3bxKdXn1zeU6JPkSQimu4rrwyR5Fwrc2NQeYNCrI=
-----END RSA PRIVATE KEY-----`)}

/*
StubLDAPServer exists to test Hologram's LDAP integration without
requiring an actual LDAP server.
*/
type StubLDAPServer struct {
	Keys      []string
	OtherKeys []string
}

func (sls *StubLDAPServer) Search(s *ldap.SearchRequest) (*ldap.SearchResult, error) {
	return &ldap.SearchResult{
		Entries: []*ldap.Entry{
			&ldap.Entry{
				DN: "testdn_0",
				Attributes: []*ldap.EntryAttribute{
					&ldap.EntryAttribute{
						Name:   "cn",
						Values: []string{"testuser"},
					},
					&ldap.EntryAttribute{
						Name:   "sshPublicKey",
						Values: sls.Keys,
					},
					&ldap.EntryAttribute{
						Name:   "otherKeysAttribute",
						Values: sls.OtherKeys,
					},
					&ldap.EntryAttribute{
						Name:   "roleAttribute",
						Values: []string{"engineer"},
					},
					&ldap.EntryAttribute{
						Name:   "timeoutAttribute",
						Values: []string{"7200"},
					},
				},
			},
			&ldap.Entry{
				DN: "testdn_1",
				Attributes: []*ldap.EntryAttribute{
					&ldap.EntryAttribute{
						Name:   "cn",
						Values: []string{"testuser"},
					},
					&ldap.EntryAttribute{
						Name:   "sshPublicKey",
						Values: sls.Keys,
					},
					&ldap.EntryAttribute{
						Name:   "otherKeysAttribute",
						Values: sls.OtherKeys,
					},
					&ldap.EntryAttribute{
						Name:   "roleAttribute",
						Values: []string{"engineer"},
					},
					&ldap.EntryAttribute{
						Name:   "timeoutAttribute",
						Values: []string{"not_an_integer"},
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

		c, err := net.Dial("unix", sshSock) // ಠ_ಠ
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
		privateKey, _ := ssh.ParsePrivateKey(testKeys[0])
		testPublicKey := base64.StdEncoding.EncodeToString(privateKey.PublicKey().Marshal())

		s := &StubLDAPServer{
			Keys: []string{keyValue, testPublicKey},
		}
		lc, err := server.NewLDAPUserCache(s, g2s.Noop(), "cn", "dc=testdn,dc=com", false, "", "", "", "groupOfNames", "sshPublicKey", "", false)
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
			oldKey := s.Keys[0]
			s.Keys[0] = testPublicKey
			lc.Update()

			// Swap the key back and try verifying.
			// We should still get a result back.
			s.Keys[0] = oldKey
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

		testAuthorizedKey := string(ssh.MarshalAuthorizedKey(privateKey.PublicKey()))

		s = &StubLDAPServer{
			Keys: []string{testAuthorizedKey},
		}
		lc, err = server.NewLDAPUserCache(s, g2s.Noop(), "cn", "dc=testdn,dc=com", false, "", "", "", "groupOfNames", "sshPublicKey", "", false)
		So(err, ShouldBeNil)
		So(lc, ShouldNotBeNil)

		Convey("The usercache should understand the SSH authorized_keys format", func() {
			challenge := randomBytes(64)
			sig, err := privateKey.Sign(cryptrand.Reader, challenge)
			if err != nil {
				t.Fatal(err)
			}
			verifiedUser, err := lc.Authenticate("ericallen", challenge, sig)
			So(verifiedUser, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})

		nonAuthorizedKey := testAuthorizedKey
		otherPrivateKey, _ := ssh.ParsePrivateKey(testKeys[1])
		testAuthorizedKey = string(ssh.MarshalAuthorizedKey(otherPrivateKey.PublicKey()))

		s = &StubLDAPServer{
			Keys:      []string{nonAuthorizedKey},
			OtherKeys: []string{testAuthorizedKey},
		}
		lc, err = server.NewLDAPUserCache(s, g2s.Noop(), "cn", "dc=testdn,dc=com", false, "", "", "", "groupOfNames", "otherKeysAttribute", "", false)
		So(err, ShouldBeNil)
		So(lc, ShouldNotBeNil)

		Convey("A non-default ssh keys attribute should be respected", func() {
			Convey("A signature using a key not in the target attribute should be rejected", func() {
				challenge := randomBytes(64)
				sig, err := privateKey.Sign(cryptrand.Reader, challenge)
				if err != nil {
					t.Fatal(err)
				}
				verifiedUser, _ := lc.Authenticate("xyzzy", challenge, sig)
				So(verifiedUser, ShouldBeNil)
			})
			Convey("A signature using a key in the target attribute should be accepted", func() {
				challenge := randomBytes(64)
				sig, err := otherPrivateKey.Sign(cryptrand.Reader, challenge)
				if err != nil {
					t.Fatal(err)
				}
				verifiedUser, err := lc.Authenticate("xyzzy", challenge, sig)
				So(err, ShouldBeNil)
				So(verifiedUser, ShouldNotBeNil)
			})
		})

		s = &StubLDAPServer{
			Keys: []string{keyValue, testPublicKey},
		}
		lc, err = server.NewLDAPUserCache(s, g2s.Noop(), "cn", "dc=testdn,dc=com", false, "roleAttribute", "", "", "groupOfNames", "sshPublicKey", "timeoutAttribute", false)
		So(err, ShouldBeNil)
		So(lc, ShouldNotBeNil)

		Convey("If a role timeout attribute is set then it should be respected", func() {
			Convey("Ensure LDAP user cache is updated with a non-default role timeout", func() {
				groups := lc.Groups()
				So(len(groups), ShouldEqual, 1)
				So(len(groups["testdn_0"].ARNs), ShouldEqual, 1)
				So(groups["testdn_0"].ARNs[0], ShouldEqual, "engineer")
				So(groups["testdn_0"].Timeout, ShouldEqual, int64(7200))
			})

			Convey("Ensure that if an invalid timeout is set in LDAP the default is used", func() {
				groups := lc.Groups()
				So(len(groups["testdn_1"].ARNs), ShouldEqual, 1)
				So(groups["testdn_1"].ARNs[0], ShouldEqual, "engineer")
				So(groups["testdn_1"].Timeout, ShouldEqual, int64(3600))
			})
		})

		lc, err = server.NewLDAPUserCache(s, g2s.Noop(), "cn", "dc=testdn,dc=com", false, "roleAttribute", "", "", "groupOfNames", "sshPublicKey", "", false)
		So(err, ShouldBeNil)
		So(lc, ShouldNotBeNil)

		Convey("If no role timeout attribute is set, the default should be used.", func() {
			groups := lc.Groups()
			So(len(groups), ShouldEqual, 1)
			So(len(groups["testdn_0"].ARNs), ShouldEqual, 1)
			So(groups["testdn_0"].ARNs[0], ShouldEqual, "engineer")
			So(groups["testdn_0"].Timeout, ShouldEqual, int64(3600))
		})

	})
}
