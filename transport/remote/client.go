// TCP+TLS transport for the Hologram protocol.
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
package remote

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/AdRoll/hologram/protocol"
)

/*
New returns a TLS connection that, if not overriden, has various useful
options set.
*/
func NewClient(address string) (retClient protocol.MessageReadWriteCloser, err error) {
	pool := x509.NewCertPool()
	ca, err := Asset("self-signed-ca.cert")
	if err != nil {
		return nil, err
	}

	pool.AppendCertsFromPEM(ca)

	tlsConf := &tls.Config{
		RootCAs: pool,
		// Hologram only uses TLS to ensure the credentials that go across the wire are kept secret, and since go uses
		// ECDHE by default, we actually don't care about leaking keys or authenticating either end of the connection.
		InsecureSkipVerify: true,
	}

	socket, err := tls.Dial("tcp", address, tlsConf)
	if err != nil {
		return
	}

	retClient = protocol.NewMessageConnection(socket)
	return
}
