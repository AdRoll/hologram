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

// Hologram proof of life utility.
package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"

	"github.com/AdRoll/hologram/protocol"
)

var host = flag.String("host", "localhost", "IP or hostname to ping")
var port = flag.Int("port", 3100, "Port to connect to for ping")

func main() {
	flag.Parse()
	connString := fmt.Sprintf("%s:%d", *host, *port)

	pool := x509.NewCertPool()

	tlsConf := &tls.Config{
		RootCAs: pool,
		// Hologram only uses TLS to ensure the credentials that go across the wire are kept secret, and since go uses
		// ECDHE by default, we actually don't care about leaking keys or authenticating either end of the connection.
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", connString, tlsConf)
	if err != nil {
		panic(err)
	}

	fmt.Printf("sending ping to %s...\n", connString)
	err = protocol.Write(conn, &protocol.Message{Ping: &protocol.Ping{}})
	response, err := protocol.Read(conn)

	if err != nil {
		panic(err)
	}

	if response.GetPing() != nil {
		fmt.Println("Got pong!")
	}
}
