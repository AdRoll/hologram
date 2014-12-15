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
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"os"
	"testing"
)

func randomBytes(length int) []byte {
	buf := make([]byte, length)

	for i := 0; i < length; i++ {
		buf[i] = byte(rand.Int() % 256)
	}

	return buf
}

func TestSSH(t *testing.T) {
	Convey("Given SSH agent has been set", t, func() {
		if os.Getenv("SSH_AUTH_SOCK") == "" {
			t.Skip()
		}
		SSHSetAgentSock(os.Getenv("SSH_AUTH_SOCK"), nil)

		testBuffer := randomBytes(64)
		_, err := SSHSign(testBuffer, 0)
		if err == noKeysError {
			t.Skip()
		}

		Convey("signature should be returned without error", func() {
			buffer := randomBytes(64)
			sig, err := SSHSign(buffer, 0)
			So(err, ShouldBeNil)
			So(sig, ShouldNotBeNil)
		})

		Convey("crazy index should return no signature", func() {
			buffer := randomBytes(64)
			sig, err := SSHSign(buffer, 1000)
			So(err, ShouldBeNil)
			So(sig, ShouldBeNil)
		})
	})

	Convey("Given an SSH key included by the CLI but no agent", t, func() {
		fixtureSSHKey, err := Asset("test_ssh_key")
		So(err, ShouldBeNil)

		SSHSetAgentSock("", fixtureSSHKey)
		Convey("A signature should still be generated without needing the agent.", func() {
			buffer := randomBytes(64)
			sig, err := SSHSign(buffer, 0)
			So(err, ShouldBeNil)
			So(sig, ShouldNotBeNil)
		})

		Convey("If the signature verification fails the first time we should not retry infinitely.", func() {
			buffer := randomBytes(64)
			sig, err := SSHSign(buffer, 1)
			So(err, ShouldEqual, sshKeyError)
			So(sig, ShouldBeNil)
		})
	})
}
