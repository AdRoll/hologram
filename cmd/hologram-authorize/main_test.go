package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestAuthorize(t *testing.T) {
	Convey("loadConfig errors on empty agent.json", t, func() {
		_, err := loadConfig()
		So(err, ShouldNotBeNil)
	})
}
