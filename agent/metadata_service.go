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

	"github.com/AdRoll/hologram/log"
	"github.com/aws/aws-sdk-go/service/sts"
)

/*
MetadataService extends Service to include information about public
port numbers for testing purposes.
*/
type MetadataService interface {
	Service
	Port() int
}

type credentialsSource interface {
	GetCredentials() (*sts.Credentials, error)
}

/*
metadataService is the internal implementation of the public interface.
It serves as a reference implementation of the EC2 HTTP API for workstations.
*/
type metadataService struct {
	listener net.Listener
	creds    credentialsSource
	allowIps []*net.IPNet
}

func (mds *metadataService) Start() error {
	go mds.listen()
	return nil
}

func makeSecure(handler func(http.ResponseWriter, *http.Request), mds *metadataService) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		allowedIP := false
		parsedIP := net.ParseIP(ip)
		for _, ipNet := range mds.allowIps {
			allowedIP = allowedIP || ipNet.Contains(parsedIP)
		}

		// Must make sure the remote ip is localhost, otherwise clients on the same network segment could
		// potentially route traffic via 169.254.169.254:80
		if !allowedIP {
			msg := fmt.Sprintf("Access denied from %s, not in the set of allowed IPs", ip)
			log.Info("Rejecting connection from ip %s", ip)
			http.Error(w, msg, http.StatusUnauthorized)
			return
		}

		// Host should always be the listener address (usually 169.254.169.254)
		expectedHost := mds.listener.Addr().String()
		// Substitute localhost for [::]
		expectedHost = strings.Replace(expectedHost, "[::]", "localhost", 1)
		// Strip the port from the listener address
		expectedHost = strings.Split(expectedHost, ":")[0]

		// Strip port number from host address
		actualHost := strings.Split(r.Host, ":")[0]
		if actualHost != expectedHost {
			msg := fmt.Sprintf("Access denied from bad host: %s", r.Host)
			fmt.Println(msg)
			http.Error(w, msg, http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}

/*
This actually creates the HTTP listener and blocks on it.
Spawned in the background.
*/
func (mds *metadataService) listen() {
	handler := http.NewServeMux()
	handler.HandleFunc("/latest", makeSecure(mds.getServices, mds))
	handler.HandleFunc("/latest/api/token", makeSecure(mds.getv2Token, mds))
	handler.HandleFunc("/latest/meta-data/iam/security-credentials/", makeSecure(mds.enumerateRoles, mds))
	handler.HandleFunc("/latest/meta-data/iam/security-credentials", makeSecure(mds.enumerateRoles, mds))
	handler.HandleFunc("/latest/meta-data/iam/security-credentials/hologram-access", makeSecure(mds.getCredentials, mds))
	handler.HandleFunc("/latest/meta-data/instance-id", makeSecure(mds.getInstanceID, mds))
	handler.HandleFunc("/latest/meta-data/placement/availability-zone", makeSecure(mds.getAvailabilityZone, mds))
	handler.HandleFunc("/latest/meta-data/public-hostname", makeSecure(mds.getPublicDNS, mds))

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
	w.Header().Set("Server", "EC2ws")
	fmt.Fprint(w, "meta-data")
}

func (mds *metadataService) getv2Token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("x-aws-ec2-metadata-token-ttl-seconds", r.Header.Get("x-aws-ec2-metadata-token-ttl-seconds"))
	fmt.Fprint(w, "AQAAAO8q4JDjNt4Nk1u6A9zFMofraQ1ZWRUQ8ppb9sWxiXEbYOSlOw==")
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
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		Token:           *creds.SessionToken,
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
func NewMetadataService(listener net.Listener, creds credentialsSource, extraAllowedIps []*net.IPNet) (MetadataService, error) {

	allowIps := []*net.IPNet{}
	if extraAllowedIps != nil {
		allowIps = extraAllowedIps
	}

	// Add default allowed nets to the list
	for _, ip := range []net.IP{net.IPv4(127, 0, 0, 1), net.IPv4(169, 254, 169, 254)} {
		ipNet := &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
		allowIps = append(allowIps, ipNet)
	}

	return &metadataService{
		listener: listener,
		creds:    creds,
		allowIps: allowIps,
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
