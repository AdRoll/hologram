// Hologram auth server.
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
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/server"
	"github.com/AdRoll/hologram/transport/remote"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/sts"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
)

func main() {
	// Parse command-line flags for this system.
	var (
		listenAddress    = flag.String("addr", "", "Address to listen to incoming requests on.")
		ldapAddress      = flag.String("ldapAddr", "", "Address to connect to LDAP.")
		ldapBindDN       = flag.String("ldapBindDN", "", "LDAP DN to bind to for login.")
		ldapInsecure     = flag.Bool("insecureLDAP", false, "INSECURE: Don't use TLS for LDAP connection.")
		ldapBindPassword = flag.String("ldapBindPassword", "", "LDAP password for bind.")
		statsdHost       = flag.String("stats", "", "Address to send statsd metrics to.")
		iamAccount       = flag.String("account", "", "AWS Account ID for generating IAM Role ARNs")
		defaultRole      = flag.String("role", "", "AWS role to assume by default.")
		configFile       = flag.String("conf", "/etc/hologram/server.json", "Config file to load.")
		debugMode        = flag.Bool("debug", false, "Enable debug mode.")
		config           Config
	)

	flag.Parse()

	// Enable debug log output if the user requested it.
	if *debugMode {
		log.DebugMode(true)
		log.Debug("Enabling debug log output. Use sparingly.")
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

	// Merge in command flag options.
	if *ldapAddress != "" {
		config.LDAP.Host = *ldapAddress
	}

	if *ldapInsecure {
		config.LDAP.InsecureLDAP = true
	}

	if *ldapBindDN != "" {
		config.LDAP.Bind.DN = *ldapBindDN
	}

	if *ldapBindPassword != "" {
		config.LDAP.Bind.Password = *ldapBindPassword
	}

	if *statsdHost != "" {
		config.Stats = *statsdHost
	}

	if *iamAccount != "" {
		config.AWS.Account = *iamAccount
	}

	if *listenAddress != "" {
		config.Listen = *listenAddress
	}

	if *defaultRole != "" {
		config.AWS.DefaultRole = *defaultRole
	}

	var stats g2s.Statter
	var statsErr error

	if config.LDAP.UserAttr == "" {
		config.LDAP.UserAttr = "cn"
	}

	if config.Stats == "" {
		log.Debug("No statsd server specified; no metrics will be emitted by this program.")
		stats = g2s.Noop()
	} else {
		stats, statsErr = g2s.Dial("udp", config.Stats)
		if statsErr != nil {
			log.Errorf("Error connecting to statsd: %s. No metrics will be emitted by this program.", statsErr.Error())
			stats = g2s.Noop()
		} else {
			log.Debug("This program will emit metrics to %s", config.Stats)
		}
	}

	// Setup the server state machine that responds to requests.
	auth, err := aws.GetAuth(os.Getenv("HOLOGRAM_AWSKEY"), os.Getenv("HOLOGRAM_AWSSECRET"), "", time.Now())
	if err != nil {
		log.Errorf("Error getting instance credentials: %s", err.Error())
		os.Exit(1)
	}

	stsConnection := sts.New(auth, aws.Regions["us-east-1"])
	credentialsService := server.NewDirectSessionTokenService(config.AWS.Account, stsConnection)

	var ldapServer *ldap.Conn

	// Connect to the LDAP server using TLS or not depending on the config
	if config.LDAP.InsecureLDAP {
		log.Debug("Connecting to LDAP at server %s (NOT using TLS).", config.LDAP.Host)
		ldapServer, err = ldap.Dial("tcp", config.LDAP.Host)
		if err != nil {
			log.Errorf("Could not dial LDAP! %s", err.Error())
			os.Exit(1)
		}
	} else {
		// Connect to the LDAP server with sample credentials.
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}

		log.Debug("Connecting to LDAP at server %s.", config.LDAP.Host)
		ldapServer, err = ldap.DialTLS("tcp", config.LDAP.Host, tlsConfig)
		if err != nil {
			log.Errorf("Could not dial LDAP! %s", err.Error())
			os.Exit(1)
		}
	}

	if bindErr := ldapServer.Bind(config.LDAP.Bind.DN, config.LDAP.Bind.Password); bindErr != nil {
		log.Errorf("Could not bind to LDAP! %s", bindErr.Error())
		os.Exit(1)
	}

	ldapCache, err := server.NewLDAPUserCache(ldapServer, stats, config.LDAP.UserAttr, config.LDAP.BaseDN)
	if err != nil {
		log.Errorf("Top-level error in LDAPUserCache layer: %s", err.Error())
		os.Exit(1)
	}

	serverHandler := server.New(ldapCache, credentialsService, config.AWS.DefaultRole, stats, ldapServer, config.LDAP.UserAttr, config.LDAP.BaseDN)
	server, err := remote.NewServer(config.Listen, serverHandler.HandleConnection)

	// Wait for a signal from the OS to shutdown.
	terminate := make(chan os.Signal)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	// SIGUSR1 and SIGUSR2 should make Hologram enable and disable debug logging,
	// respectively.
	debugEnable := make(chan os.Signal)
	debugDisable := make(chan os.Signal)
	signal.Notify(debugEnable, syscall.SIGUSR1)
	signal.Notify(debugDisable, syscall.SIGUSR2)

	// SIGHUP should make Hologram server reload its cache of user information
	// from LDAP.
	reloadCache := make(chan os.Signal)
	signal.Notify(reloadCache, syscall.SIGHUP)

	log.Info("Hologram server is online, waiting for termination.")

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
		case <-reloadCache:
			log.Info("Force-reloading user cache.")
			ldapCache.Update()
		}
	}

	log.Info("Caught signal; shutting down now.")
	server.Close()
}
