package server

import (
	"errors"
	"fmt"

	"github.com/nmcclain/ldap"
)

type persistentLDAP struct {
	open     func() (LDAPImplementation, error)
	conn     LDAPImplementation
	baseDN   string
	userAttr string
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

func (pl *persistentLDAP) SearchUser(userData map[string]string) (map[string]interface{}, error) {
	sr := ldap.NewSearchRequest(
		pl.baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(%s=%s)", pl.userAttr, userData["username"]),
		[]string{"sshPublicKey", pl.userAttr, "userPassword"},
		nil)
	r, err := pl.Search(sr)
	if err != nil {
		return nil, err
	}

	if len(r.Entries) == 0 {
		return nil, errors.New(fmt.Sprintf("User %s not found!", userData["username"]))
	}

	return map[string]interface{}{
		"password":      r.Entries[0].GetAttributeValue("userPassword"),
		"sshPublicKeys": r.Entries[0].GetAttributeValues("sshPublicKey"),
	}, nil
}

func (pl *persistentLDAP) Modify(modifyRequest *ldap.ModifyRequest) error {
	if err := pl.conn.Modify(modifyRequest); err != nil && err.(*ldap.Error).ResultCode == ldap.ErrorNetwork {
		pl.Refresh()
		return pl.conn.Modify(modifyRequest)
	} else {
		return err
	}
}

func (pl *persistentLDAP) ModifyUser(data map[string]string) error {
	mr := ldap.NewModifyRequest(data["DN"])
	mr.Add("sshPublicKey", []string{data["sshPublicKey"]})
	return pl.Modify(mr)
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
