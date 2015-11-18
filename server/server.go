// Package server implements the connection-oriented state machine for
// the Hologram centralised server.
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
package server

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/protocol"
	"github.com/goamz/goamz/sts"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
	"golang.org/x/crypto/ssh"
)

type Authenticator interface {
	Authenticate(username string, challenge []byte, sig *ssh.Signature) (user *User, err error)
}

/*
server is a wrapper for all of the connection and message
handlers that this server implements.
*/
type server struct {
	authenticator   Authenticator
	userCache       UserCache
	credentials     CredentialService
	stats           g2s.Statter
	defaultRole     string
	ldapServer      LDAPImplementation
	userAttr        string
	baseDN          string
	enableLDAPRoles bool
	defaultRoleAttr string
}

/*
ConnectionHandler is the root of the state machine created for
each socket that is opened.
*/
func (sm *server) HandleConnection(m protocol.MessageReadWriteCloser) {
	// Loop as long as we have this connection alive.
	log.Debug("Opening new connection handler.")
	for {
		recvMsg, err := m.Read()
		if err != nil {
			// EOFs are normal, so we don't want to report them as errors.
			if err.Error() != "EOF" {
				log.Errorf("Error reading data from stream: %s", err.Error())
			}
			// Right now the behaviour of this is to terminate the connection
			// when we run into an error; should it perhaps send a NAK response
			// and keep the connection open for another retry?
			break
		}

		if pingMsg := recvMsg.GetPing(); pingMsg != nil {
			sm.HandlePing(m, pingMsg)
		} else if reqMsg := recvMsg.GetServerRequest(); reqMsg != nil {
			sm.HandleServerRequest(m, reqMsg)
		}
	}
}

/*
PingHandler returns the correct response for a ping.
*/
func (sm *server) HandlePing(m protocol.MessageReadWriteCloser, p *protocol.Ping) {
	log.Debug("Handling a ping request.")
	sm.stats.Counter(1.0, "messages.ping", 1)

	pingType := protocol.Ping_RESPONSE
	pingMsg := &protocol.Message{
		Ping: &protocol.Ping{
			Type: &pingType,
		},
	}
	m.Write(pingMsg)
}

/*
HandleServerRequest handles the flow for messages that this server
accepts from clients.
*/
func (sm *server) HandleServerRequest(m protocol.MessageReadWriteCloser, r *protocol.ServerRequest) {
	if assumeRoleMsg := r.GetAssumeRole(); assumeRoleMsg != nil {
		sm.stats.Counter(1.0, "messages.assumeRole", 1)

		role := assumeRoleMsg.GetRole()

		user, err := sm.SSHChallenge(m)

		if err != nil {
			m.Close()
			return
		}

		if user != nil {
			creds, err := sm.credentials.AssumeRole(user, role, sm.enableLDAPRoles)
			if err != nil {
				// error message from Amazon, so forward that on to the client
				errStr := err.Error()
				errMsg := &protocol.Message{
					Error: &errStr,
				}
				log.Errorf("Error from AWS for AssumeRole: %s", err.Error())
				m.Write(errMsg)
				sm.stats.Counter(1.0, "errors.assumeRole", 1)

				// Attempt to use the default role to fall back
				creds, err = sm.credentials.AssumeRole(user, user.DefaultRole, sm.enableLDAPRoles)
				if err == nil {
					m.Write(makeCredsResponse(creds))
				}
				return
			}
			m.Write(makeCredsResponse(creds))
			return
		}
	} else if getUserCredentialsMsg := r.GetGetUserCredentials(); getUserCredentialsMsg != nil {
		sm.stats.Counter(1.0, "messages.getUserCredentialsMsg", 1)
		user, err := sm.SSHChallenge(m)
		if err != nil {
			log.Errorf("Error trying to handle GetUserCredentials: %s", err.Error())
			m.Close()
			return
		}

		if user != nil {
			creds, err := sm.credentials.AssumeRole(user, user.DefaultRole, sm.enableLDAPRoles)
			if err != nil {
				log.Errorf("Error trying to handle GetUserCredentials: %s", err.Error())
				// Update user cache and try again
				sm.userCache.Update()
				creds, err = sm.credentials.AssumeRole(user, user.DefaultRole, sm.enableLDAPRoles)
				if err != nil {
					errStr := fmt.Sprintf("Could not get user credentials. %s may not have been given Hologram access yet.", user.Username)
					errMsg := &protocol.Message{
						Error: &errStr,
					}
					m.Write(errMsg)
				}
				m.Close()
				return
			}
			m.Write(makeCredsResponse(creds))
			return
		}
	} else if addSSHKeyMsg := r.GetAddSSHkey(); addSSHKeyMsg != nil {
		sm.stats.Counter(1.0, "messages.addSSHKeyMsg", 1)

		// Search for the user specified in this request.
		sr := ldap.NewSearchRequest(
			sm.baseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf("(%s=%s)", sm.userAttr, addSSHKeyMsg.GetUsername()),
			[]string{"sshPublicKey", sm.userAttr, "userPassword"},
			nil)

		user, err := sm.ldapServer.Search(sr)
		if err != nil {
			log.Errorf("Error trying to handle addSSHKeyMsg: %s", err.Error())
			return
		}

		if len(user.Entries) == 0 {
			log.Errorf("User %s not found!", addSSHKeyMsg.GetUsername())
			return
		}

		// Check their password.
		password := user.Entries[0].GetAttributeValue("userPassword")
		if password != addSSHKeyMsg.GetPasswordhash() {
			log.Errorf("Provided password for user %s does not match %s!", addSSHKeyMsg.GetUsername(), password)
			return
		}

		// Check to see if this SSH key already exists.
		for _, k := range user.Entries[0].GetAttributeValues("sshPublicKey") {
			if k == addSSHKeyMsg.GetSshkeybytes() {
				log.Warning("User %s already has this SSH key. Doing nothing.", addSSHKeyMsg.GetUsername())
				successMsg := &protocol.Message{Success: &protocol.Success{}}
				m.Write(successMsg)
				return
			}
		}

		mr := ldap.NewModifyRequest(user.Entries[0].DN)
		mr.Add("sshPublicKey", []string{addSSHKeyMsg.GetSshkeybytes()})
		err = sm.ldapServer.Modify(mr)
		if err != nil {
			log.Errorf("Could not modify LDAP user: %s", err.Error())
			return
		}

		successMsg := &protocol.Message{Success: &protocol.Success{}}
		m.Write(successMsg)
		return
	}
}

