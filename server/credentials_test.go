package server_test

import (
	"testing"

	"github.com/AdRoll/hologram/server"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildARN(t *testing.T) {
	aliases := map[string]string{
		"a1": "arn:aws:iam::1234",
		"a2": "arn:aws:iam::5432",
	}
	Convey("A role without an alias should return the default account", t, func() {
		role := server.BuildARN("rolename", "99999", &aliases)
		So(role, ShouldResemble, "arn:aws:iam::99999:role/rolename")
	})

	Convey("A role with an alias should return the alias", t, func() {
		role := server.BuildARN("a1/rolename", "99999", &aliases)
		So(role, ShouldResemble, "arn:aws:iam::1234:role/rolename")
	})

}
