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

// Hologram workstation agent
package agent

import (
	"errors"
	"fmt"
	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/transport/remote"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"strings"
	"time"
)

type CredentialsReceiver interface {
	SetCredentials(*sts.Credentials, string)
	SetClient(Client)
}

type Client interface {
	AssumeRole(role string) error
	GetUserCredentials() error
}

type client struct {
	connectionString string
	cr               CredentialsReceiver
}

type accessKeyClient struct {
	sts         *sts.STS
	iamAccount  string
	iamUsername string
	cr          CredentialsReceiver
}

func AccessKeyClient(cr CredentialsReceiver) *accessKeyClient {
	config := aws.Config{}
	sess, err := session.NewSession(&config)
	if err != nil {
		log.Errorf("Unable to load aws sdk session.  Err: %s", err)
	}
	sts := sts.New(sess)
	iamconn := iam.New(sess)
	iamUser, err := iamconn.GetUser(&iam.GetUserInput{})
	if err != nil {
		log.Errorf("Unable to get current user.  Err: %s", err)
	}
	iamAccount := strings.Split(*iamUser.User.Arn, ":")[4]
	iamUsername := iamUser.User.UserName

	c := &accessKeyClient{
		sts:         sts,
		iamAccount:  iamAccount,
		iamUsername: *iamUsername,
		cr:          cr,
	}
	if cr != nil {
		cr.SetClient(c)
	}
	return c
}

func (c *accessKeyClient) buildARN(role string) *string {
	var arn string

	if strings.HasPrefix(role, "arn:aws:iam") {
		arn = role
	} else if strings.Contains(role, ":role/") {
		arn = fmt.Sprintf("arn:aws:iam::%s", role)
	} else {
		arn = fmt.Sprintf("arn:aws:iam::%s:role/%s", c.iamAccount, role)
	}

	return &arn
}

func (c *accessKeyClient) AssumeRole(role string) error {
	durationSeconds := int64(3600)
	roleArn := c.buildARN(role)
	roleSessionName := c.iamUsername
	options := &sts.AssumeRoleInput{
		DurationSeconds: &durationSeconds,
		RoleArn:         roleArn,
		RoleSessionName: &roleSessionName,
	}

	response, err := c.sts.AssumeRole(options)
	if err != nil {
		return err
	}
	creds := &sts.Credentials{
		AccessKeyId:     response.Credentials.AccessKeyId,
		SessionToken:    response.Credentials.SessionToken,
		SecretAccessKey: response.Credentials.SecretAccessKey,
		Expiration:      response.Credentials.Expiration,
	}
	if err != nil {
		return err
	}
	c.cr.SetCredentials(creds, role)
	return nil
}

func (c *accessKeyClient) GetUserCredentials() error {
	durationSeconds := int64(3600)
	options := &sts.GetSessionTokenInput{
		DurationSeconds: &durationSeconds,
	}
	response, err := c.sts.GetSessionToken(options)
	creds := &sts.Credentials{
		AccessKeyId:     response.Credentials.AccessKeyId,
		SessionToken:    response.Credentials.SessionToken,
		SecretAccessKey: response.Credentials.SecretAccessKey,
		Expiration:      response.Credentials.Expiration,
	}
	if err != nil {
		return err
	}
	c.cr.SetCredentials(creds, "")
	return nil
}

func NewClient(connectionString string, cr CredentialsReceiver) *client {
	c := &client{
		connectionString: connectionString,
		cr:               cr,
	}
	if cr != nil {
		cr.SetClient(c)
	}
	return c
}

func (c *client) AssumeRole(role string) error {
	req := &protocol.ServerRequest{
		AssumeRole: &protocol.AssumeRole{
			Role: &role,
		},
	}

	return c.requestCredentials(req, role)
}

func (c *client) GetUserCredentials() error {
	req := &protocol.ServerRequest{
		GetUserCredentials: &protocol.GetUserCredentials{},
	}

	return c.requestCredentials(req, "")
}

func (c *client) requestCredentials(req *protocol.ServerRequest, role string) error {
	conn, err := remote.NewClient(c.connectionString)
	if err != nil {
		return err
	}

	msg := &protocol.Message{ServerRequest: req}

	err = conn.Write(msg)

	if err != nil {
		return err
	}

	for skip := 0; ; {
		msg, err = conn.Read()
		if err != nil {
			return err
		}
		if msg.GetServerResponse() != nil {
			serverResponse := msg.GetServerResponse()
			if serverResponse.GetChallenge() != nil {
				challenge := serverResponse.GetChallenge().GetChallenge()

				signature, err := SSHSign([]byte(challenge), skip)
				if err != nil {
					return err
				}
				if signature == nil {
					return errors.New("No keys worked")
				}

				msg = &protocol.Message{
					ServerRequest: &protocol.ServerRequest{
						ChallengeResponse: &protocol.SSHChallengeResponse{
							Signature: signature.Blob,
							Format:    &signature.Format,
						},
					},
				}

				err = conn.Write(msg)
				if err != nil {
					return err
				}
			} else if serverResponse.GetCredentials() != nil {
				credsResponse := serverResponse.GetCredentials()
				accessKeyId := credsResponse.GetAccessKeyId()
				sessionToken := credsResponse.GetAccessToken()
				secretAccessKey := credsResponse.GetSecretAccessKey()
				expiration := time.Unix(credsResponse.GetExpiration(), 0)

				creds := &sts.Credentials{
					AccessKeyId:     &accessKeyId,
					SessionToken:    &sessionToken,
					SecretAccessKey: &secretAccessKey,
					Expiration:      &expiration,
				}
				c.cr.SetCredentials(creds, role)
				return nil
			} else if serverResponse.GetVerificationFailure() != nil {
				// try the next key
				skip++
			} else {
				return fmt.Errorf("unexpected message from server: %v", msg)
			}
		} else if msg.GetError() != "" {
			return errors.New(msg.GetError())
		} else {
			return fmt.Errorf("unexpected message from server: %v", msg)
		}
	}
}
