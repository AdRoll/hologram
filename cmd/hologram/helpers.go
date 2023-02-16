package main

import (
	"fmt"
	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/transport/local"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"os"
)

func request(req *protocol.AgentRequest) (*protocol.AgentResponse, error) {
	client, err := local.NewClient("/var/run/hologram.sock")
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to hologram socket.  Is hologram-agent running? Error: %s", err.Error())
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
		return nil, fmt.Errorf("unexpected response type: %v", response)
	}

	return response.GetAgentResponse(), nil
}
