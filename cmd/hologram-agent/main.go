// Hologram workstation agent.
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
	"flag"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/AdRoll/hologram/agent"
	"github.com/AdRoll/hologram/log"
)

var (
	dialAddress = flag.String("addr", "", "Address to connect to hologram server on.")
	debugMode   = flag.Bool("debug", false, "Enable debug mode.")
	configFile  = flag.String("conf", "/etc/hologram/agent.json", "Config file to load.")
	config      Config
)

func main() {
	flag.Parse()

	if *debugMode {
		log.DebugMode(true)
		log.Debug("Enabling debug mode. Use sparingly.")
	}

	// Parse in options from the given config file.
	log.Debug("Loading configuration from %s", *configFile)
	configContents, configErr := ioutil.ReadFile(*configFile)
	if configErr != nil {
		log.Errorf("Could not read from config file. The error was: %s", configErr.Error())
		os.Exit(1)
	}

	configParseErr := json.Unmarshal(configContents, &config)
	if configParseErr != nil {
		log.Errorf("Error in parsing config file: %s", configParseErr.Error())
		os.Exit(1)
	}

	// Resolve configuration from the file and commond-line flags.
	// Flags will always take precedence.
	if *dialAddress != "" {
		log.Debug("Using command-line remote address.")
		config.Host = *dialAddress
	}

	// Emit the final config options for debugging if requested.
	log.Debug("Final config:")
	log.Debug("Hologram server address: %s", config.Host)

	defer func() {
		log.Debug("Removing UNIX socket.")
		os.Remove("/var/run/hologram.sock")
	}()

	// Startup the HTTP server and respond to requests.
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("169.254.169.254"),
		Port: 80,
	})
	if err != nil {
		log.Errorf("Could not startup the metadata interface: %s", err)
		os.Exit(1)
	}

	credsManager := agent.NewCredentialsExpirationManager()

	mds, metadataError := agent.NewMetadataService(listener, credsManager)
	if metadataError != nil {
		log.Errorf("Could not create metadata service: %s", metadataError.Error())
		os.Exit(1)
	}
	mds.Start()

	// Create a hologram client that can be used by other services to talk to the server
	client := agent.NewClient(config.Host, credsManager)

	agentServer := agent.NewCliHandler("/var/run/hologram.sock", client)
	err = agentServer.Start()
	if err != nil {
		log.Errorf("Could not start agentServer: %s", err.Error())
		os.Exit(1)
	}

	// Wait for a graceful shutdown signal
	terminate := make(chan os.Signal)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	// SIGUSR1 and SIGUSR2 should make Hologram enable and disable debug logging,
	// respectively.
	debugEnable := make(chan os.Signal)
	debugDisable := make(chan os.Signal)
	signal.Notify(debugEnable, syscall.SIGUSR1)
	signal.Notify(debugDisable, syscall.SIGUSR2)

	log.Info("Hologram agent is online, waiting for termination.")

WaitForTermination:
	for {
		select {
		case <-terminate:
			break WaitForTermination
		case <-debugEnable:
			log.Info("Enabling debug mode.")
			log.DebugMode(true)
		case <-debugDisable:
			log.Info("Disabling debug mode.")
			log.DebugMode(false)
		}
	}

	log.Info("Caught signal; shutting down now.")
}
