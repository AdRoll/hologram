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
	"errors"
	"time"

	"github.com/goamz/goamz/sts"
)

type credentialsExpirationManager struct {
	creds  *sts.Credentials
	user   string
	role   string
	client Client
}

func NewCredentialsExpirationManager() *credentialsExpirationManager {
	return &credentialsExpirationManager{}
}

func (m *credentialsExpirationManager) SetCredentials(newCreds *sts.Credentials, user string, role string) {
	m.creds = newCreds
	m.user = user
	m.role = role
}

func (m *credentialsExpirationManager) SetClient(client Client) {
	m.client = client
}

func (m *credentialsExpirationManager) GetCredentials() (*sts.Credentials, error) {
	if m.creds == nil {
		return nil, errors.New("No credentials set. Please activate hologram from the CLI first")
	}

	err := m.maybeRefreshCredentials()
	if err != nil {
		return nil, err
	}
	return m.creds, nil
}

func (m *credentialsExpirationManager) maybeRefreshCredentials() error {
	if m.client == nil {
		return errors.New("No client set for refreshing credentials")
	}
	if m.creds.Expiration.Before(time.Now()) {
		if m.role != "" {
			// and we used AssumeRole to generate the current creds
			// then use AssumeRole to refresh 'em
			return m.client.AssumeRole(m.user, m.role)
		}
		// go ahead and refresh our creds, just to be safe
		return m.client.GetUserCredentials()
	}
	return nil
}