/*
SSHChallenge performs the challenge-response process to authenticate a connecting client to its SSH keys.
*/
func (sm *server) SSHChallenge(m protocol.MessageReadWriteCloser) (*User, error) {
	for {
		challenge := make([]byte, 64)
		for i := 0; i < len(challenge); i++ {
			challenge[i] = byte(rand.Int() % 256)
		}

		response := &protocol.Message{
			ServerResponse: &protocol.ServerResponse{
				Challenge: &protocol.SSHChallenge{
					Challenge: challenge,
				},
			},
		}

		err := m.Write(response)
		if err != nil {
			return nil, err
		}

		challengeResponseMessage, err := m.Read()
		if err != nil {
			return nil, err
		}

		r := challengeResponseMessage.GetServerRequest()
		if r == nil {
			return nil, errors.New("not a server request")
		}
		cr := r.GetChallengeResponse()
		if cr == nil {
			return nil, errors.New("not a server request")
		}

		// Compose this into the proper format for Authenticate.
		sig := &ssh.Signature{
			Format: cr.GetFormat(),
			Blob:   cr.GetSignature(),
		}
		verifiedUser, err := sm.authenticator.Authenticate("derp", challenge, sig)
		if err != nil {
			return nil, err
		}
		if verifiedUser != nil {
			log.Debug("Verification completed for user %s!", verifiedUser.Username)
			return verifiedUser, nil
		}
		// continue around the loop, letting the client try another key
		verificationFailure := &protocol.Message{
			ServerResponse: &protocol.ServerResponse{
				VerificationFailure: &protocol.SSHVerificationFailure{},
			},
		}
		err = m.Write(verificationFailure)
		if err != nil {
			return nil, err
		}

	}
}

func makeCredsResponse(creds *sts.Credentials) *protocol.Message {
	expiration := creds.Expiration.Unix()
	credsResponse := &protocol.Message{
		ServerResponse: &protocol.ServerResponse{
			Credentials: &protocol.STSCredentials{
				AccessKeyId:     &creds.AccessKeyId,
				SecretAccessKey: &creds.SecretAccessKey,
				AccessToken:     &creds.SessionToken,
				Expiration:      &expiration,
			},
		},
	}
	return credsResponse
}

/*
New returns a server that can be used as a handler for a
MessageConnection loop.
*/
func New(userCache UserCache,
	credentials CredentialService,
	defaultRole string,
	stats g2s.Statter,
	ldapServer LDAPImplementation,
	userAttr string,
	baseDN string,
	enableLDAPRoles bool,
	defaultRoleAttr string) *server {
	return &server{
		credentials:     credentials,
		authenticator:   userCache,
		userCache:       userCache,
		defaultRole:     defaultRole,
		stats:           stats,
		ldapServer:      ldapServer,
		userAttr:        userAttr,
		baseDN:          baseDN,
		enableLDAPRoles: enableLDAPRoles,
		defaultRoleAttr: defaultRoleAttr,
	}
}
