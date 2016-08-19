package server_test

import (
	"sort"
	"testing"

	"github.com/AdRoll/hologram/server"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPersistentKeysFile(t *testing.T) {
	data := `{
        "KEY1": {"username": "user1", "password": "pass1", "roles": ["role1", "role11"]},
        "KEY2": {"username": "user2", "password": "pass2", "roles": ["role2", "role22"]},
        "KEY3": {"username": "user1", "password": "pass1", "roles": ["role111", "role1111"]}
    }`

	open := func() ([]byte, error) {
		return []byte(data), nil
	}

	dump := func([]byte) error {
		return nil
	}

	Convey("Given data from keys file", t, func() {
		Convey("Content from file should be loaded correctly", func() {
			pkf := server.NewPersistentKeysFile(open, dump, "username", "roles")
			err := pkf.Load()
			So(err, ShouldBeNil)
		})

		Convey("An existing key in file should be found", func() {
			pkf := server.NewPersistentKeysFile(open, dump, "username", "roles")

			keys := []string{"KEY3", "KEY1"}
			sort.Strings(keys)
			expected := map[string]interface{}{
				"username":      "user1",
				"sshPublicKeys": keys,
				"password":      "pass1",
			}
			actual, err := pkf.Search("user1")
			actualKeys := actual["sshPublicKeys"]
			sort.Strings(actualKeys.([]string))
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expected)
		})

		Convey("An non existing key in file shouldn't be found", func() {
			pkf := server.NewPersistentKeysFile(open, dump, "username", "roles")

			user, err := pkf.Search("missing user")
			So(err, ShouldNotBeNil)
			So(user, ShouldBeNil)
		})
	})
}
