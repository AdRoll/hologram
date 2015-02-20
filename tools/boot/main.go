// Script to launch "hologram me" as soon as the network/server are ready
//
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
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"time"
)

const configFile = "/etc/hologram/agent.json"

type Config struct {
	Host string `json:"host"`
}

func main() {
	var config Config
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(contents, &config)
	if err != nil {
		log.Fatal(err)
	}

	for {
		time.Sleep(1 * time.Second)
		_, err := net.Dial("tcp", config.Host)
		if err != nil {
			// TODO: Better error handling. Exponential backoff if server is truly down
			log.Println("Error connecting to server", err)
			continue
		}

		log.Println("Booting hologram...")
		cmd := exec.Command("/usr/bin/hologram", "me")
		err = cmd.Run()
		if err != nil {
			log.Fatal("Error when starting up hologram", err)
		}
		break
	}
}
