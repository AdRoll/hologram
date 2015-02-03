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
	"encoding/base64"
	"github.com/AdRoll/hologram/log"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
	"golang.org/x/crypto/ssh"
	"time"
)

/*
User represents information about a user stored in the cache.
*/
type User struct {
	Username string
	SSHKeys  []ssh.PublicKey
}

/*
UserCache implementers provide information about registered users.
*/
type UserCache interface {
	// They also need to implement the SSH key verification interface.
	Authenticator
	Update() error
}

/*
LDAPImplementation implementers provide access to LDAP servers for
operations that Hologram uses.
This interface exists for testing purposes.
*/
type LDAPImplementation interface {
	Search(*ldap.SearchRequest) (*ldap.SearchResult, error)
	Modify(*ldap.ModifyRequest) error
}

/*
ldapUserCache connects to LDAP and pulls user settings from it.
*/
type ldapUserCache struct {
	users  map[string]*User
	server LDAPImplementation
	stats  g2s.Statter
}

/*
Update() searches LDAP for the current user set that supports
the necessary properties for Hologram.

TODO: call this at some point during verification failure so that keys that have
been recently added to LDAP work, instead of requiring a server restart.
*/
func (luc *ldapUserCache) Update() error {
	start := time.Now()
	filter := "(sshPublicKey=*)"
	searchRequest := ldap.NewSearchRequest(
		"dc=keeponprovoking,dc=com",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		0, 0, false,
		filter, []string{"sshPublicKey", "cn"},
		nil,
	)

	searchResult, err := luc.server.Search(searchRequest)
	if err != nil {
		return err
	}

	for _, entry := range searchResult.Entries {
		username := entry.GetAttributeValue("cn")
		userKeys := []ssh.PublicKey{}
		for _, eachKey := range entry.GetAttributeValues("sshPublicKey") {
			sshKeyBytes, _ := base64.StdEncoding.DecodeString(eachKey)
			userSSHKey, err := ssh.ParsePublicKey(sshKeyBytes)
			if err != nil {
				log.Error("SSH key parsing for %s failed! This key will not be added into LDAP.", username)
				continue
			}

			userKeys = append(userKeys, userSSHKey)
		}

		luc.users[username] = &User{
			SSHKeys:  userKeys,
			Username: username,
		}

		log.Debug("Information on %s (re-)generated.", username)
	}

	log.Debug("LDAP information re-cached.")
	luc.stats.Timing(1.0, "ldapCacheUpdate", time.Since(start))
	return nil
}

func (luc *ldapUserCache) Users() map[string]*User {
	return luc.users
}

func (luc *ldapUserCache) _verify(username string, challenge []byte, sshSig *ssh.Signature) (
	*User, error) {
	for _, user := range luc.users {
		for _, key := range user.SSHKeys {
			verifyErr := key.Verify(challenge, sshSig)
			if verifyErr == nil {
				return user, nil
			}
		}
	}

	return nil, nil
}

/*

*/
func (luc *ldapUserCache) Authenticate(username string, challenge []byte, sshSig *ssh.Signature) (
	*User, error) {
	// Loop through all of the keys and attempt verification.
	retUser, _ := luc._verify(username, challenge, sshSig)

	if retUser == nil {
		log.Debug("Could not find %s in the LDAP cache; updating from the server.", username)
		luc.stats.Counter(1.0, "ldapCacheMiss", 1)

		// We should update LDAP cache again to retry keys.
		luc.Update()
		return luc._verify(username, challenge, sshSig)
	} else {
		return retUser, nil
	}
	return nil, nil
}

/*
NewLDAPUserCache returns a properly-configured LDAP cache.
*/
func NewLDAPUserCache(server LDAPImplementation, stats g2s.Statter) (*ldapUserCache, error) {
	retCache := &ldapUserCache{
		users:  map[string]*User{},
		server: server,
		stats:  stats,
	}

	updateError := retCache.Update()

	// Start updating the user cache.
	return retCache, updateError
}
