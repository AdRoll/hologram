package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/AdRoll/hologram/protocol"
	"github.com/AdRoll/hologram/transport/remote"
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

func getUserHomeDirSSHKey() string {
	if sshDir, homeErr := homedir.Expand("~/.ssh"); homeErr == nil {
		sshFilename := fmt.Sprintf("%s/id_rsa.pub", sshDir)
		if sshKeyBytes, keyReadErr := ioutil.ReadFile(sshFilename); keyReadErr == nil {
			k, _, _, _, _ := ssh.ParseAuthorizedKey(sshKeyBytes)
			return base64.StdEncoding.EncodeToString(k.Marshal())
		} else {
			// Fallback on a user's DSA key if they have one.
			sshFilename := fmt.Sprintf("%s/id_dsa.pub", sshDir)
			if sshKeyBytes, keyReadErr := ioutil.ReadFile(sshFilename); keyReadErr == nil {
				k, _, _, _, _ := ssh.ParseAuthorizedKey(sshKeyBytes)
				return base64.StdEncoding.EncodeToString(k.Marshal())
			}
		}
	}

	return ""
}

func main() {
	// Figure out which Hologram server we need.
	configContents, _ := ioutil.ReadFile("/etc/hologram/agent.json")
	var config Config

	json.Unmarshal(configContents, &config)

	c, _ := remote.NewClient(config.Host)

	// Prompt the user for their username and password.
	var (
		user     string
		password string
		sshKey   string
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
		fmt.Scanf("%s", &password)
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
