// Hologram workstation CLI.
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
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/SemanticSugar/hologram/log"
	"github.com/SemanticSugar/hologram/protocol"
	"github.com/SemanticSugar/hologram/transport/local"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"os"
	"os/user"
)

func main() {
	flag.Parse()

	args := flag.Args()

	var err error

	if len(args) < 1 {
		fmt.Println("Usage hologram <cmd>")
		os.Exit(1)
	}

	switch args[0] {
	case "use":
		if len(args) < 2 {
			fmt.Println("Usage: hologram use <role>")
			os.Exit(1)
		}
		err = use(args[1])
		break
	case "me":
		err = me()
		break
	default:
		fmt.Println("Usage: hologram use <role>")
		os.Exit(1)
	}

	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func use(role string) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	response, err := request(&protocol.AgentRequest{
		AssumeRole: &protocol.AssumeRole{
			User: &u.Username,
			Role: &role,
		},
	})
	if err != nil {
		return err
	}

	if response.GetFailure() != nil {
		return errors.New(fmt.Sprintf(response.GetFailure().GetErrorMessage()))
	}

	if response.GetSuccess() != nil {
		output := fmt.Sprintf("Successfully got credentials for role '%s'", role)
		log.Info(output)
		return nil
	}

	return errors.New(fmt.Sprintf("Unexpected response type: %v", response))
}

func me() error {
	response, err := request(&protocol.AgentRequest{
		GetUserCredentials: &protocol.GetUserCredentials{},
	})

	if err != nil {
		return err
	}

	if response.GetFailure() != nil {
		return errors.New(fmt.Sprintf("Error from server: %s", response.GetFailure().GetErrorMessage()))
	}

	if response.GetSuccess() != nil {
		log.Info("Successfully loaded credentials for you")
		return nil
	}

	return errors.New(fmt.Sprintf("Unexpected response type: %v", response))
}

func request(req *protocol.AgentRequest) (*protocol.AgentResponse, error) {
	client, err := local.NewClient("/var/run/hologram.sock")
	if err != nil {
		return nil, err
	}

	// Try to get to the user's SSH agent, for best compatibility.
	// However, some agents are broken, so we should also try to
	// include the ssh key contents.
	sshAgentSock := os.Getenv("SSH_AUTH_SOCK")
	req.SshAgentSock = &sshAgentSock

	// Send along the raw bytes of the SSH key.
	// TODO(silversupreme): Add in logic for id_dsa, id_ecdsa, etc.
	if sshDir, homeErr := homedir.Expand("~/.ssh"); homeErr == nil {
		sshFilename := fmt.Sprintf("%s/id_rsa", sshDir)
		if sshKeyBytes, keyReadErr := ioutil.ReadFile(sshFilename); keyReadErr == nil {
			req.SshKeyFile = sshKeyBytes
		} else {
			log.Debug("Falling back on DSA key.")
			// Fallback on a user's DSA key if they have one.
			sshFilename := fmt.Sprintf("%s/id_dsa", sshDir)
			if sshKeyBytes, keyReadErr := ioutil.ReadFile(sshFilename); keyReadErr == nil {
				req.SshKeyFile = sshKeyBytes
			}
		}
	}

	msg := &protocol.Message{
		AgentRequest: req,
	}

	err = client.Write(msg)
	if err != nil {
		return nil, err
	}

	response, err := client.Read()

	if response.GetAgentResponse() == nil {
		return nil, errors.New(fmt.Sprintf("Unexpected response type: %v", response))
	}

	return response.GetAgentResponse(), nil
}
