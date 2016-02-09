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
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/goamz/goamz/sts"
)

/*
MetadataService extends Service to include information about public
port numbers for testing purposes.
*/
type MetadataService interface {
	Service
	Port() int
}

type CredentialsSource interface {
	GetCredentials() (*sts.Credentials, error)
}

/*
metadataService is the internal implementation of the public interface.
It serves as a reference implementation of the EC2 HTTP API for workstations.
*/
type metadataService struct {
	listener net.Listener
	creds    CredentialsSource
}

func (mds *metadataService) Start() error {
	go mds.listen()
	return nil
}

/*
This actually creates the HTTP listener and blocks on it.
Spawned in the background.
*/
func (mds *metadataService) listen() {
	handler := http.NewServeMux()
	handler.HandleFunc("/latest", mds.getServices)
	handler.HandleFunc("/latest/meta-data/iam/security-credentials/", mds.enumerateRoles)
	handler.HandleFunc("/latest/meta-data/iam/security-credentials/hologram-access", mds.getCredentials)
	handler.HandleFunc("/latest/meta-data/instance-id", mds.getInstanceID)
	handler.HandleFunc("/latest/meta-data/placement/availability-zone", mds.getAvailabilityZone)
	handler.HandleFunc("/latest/meta-data/public-hostname", mds.getPublicDNS)

	err := http.Serve(mds.listener, handler)

	if err != nil {
		if strings.HasSuffix(err.Error(), "use of closed network connection") {
			// this happens when Close() is called, and it's normal
			return
		}
		panic(err)
	}
}

/*
Stops the HTTP server and closes all extant connections.
*/
func (mds *metadataService) Stop() error {
	return mds.listener.Close()
}

/*
Returns the port number currently in use by the HTTP server.
Only really used in tests.
*/
func (mds *metadataService) Port() int {
	return mds.listener.Addr().(*net.TCPAddr).Port
}

/*
Enumerates the available instance profiles on this fake instance.
Seems like Amazon only supports one.
*/
func (mds *metadataService) enumerateRoles(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hologram-access")
}

/*
Return fake data for programs that depend on data from the metadata service.

These fields are constructed to be obviously wrong and would never be found in the
production environment.
*/
func (mds *metadataService) getServices(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "fake-meta-data")
}

func (mds *metadataService) getInstanceID(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "i-deadbeef")
}

func (mds *metadataService) getAvailabilityZone(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "us-west-2x")
}

func (mds *metadataService) getPublicDNS(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ec2-0-0-0-0.us-west-2.compute.amazonaws.com")
}

/*
Returns credentials for interested clients.
*/
func (mds *metadataService) getCredentials(w http.ResponseWriter, r *http.Request) {
	creds, err := mds.creds.GetCredentials()
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())
		return
	}

	resp := &securityCredentialsResponse{
		Code:            "Success",
		LastUpdated:     time.Now().UTC().Format(time.RFC3339),
		Type:            "AWS-HMAC",
		AccessKeyId:     creds.AccessKeyId,
		SecretAccessKey: creds.SecretAccessKey,
		Token:           creds.SessionToken,
		Expiration:      creds.Expiration.UTC().Format(time.RFC3339),
	}
	respBody, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	w.Write(respBody)
}

/*
NewMetadataService returns a properly-initialized metadataService for use.
*/
func NewMetadataService(listener net.Listener, creds CredentialsSource) (MetadataService, error) {
	return &metadataService{
		listener: listener,
		creds:    creds,
	}, nil
}

/*
Structure encoded as JSON for credential clients.
*/
type securityCredentialsResponse struct {
	Code            string `json:"Code"`
	LastUpdated     string `json:"LastUpdated"`
	Type            string `json:"Type"`
	AccessKeyId     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	Token           string `json:"Token"`
	Expiration      string `json:"Expiration"`
}
