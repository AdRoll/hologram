package server_test

import (
	"testing"

	"github.com/AdRoll/hologram/server"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPersistentKeysFile(t *testing.T) {
	data := `{
        "KEY1": {"username": "user1", "roles": ["role1", "role11"]},
        "KEY2": {"username": "user2", "roles": ["role2", "role22"]}
    }`

	open := func() ([]byte, error) {
		return []byte(data), nil
	}

	dump := func([]byte) error {
		return nil
	}

	Convey("Given data from keys file", t, func() {
		Convey("Content from file should be loaded correctly", func() {
			pkf := server.NewPersistentKeysFile(open, dump)
			err := pkf.Load()
			So(err, ShouldBeNil)
		})

		Convey("An existing key in file should be found", func() {
			pkf := server.NewPersistentKeysFile(open, dump)

			expected := map[string]interface{}{
				"username": "user1",
				"roles":    []interface{}{"role1", "role11"},
			}
			actual, err := pkf.Search("KEY1")
			So(err, ShouldBeNil)
			So(expected, ShouldResemble, actual)
		})

		Convey("An non existing key in file shouldn't be found", func() {
			pkf := server.NewPersistentKeysFile(open, dump)

			var expected map[string]interface{}
			actual, err := pkf.Search("MISSING_KEY")
			So(err, ShouldBeNil)
			So(expected, ShouldResemble, actual)
		})
	})
}
