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

// Hologram workstation agent.
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
	httpPort    = flag.Int("port", 80, "Port for metadata service to listen on")
	config      Config
)

func ensureCIDR(s string) (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(s)
	if err == nil {
		return ipNet, nil
	}
	log.Warning(err.Error())
	// Maybe an ip so we'll add /32 to make it a single ip subnet
	_, ipNet, err = net.ParseCIDR(s + "/32")
	if err == nil {
		return ipNet, nil
	}
	log.Warning(err.Error())
	return nil, err
}

func main() {
	flag.Parse()

	if *debugMode {
		log.DebugMode(true)
		log.Debug("Enabling debug mode. Use sparingly.")
	}

	// Parse in options from the given config file.
	log.Debug("Loading configuration from %s", *configFile)
	configContents, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Errorf("Error reading from config file: %s", err.Error())
		os.Exit(1)
	}
	log.Info("Config contents %v", configContents)

	configParseErr := json.Unmarshal(configContents, &config)
	if configParseErr != nil {
		log.Errorf("Error in parsing config file: %s", configParseErr.Error())
		os.Exit(1)
	}

	// Resolve configuration from the file and command-line flags.
	// Flags will always take precedence.
	if *dialAddress != "" {
		log.Debug("Using command-line remote address.")
		config.Host = *dialAddress
	}

	// Emit the final config options for debugging if requested.
	log.Debug("Hologram server address: %s", config.Host)

	// Startup the HTTP server and respond to requests.
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP("169.254.169.254"),
		Port: *httpPort,
	})
	if err != nil {
		log.Errorf("Could not start up the metadata interface: %s", err.Error())
		os.Exit(1)
	}

	credsManager := agent.NewCredentialsExpirationManager()

	extraIps := []*net.IPNet{}
	log.Info("Extra allowed addresses: %v", config.ExtraAllowedIps)
	for _, ip := range config.ExtraAllowedIps {
		ipNet, err := ensureCIDR(ip)
		if err == nil {
			extraIps = append(extraIps, ipNet)
		} else {
			log.Warning("Ignoring invalid IP: %s", ip)
		}
	}

	mds, err := agent.NewMetadataService(listener, credsManager, extraIps)
	if err != nil {
		log.Errorf("Could not create metadata service: %s", err.Error())
		os.Exit(1)
	}
	mds.Start()

	// Create a hologram client that can be used by other services to talk to the server
	var client (agent.Client)
	if config.Host != "" {
		client = agent.NewClient(config.Host, credsManager)
	} else {
		client = agent.AccessKeyClient(credsManager, &config.AccountAliases)
	}

	agentServer := agent.NewCliHandler("/var/run/hologram.sock", client)
	if err := agentServer.Start(); err != nil {
		log.Errorf("Could not start agentServer: %s", err.Error())
		os.Exit(1)
	}

	defer func() {
		log.Debug("Removing UNIX socket.")
		os.Remove("/var/run/hologram.sock")
	}()

	// Wait for a graceful shutdown signal
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool)

	// SIGUSR1 and SIGUSR2 should make Hologram enable and disable debug logging, respectively.
	debugEnable := make(chan os.Signal, 1)
	debugDisable := make(chan os.Signal, 1)
	signal.Notify(debugEnable, syscall.SIGUSR1)
	signal.Notify(debugDisable, syscall.SIGUSR2)

	log.Info("Hologram agent is online, waiting for termination.")

	// Handle termination
	go func() {
		s := <-terminate
		log.Info("Signal received by termination handler: %+v", s)
		done <- true
	}()

	// Handle dynamic settings changes)
	go func() {
		for {
			select {
			case <-debugEnable:
				log.Info("Enabling debug mode.")
				log.DebugMode(true)
			case <-debugDisable:
				log.Info("Disabling debug mode.")
				log.DebugMode(false)
			}
		}
	}()

	<-done
	log.Info("Caught signal; shutting down now.")
}
