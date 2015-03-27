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
	"os"

	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/transport/local"
)

type cliHandler struct {
	client  Client
	address string
}

func NewCliHandler(address string, client Client) *cliHandler {
	return &cliHandler{client: client, address: address}
}

func (h *cliHandler) Start() error {
	_, err := local.NewServer(h.address, h.HandleConnection)
	if err != nil {
		return err
	}
	// we run as root, so let others connect to the socket
	os.Chmod(h.address, 0777)
	return nil
}

func (h *cliHandler) HandleConnection(c protocol.MessageReadWriteCloser) {
	for {
		msg, err := c.Read()
		if err != nil {
			return
		}
		if msg.GetAgentRequest() != nil {
			dr := msg.GetAgentRequest()

			var (
				sshAgentSock string
				sshKeyBytes  []byte
			)

			sshAgentSock = dr.GetSshAgentSock()
			if sshAgentSock != "" {
				log.Debug("SSH_AUTH_SOCK included in this request: %s", sshAgentSock)
			}

			sshKeyBytes = dr.GetSshKeyFile()
			if sshKeyBytes != nil {
				log.Debug("SSH keyfile included in this request.")
			}

			SSHSetAgentSock(sshAgentSock, sshKeyBytes)

			if dr.GetAssumeRole() != nil {
				log.Debug("Handling AssumeRole request.")
				assumeRole := dr.GetAssumeRole()

				err := h.client.AssumeRole(assumeRole.GetUser(), assumeRole.GetRole())

				var agentResponse protocol.AgentResponse
				if err == nil {
					agentResponse.Success = &protocol.Success{}
				} else {
					log.Errorf(err.Error())
					e := err.Error()
					agentResponse.Failure = &protocol.Failure{
						ErrorMessage: &e,
					}
				}
				msg = &protocol.Message{
					AgentResponse: &agentResponse,
				}
				err = c.Write(msg)
				if err != nil {
					return
				}
			} else if dr.GetGetUserCredentials() != nil {
				log.Debug("Handling GetSessionToken request.")
				err := h.client.GetUserCredentials()

				var agentResponse protocol.AgentResponse
				if err == nil {
					agentResponse.Success = &protocol.Success{}
				} else {
					log.Errorf(err.Error())
					e := err.Error()
					agentResponse.Failure = &protocol.Failure{
						ErrorMessage: &e,
					}
				}
				msg = &protocol.Message{
					AgentResponse: &agentResponse,
				}
				err = c.Write(msg)
				if err != nil {
					return
				}
			} else {
				log.Errorf("Unexpected agent request: %s", dr)
				c.Close()
				return
			}
		} else {
			log.Errorf("Unexpected message: %s", msg)
			c.Close()
			return
		}
	}
}
