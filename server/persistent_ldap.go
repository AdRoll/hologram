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
