package server_test

import (
	"errors"
	"testing"

	"github.com/AdRoll/hologram/server"
	"github.com/nmcclain/ldap"
	. "github.com/smartystreets/goconvey/convey"
)

// A server that fails after every call to Search/Modify!
type FallibleLDAPServer struct {
	underlying *StubLDAPServer
	dead bool
}

func (fls *FallibleLDAPServer) Search(s *ldap.SearchRequest) (*ldap.SearchResult, error) {
	if fls.dead {
		return nil, ldap.NewError(ldap.ErrorNetwork, errors.New("connection died in search"))
	}
	fls.dead = true
	return fls.underlying.Search(s)
}

func (fls *FallibleLDAPServer) Modify(m *ldap.ModifyRequest) error {
	if fls.dead {
		return ldap.NewError(ldap.ErrorNetwork, errors.New("connection died in modify"))
	}
	fls.dead = true
	return fls.underlying.Modify(m)
}


func TestPersistentLDAP(t *testing.T) {
	connWillFail := false

	s := &StubLDAPServer{
		Keys: []string{},
	}

	open := func() (server.LDAPImplementation, error) {
		if connWillFail {
			return nil, ldap.NewError(ldap.ErrorNetwork, errors.New("failed reconnect"))
		}
		return &FallibleLDAPServer{
			underlying: s,
			dead:       false,
		}, nil
	}

	ldapServer, err := server.NewPersistentLDAP(open)

	Convey("Given an initially working connection to an LDAP server", t, func() {
		So(err, ShouldBeNil)
		So(ldapServer, ShouldNotBeNil)

		Convey("A search should return real results", func() {
			expected, err := s.Search(nil)
			So(err, ShouldBeNil)
			actual, err := ldapServer.Search(nil)
			So(err,      ShouldBeNil)
			So(expected, ShouldResemble, actual)
		})

		Convey("A search after failing should reconnect and seamlessly return real results", func() {
			expected, err := s.Search(nil)
			So(err, ShouldBeNil)
			actual, err := ldapServer.Search(nil)
			So(err,      ShouldBeNil)
			So(expected, ShouldResemble, actual)
		})

		Convey("A search that fails to reconnect should return an error", func() {
			connWillFail = true
			res, err := ldapServer.Search(nil)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
	})

	Convey("An initially broken connection to an LDAP server should fail fast", t, func() {
		ldapServer, err = server.NewPersistentLDAP(open)
		So(err,        ShouldNotBeNil)
		So(ldapServer, ShouldBeNil)
	})
}
