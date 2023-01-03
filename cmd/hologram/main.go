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

// Hologram workstation CLI.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"

	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/transport/local"
	"github.com/mitchellh/go-homedir"
)

// Version will be linked at compile time
var Version = "Unknown - Not built using standard process"

func main() {
	flag.Parse()

	args := flag.Args()

	var err error

	if len(args) < 1 {
		fmt.Println("Usage: hologram <cmd>")
		os.Exit(1)
	}

	switch args[0] {
	case "use":
		if len(args) < 2 {
			fmt.Println("Usage: hologram use <role>")
			os.Exit(1)
		}
		err = use(args[1])
		break
	case "me":
		err = me()
		break
	case "console":
		err = launchConsole()
	case "version":
		fmt.Println(Version)
		break
	case "help":
		fmt.Println("Usage: hologram <cmd>")
		fmt.Println("Commands:")
		fmt.Println("  use <role> - Assume a role")
		fmt.Println("  me - Get credentials for the current user")
		fmt.Println("  console - Log into the AWS console via the default browser")
		fmt.Println("  version - Print version")
		fmt.Println("  help - Print this help message")
	default:
		fmt.Println("Usage: hologram use <role>")
		os.Exit(1)
	}

	if err != nil {
		log.Errorf(err.Error())
		os.Exit(1)
	}
}

func use(role string) error {
	response, err := request(&protocol.AgentRequest{
		AssumeRole: &protocol.AssumeRole{
			Role: &role,
		},
	})
	if err != nil {
		return err
	}

	if response.GetFailure() != nil {
		return fmt.Errorf(response.GetFailure().GetErrorMessage())
	}

	if response.GetSuccess() != nil {
		output := fmt.Sprintf("Successfully got credentials for role '%s'", role)
		log.Info(output)
		return nil
	}

	return fmt.Errorf("Unexpected response type: %v", response)
}

func me() error {
	response, err := request(&protocol.AgentRequest{
		GetUserCredentials: &protocol.GetUserCredentials{},
	})

	if err != nil {
		return err
	}

	if response.GetFailure() != nil {
		return fmt.Errorf("Error from server: %s", response.GetFailure().GetErrorMessage())
	}

	if response.GetSuccess() != nil {
		log.Info("Successfully loaded credentials for you")
		return nil
	}

	return fmt.Errorf("Unexpected response type: %v", response)
}

func request(req *protocol.AgentRequest) (*protocol.AgentResponse, error) {
	client, err := local.NewClient("/var/run/hologram.sock")
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to hologram socket.  Is hologram-agent running? Error: %s", err.Error())
	}

	// Try to get to the user's SSH agent, for best compatibility.
	// However, some agents are broken, so we should also try to
	// include the ssh key contents.
	sshAgentSock := os.Getenv("SSH_AUTH_SOCK")
	req.SshAgentSock = &sshAgentSock

	// Send along the raw bytes of the SSH key.
	// TODO(silversupreme): Add in logic for id_dsa, id_ecdsa, etc.
	if sshDir, homeErr := homedir.Expand("~/.ssh"); homeErr == nil {
		sshFilename := fmt.Sprintf("%s/id_rsa", sshDir)
		if sshKeyBytes, keyReadErr := ioutil.ReadFile(sshFilename); keyReadErr == nil {
			req.SshKeyFile = sshKeyBytes
		} else {
			log.Debug("Falling back on DSA key.")
			// Fallback on a user's DSA key if they have one.
			sshFilename := fmt.Sprintf("%s/id_dsa", sshDir)
			if sshKeyBytes, keyReadErr := ioutil.ReadFile(sshFilename); keyReadErr == nil {
				req.SshKeyFile = sshKeyBytes
			}
		}
	}

	msg := &protocol.Message{
		AgentRequest: req,
	}

	err = client.Write(msg)
	if err != nil {
		return nil, err
	}

	response, err := client.Read()

	if response.GetAgentResponse() == nil {
		return nil, fmt.Errorf("Unexpected response type: %v", response)
	}

	return response.GetAgentResponse(), nil
}

type HttpHologramCredentials struct {
	Code string
	LastUpdated string
	Type string
	AccessKeyId string
	SecretAccessKey string
	Token string
	Expiration string
}

type HttpAwsCredentials struct {
	SessionId string `json:"sessionId"`
	SessionKey string `json:"sessionKey"`
	SessionToken string `json:"sessionToken"`
}

type HttpFederationSigninToken struct {
	SigninToken string
}

func launchConsole() error {
	federationUrlBase := "https://signin.aws.amazon.com/federation"
	profileUrl := "http://169.254.169.254/latest/meta-data/iam/security-credentials/"
	awsConsoleUrl := "https://console.aws.amazon.com/"

	// Get the profile name from the metadata service
	response, err := http.Get(profileUrl)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	profileBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	profile := string(profileBytes)

	// Get the credentials from the metadata service
	metadataUrl := fmt.Sprintf("%v%v", profileUrl, profile)
	response, err = http.Get(metadataUrl)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("error getting credentials. Try running 'hologram me'")
	}
	metadataBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	credentials := HttpHologramCredentials{}
	err = json.Unmarshal(metadataBytes, &credentials)
	if err != nil {
		// TODO add SyntaxError handling for when hologram me hasn't run yet
		return err
	}

	// Get the federation signin token
	awsCreds := HttpAwsCredentials{
		SessionId: credentials.AccessKeyId,
		SessionKey: credentials.SecretAccessKey,
		SessionToken: credentials.Token,
	}
	awsCredsJson, err := json.Marshal(awsCreds)
	signinTokenUrl := fmt.Sprintf("%v?Action=getSigninToken&SessionDuration=43200&Session=%v", federationUrlBase, url.QueryEscape(string(awsCredsJson)))
	response, err = http.Get(signinTokenUrl)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	signinToken_bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	signinToken := HttpFederationSigninToken{}
	err = json.Unmarshal(signinToken_bytes, &signinToken)
	if err != nil {
		return err
	}

	// Get the federation login URL
	federationUrl := fmt.Sprintf("%v?Action=login&Issuer=Hologram&Destination=%v&SigninToken=%v", federationUrlBase, url.QueryEscape(awsConsoleUrl), signinToken.SigninToken)

	// Open the URL in the browser
	switch runtime.GOOS {
		case "darwin":
			err = exec.Command("open", federationUrl).Start()
	default:
		return fmt.Errorf("unsupported OS: %v", runtime.GOOS)
	}

	return err
}
