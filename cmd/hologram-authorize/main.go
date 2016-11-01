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
package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/transport/remote"
	"github.com/howeyc/gopass"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Config struct {
	Host string
}

func getAgentSSHKey() string {
	// Check to see if we have an SSH agent running.
	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	d, err := net.Dial("unix", sshAuthSock)
	if err != nil {
		return ""
	}

	agent := agent.NewClient(d)

	keys, _ := agent.List()
	if len(keys) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(keys[0].Marshal())
}

func loadPubKey(file string, path string) (string, error) {
	filePath := filepath.Join(path, file)

	sshKeyBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	k, _, _, _, err := ssh.ParseAuthorizedKey(sshKeyBytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(k.Marshal()), nil
}

func getUserHomeDirSSHKey() string {
	sshDir, homeErr := homedir.Expand("~/.ssh")
	if homeErr != nil {
		return ""
	}

	// Go in order through the list until we find one key we can use
	listFiles := []string{"id_rsa.pub", "id_dsa.pub"}
	for _, file := range listFiles {
		key, err := loadPubKey(file, sshDir)
		if err != nil {
			// TODO: Probably log it?
			continue
		}
		return key
	}

	return ""
}

func loadConfig() (Config, error) {
	// Figure out which Hologram server we need.
	configContents, _ := ioutil.ReadFile("/etc/hologram/agent.json")
	var config Config

	json.Unmarshal(configContents, &config)
	if config.Host == "" || config.Host == "" {
		return config, fmt.Errorf("hologram server (host) is not set")
	}
	return config, nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading /etc/hologram/agent.json: %s\n", err)
		os.Exit(1)
	}

	c, err := remote.NewClient(config.Host)

	if err != nil {
		fmt.Printf("Error connectiong to Hologram server: %s\n", err)
		os.Exit(2)
	}

	// Prompt the user for their username and password.
	var (
		user          string
		password      string
		passwordBytes []byte
		sshKey        string
	)

	sshKey = getAgentSSHKey()
	if sshKey == "" {
		sshKey = getUserHomeDirSSHKey()
	}

	if sshKey == "" {
		fmt.Printf("Cannot find your SSH key. Aborting.\n")
		os.Exit(1)
	}

	// Try to get the user's password from the environment.
	// This is useful for automated installation processes.
	user = os.Getenv("LDAP_USER")
	if user == "" {
		fmt.Printf("LDAP Username (not email): ")
		fmt.Scanf("%s", &user)
	}
	password = os.Getenv("LDAP_PASSWORD")
	if password == "" {
		fmt.Printf("LDAP Password: ")
		passwordBytes, err = gopass.GetPasswdMasked()
		if err != nil {
			fmt.Printf("Cannot parse you LDAP passwor. Aborting. (%v)\n", err)
			os.Exit(1)
		}
		password = string(passwordBytes[:len(passwordBytes)])
	}

	// Hash the password so we don't send it in the clear.
	hasher := md5.New()
	hasher.Write([]byte(password))

	// Go through additional steps to match the format of LDAP passwords.
	password = fmt.Sprintf("{MD5}%s", base64.StdEncoding.EncodeToString(hasher.Sum(nil)))

	testMessage := &protocol.Message{
		ServerRequest: &protocol.ServerRequest{
			AddSSHkey: &protocol.AddSSHKey{
				Username:     &user,
				Passwordhash: &password,
				Sshkeybytes:  &sshKey,
			},
		},
	}

	c.Write(testMessage)
}
