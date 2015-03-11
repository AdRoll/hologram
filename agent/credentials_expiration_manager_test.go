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
package agent

import (
	"testing"
	"time"

	"github.com/goamz/goamz/sts"
	. "github.com/smartystreets/goconvey/convey"
)

type dummyClient2 struct {
	assumeRoleCount         int
	getUserCredentialsCount int
}

func (d *dummyClient2) AssumeRole(user string, role string) error {
	d.assumeRoleCount += 1
	return nil
}

func (d *dummyClient2) GetUserCredentials() error {
	d.getUserCredentialsCount += 1
	return nil
}

func TestCredentialsExpirationManager(t *testing.T) {
	Convey("TestCredentialsExpirationManager", t, func() {
		c := &dummyClient2{}

		credsManager := NewCredentialsExpirationManager()
		credsManager.SetClient(c)

		Convey("Valid credentials are returned", func() {
			creds := sts.Credentials{
				AccessKeyId: "derp",
			}
			credsManager.SetCredentials(&creds, "", "")

			retrievedCreds, err := credsManager.GetCredentials()
			So(err, ShouldBeNil)
			So(retrievedCreds, ShouldEqual, &creds)
		})

		Convey("Old credentials are refreshed", func() {
			creds := sts.Credentials{
				AccessKeyId: "derp",
				Expiration:  time.Now().Add(-time.Duration(1 * time.Hour)),
			}
			credsManager.SetCredentials(&creds, "", "")

			_, err := credsManager.GetCredentials()
			So(err, ShouldBeNil)
			So(c.getUserCredentialsCount, ShouldEqual, 1)
		})

		Convey("Old credentials from AssumeRole are refreshed", func() {
			creds := sts.Credentials{
				AccessKeyId: "derp",
				Expiration:  time.Now().Add(-time.Duration(5 * time.Minute)),
			}
			credsManager.SetCredentials(&creds, "user", "role")

			_, err := credsManager.GetCredentials()
			So(err, ShouldBeNil)
			So(c.assumeRoleCount, ShouldEqual, 1)
		})
	})
}
