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
	"github.com/nmcclain/ldap"
)

type persistentLDAP struct {
	open func() (LDAPImplementation, error)
	conn LDAPImplementation
}

func (pl *persistentLDAP) Refresh() error {
	conn, err := pl.open()
	if err != nil {
		return err
	}

	pl.conn = conn
	return nil
}

func (pl *persistentLDAP) Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error) {
	if conn, err := pl.conn.Search(searchRequest); err != nil && err.(*ldap.Error).ResultCode == ldap.ErrorNetwork {
		pl.Refresh()
		return pl.conn.Search(searchRequest)
	} else {
		return conn, err
	}
}

func (pl *persistentLDAP) Modify(modifyRequest *ldap.ModifyRequest) error {
  if err := pl.conn.Modify(modifyRequest); err != nil && err.(*ldap.Error).ResultCode == ldap.ErrorNetwork {
		pl.Refresh()
		return pl.conn.Modify(modifyRequest)
	} else {
		return err
	}
}

func NewPersistentLDAP(open func() (LDAPImplementation, error)) (LDAPImplementation, error) {
	conn, err := open()
	if err != nil {
		return nil, err
	}

	ret := &persistentLDAP{
		open: open,
		conn: conn,
	}

	return ret, nil
}
