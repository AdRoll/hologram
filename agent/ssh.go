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
	"crypto/rand"
	"errors"
	"net"

	"github.com/AdRoll/hologram/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var (
	// Not sure if this needs a mutex around it. Probably not, because it only gets written once by one thing.
	socketAddress  string
	successfulKey  *agent.Key
	providedSshKey ssh.Signer
	noKeysError    = errors.New("No keys available in ssh-agent")
	sshKeyError    = errors.New("Could not use the provided SSH key.")
)

func SSHSetAgentSock(socketAddressFromCli string, sshKeyFromCli []byte) {
	socketAddress = socketAddressFromCli

	if sshKeyFromCli != nil {
		sshKey, keyErr := ssh.ParsePrivateKey(sshKeyFromCli)
		if keyErr != nil {
			log.Error("Could not parse SSH key given by the CLI.")
		} else {
			providedSshKey = sshKey
		}
	}
}

// SSHSign signs the provided challenge using a key from the ssh-agent keyring. The key is chosen by enumerating all
// keys, then skipping the requested number of keys.
func SSHSign(challenge []byte, skip int) (*ssh.Signature, error) {
	var signer ssh.Signer

	if socketAddress == "" {
		// Do not infinitely loop trying to use our provided SSH key.
		if skip > 0 {
			return nil, sshKeyError
		}

		log.Debug("Falling back on provided SSH key.")
		if providedSshKey == nil {
			return nil, sshKeyError
		}
		signer = providedSshKey
	} else {
		c, err := net.Dial("unix", socketAddress)
		if err != nil {
			return nil, err
		}
		agent := agent.NewClient(c)

		keys, err := agent.List()
		if err != nil {
			return nil, err
		}

		if len(keys) == 0 {
			return nil, noKeysError
		}

		if skip >= len(keys) {
			// indicate that we've tried everything and exhausted the keyring
			return nil, nil
		}

		signers, getSignersErr := agent.Signers()
		if getSignersErr != nil {
			return nil, getSignersErr
		}

		signer = signers[skip]
	}

	sig, err := signer.Sign(rand.Reader, challenge)
	return sig, err
}
