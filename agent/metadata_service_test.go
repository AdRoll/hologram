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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/goamz/goamz/sts"
	. "github.com/smartystreets/goconvey/convey"
)

type dummyCredentialsSource struct {
	creds *sts.Credentials
	err   error
}

func (d *dummyCredentialsSource) GetCredentials() (*sts.Credentials, error) {
	return d.creds, d.err
}

func TestMetadataService(t *testing.T) {
	Convey("Given a test server", t, func() {
		testListener, err := net.ListenTCP("tcp", &net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: 0,
		})

		dummyCreds := &dummyCredentialsSource{creds: &sts.Credentials{
			AccessKeyId:     "access_key",
			SessionToken:    "token",
			SecretAccessKey: "secret",
			Expiration:      time.Date(2014, 10, 22, 12, 21, 17, 00, time.UTC)}}

		service, err := NewMetadataService(testListener, dummyCreds)

		So(err, ShouldBeNil)
		So(service, ShouldNotBeNil)

		Reset(func() {
			service.Stop()
		})

		service.Start()

		Convey("It should enumerate roles", func() {
			respBody := string(request(service.Port(), "/latest/meta-data/iam/security-credentials/"))
			So(respBody, ShouldEqual, "hologram-access")
		})

		Convey("It should return access credentials", func() {
			respBody := request(service.Port(), "/latest/meta-data/iam/security-credentials/hologram-access")
			var creds securityCredentialsResponse
			So(json.Unmarshal(respBody, &creds), ShouldBeNil)
			So(creds.Code, ShouldEqual, "Success")
			So(creds.LastUpdated, ShouldNotBeNil)
			So(creds.LastUpdated, ShouldNotBeEmpty)
			So(creds.Type, ShouldEqual, "AWS-HMAC")
			So(creds.AccessKeyId, ShouldEqual, "access_key")
			So(creds.SecretAccessKey, ShouldEqual, "secret")
			So(creds.Token, ShouldEqual, "token")
			So(creds.Expiration, ShouldEqual, "2014-10-22T12:21:17Z")
		})

		Convey("It should return a fake services list.", func() {
			respBody := string(request(service.Port(), "/latest"))
			So(respBody, ShouldEqual, "meta-data")
		})

		Convey("It should return a fake instance ID.", func() {
			respBody := string(request(service.Port(), "/latest/meta-data/instance-id"))
			So(respBody, ShouldEqual, "i-deadbeef")
		})

		Convey("It should return a fake availability zone..", func() {
			respBody := string(request(service.Port(), "/latest/meta-data/placement/availability-zone"))
			So(respBody, ShouldEqual, "us-west-2x")
		})

		Convey("It should return a fake public DNS name.", func() {
			respBody := string(request(service.Port(), "/latest/meta-data/public-hostname"))
			So(respBody, ShouldEqual, "ec2-0-0-0-0.us-west-2.compute.amazonaws.com")
		})

		Convey("It should return a 500 error if there are no credentials", func() {
			dummyCreds.creds = nil
			dummyCreds.err = errors.New("testing")
			url := fmt.Sprintf("http://localhost:%v/latest/meta-data/iam/security-credentials/hologram-access", service.Port())
			response, err := http.Get(url)
			So(err, ShouldBeNil)
			So(response.StatusCode, ShouldEqual, 500)
		})
	})
}

func request(port int, path string) []byte {
	url := fmt.Sprintf("http://localhost:%v%v", port, path)
	response, err := http.Get(url)
	So(err, ShouldBeNil)
	So(response.StatusCode, ShouldEqual, 200)
	respBodyBytes := make([]byte, response.ContentLength)
	response.Body.Read(respBodyBytes)
	return respBodyBytes
}
