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
	"fmt"
	"strconv"
	"time"

	"github.com/AdRoll/hologram/log"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
	"golang.org/x/crypto/ssh"
)

/*
User represents information about a user stored in the cache.
*/
type User struct {
	Username    string
	SSHKeys     []ssh.PublicKey
	Groups      []*Group
	DefaultRole string
}

type Group struct {
	ARNs    []string
	Timeout int64
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
	users           map[string]*User
	groups          map[string]*Group
	server          LDAPImplementation
	stats           g2s.Statter
	userAttr        string
	baseDN          string
	enableLDAPRoles bool
	roleAttribute   string
	defaultRole     string
	defaultRoleAttr string
	groupClassAttr  string
	pubKeysAttr     string
	roleTimeoutAttr string
	noUpdate        bool
}

/*
Update() searches LDAP for the current user set that supports
the necessary properties for Hologram.

TODO: call this at some point during verification failure so that keys that have
been recently added to LDAP work, instead of requiring a server restart.
*/
func (luc *ldapUserCache) Update() error {
	start := time.Now()
	if luc.enableLDAPRoles {
		// Search for groups and their members
		groupSearchRequest := ldap.NewSearchRequest(
			luc.baseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
			0, 0, false,
			fmt.Sprintf("(objectClass=%s)", luc.groupClassAttr),
			[]string{luc.roleAttribute, luc.roleTimeoutAttr},
			nil,
		)

		groupSearchResult, err := luc.server.Search(groupSearchRequest)
		if err != nil {
			return err
		}

		for _, entry := range groupSearchResult.Entries {
			dn := entry.DN
			ARNs := entry.GetAttributeValues(luc.roleAttribute)

			timeout := int64(3600)
			if luc.roleTimeoutAttr != "" {
				entries := entry.GetAttributeValues(luc.roleTimeoutAttr)
				if len(entries) != 0 {
					// If the attribute has been defined on the group, fetch the first
					// definition of it
					timeoutStr := entries[0]
					timeout, err = strconv.ParseInt(timeoutStr, 10, 64)
					if err != nil {
						// Default back to a 1 hour Timeout if there is a parsing error
						timeout = int64(3600)
						log.Warning("Encountered error parsing Timeout %s on group %s", timeoutStr, dn)
					}
				}
			}

			log.Debug("Adding %s to %s with Timeout %d", ARNs, dn, timeout)
			luc.groups[dn] = &Group{
				ARNs:    ARNs,
				Timeout: timeout,
			}
		}
	}

	filter := fmt.Sprintf("(%s=*)", luc.pubKeysAttr)
	searchRequest := ldap.NewSearchRequest(
		luc.baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		0, 0, false,
		filter, []string{luc.pubKeysAttr, luc.userAttr, "memberOf", luc.defaultRoleAttr},
		nil,
	)

	searchResult, err := luc.server.Search(searchRequest)
	if err != nil {
		return err
	}
	for _, entry := range searchResult.Entries {
		username := entry.GetAttributeValue(luc.userAttr)
		userKeys := []ssh.PublicKey{}
		for _, eachKey := range entry.GetAttributeValues(luc.pubKeysAttr) {
			sshKeyBytes, _ := base64.StdEncoding.DecodeString(eachKey)
			userSSHKey, err := ssh.ParsePublicKey(sshKeyBytes)
			if err != nil {
				userSSHKey, _, _, _, err = ssh.ParseAuthorizedKey([]byte(eachKey))
				if err != nil {
					log.Warning("SSH key parsing for user %s failed (key was '%s')!", username, eachKey)
					continue
				}
			}
			userKeys = append(userKeys, userSSHKey)
		}

		userDefaultRole := luc.defaultRole
		groups := []*Group{}
		if luc.enableLDAPRoles {
			userDefaultRole = entry.GetAttributeValue(luc.defaultRoleAttr)
			if userDefaultRole == "" {
				userDefaultRole = luc.defaultRole
			}
			for _, groupDN := range entry.GetAttributeValues("memberOf") {
				log.Debug(groupDN)
				groups = append(groups, luc.groups[groupDN])
			}
		}

		luc.users[username] = &User{
			SSHKeys:     userKeys,
			Username:    username,
			Groups:      groups,
			DefaultRole: userDefaultRole,
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

func (luc *ldapUserCache) Groups() map[string]*Group {
	return luc.groups
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
		if !(luc.noUpdate) {
			luc.Update()
		}
		// Retry verification.
		return luc._verify(username, challenge, sshSig)
	}
	return retUser, nil
}

/*
NewLDAPUserCache returns a properly-configured LDAP cache.
*/
func NewLDAPUserCache(server LDAPImplementation, stats g2s.Statter, userAttr string, baseDN string, enableLDAPRoles bool, roleAttribute string, defaultRole string, defaultRoleAttr string, groupClassAttr string, pubKeysAttr string, roleTimeoutAttr string, noUpdate bool) (*ldapUserCache, error) {
	retCache := &ldapUserCache{
		users:           map[string]*User{},
		groups:          map[string]*Group{},
		server:          server,
		stats:           stats,
		userAttr:        userAttr,
		baseDN:          baseDN,
		enableLDAPRoles: enableLDAPRoles,
		roleAttribute:   roleAttribute,
		defaultRole:     defaultRole,
		defaultRoleAttr: defaultRoleAttr,
		groupClassAttr:  groupClassAttr,
		pubKeysAttr:     pubKeysAttr,
		roleTimeoutAttr: roleTimeoutAttr,
		noUpdate:        noUpdate,
	}

	updateError := retCache.Update()

	// Start updating the user cache.
	return retCache, updateError
}
