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

// Hologram auth server.
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/server"
	"github.com/AdRoll/hologram/transport/remote"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/nmcclain/ldap"
	"github.com/peterbourgon/g2s"
)

func ConnectLDAP(conf LDAP) (*ldap.Conn, error) {
	var ldapServer *ldap.Conn
	var err error

	// Connect to the LDAP server using TLS or not depending on the config
	if conf.InsecureLDAP {
		log.Debug("Connecting to LDAP at server %s (NOT using TLS).", conf.Host)
		ldapServer, err = ldap.Dial("tcp", conf.Host)
	} else {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		log.Debug("Connecting to LDAP at server %s.", conf.Host)
		ldapServer, err = ldap.DialTLS("tcp", conf.Host, tlsConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("Could not dial LDAP! %v", err)
	}

	if err = ldapServer.Bind(conf.Bind.DN, conf.Bind.Password); err != nil {
		return nil, fmt.Errorf("Could not bind to LDAP! %v", err)
	}

	return ldapServer, nil
}

func main() {
	// Parse command-line flags for this system.
	var (
		listenAddress    = flag.String("addr", "", "Address to listen to incoming requests on.")
		ldapAddress      = flag.String("ldapAddr", "", "Address to connect to LDAP.")
		ldapBindDN       = flag.String("ldapBindDN", "", "LDAP DN to bind to for login.")
		ldapInsecure     = flag.Bool("insecureLDAP", false, "INSECURE: Don't use TLS for LDAP connection.")
		ldapBindPassword = flag.String("ldapBindPassword", "", "LDAP password for bind.")
		statsdHost       = flag.String("statsHost", "", "Address to send statsd metrics to.")
		iamAccount       = flag.String("iamaccount", "", "AWS Account ID for generating IAM Role ARNs")
		enableLDAPRoles  = flag.Bool("ldaproles", false, "Enable role support using LDAP directory.")
		roleAttribute    = flag.String("roleattribute", "", "Group attribute to get role from.")
		defaultRoleAttr  = flag.String("defaultroleattr", "", "User attribute to check to determine a user's default role.")
		groupClassAttr   = flag.String("groupclassattr", "", "LDAP objectclass to determine the groups in LDAP.")
		defaultRole      = flag.String("role", "", "AWS role to assume by default.")
		configFile       = flag.String("conf", "/etc/hologram/server.json", "Config file to load.")
		cacheTimeout     = flag.Int("cachetime", 3600, "Time in seconds after which to refresh LDAP user cache.")
		debugMode        = flag.Bool("debug", false, "Enable debug mode.")
		pubKeysAttr      = flag.String("pubkeysattr", "", "Name of the LDAP user attribute containing ssh public key data.")
		roleTimeoutAttr  = flag.String("roletimeoutattr", "", "Name of the LDAP group attribute containing role timeout in seconds.")
		noUpdateAttr     = flag.Bool("noUpdate", false, "Disable cache update on missed user.")
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

	if *enableLDAPRoles {
		config.LDAP.EnableLDAPRoles = true
	}

	if *defaultRoleAttr != "" {
		config.LDAP.DefaultRoleAttr = *defaultRoleAttr
	}

	if *roleAttribute != "" {
		config.LDAP.RoleAttribute = *roleAttribute
	}

	if *groupClassAttr != "" {
		config.LDAP.GroupClassAttr = *groupClassAttr
	} else if config.LDAP.GroupClassAttr == "" {
		config.LDAP.GroupClassAttr = "groupOfNames"
	}

	if *pubKeysAttr != "" {
		config.LDAP.PubKeysAttr = *pubKeysAttr
	} else if config.LDAP.PubKeysAttr == "" {
		config.LDAP.PubKeysAttr = "sshPublicKey"
	}

	if *roleTimeoutAttr != "" {
		config.LDAP.RoleTimeoutAttr = *roleTimeoutAttr
	} else if config.LDAP.RoleTimeoutAttr == "" {
		config.LDAP.RoleTimeoutAttr = ""
	}

	if *cacheTimeout != 3600 {
		config.CacheTimeout = *cacheTimeout
	}

	if *noUpdateAttr != false {
		config.LDAP.NoUpdateAttr = *noUpdateAttr

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
	stsConnection := sts.New(session.New(&aws.Config{}))
	credentialsService := server.NewDirectSessionTokenService(config.AWS.Account, stsConnection, &config.AccountAliases)

	open := func() (server.LDAPImplementation, error) { return ConnectLDAP(config.LDAP) }
	ldapServer, err := server.NewPersistentLDAP(open)
	if err != nil {
		log.Errorf("Fatal error, exiting: %s", err.Error())
		os.Exit(1)
	}

	ldapCache, err := server.NewLDAPUserCache(ldapServer, stats, config.LDAP.UserAttr, config.LDAP.BaseDN,
		config.LDAP.EnableLDAPRoles, config.LDAP.RoleAttribute, config.AWS.DefaultRole, config.LDAP.DefaultRoleAttr,
		config.LDAP.GroupClassAttr, config.LDAP.PubKeysAttr, config.LDAP.RoleTimeoutAttr, config.LDAP.NoUpdateAttr)
	if err != nil {
		log.Errorf("Top-level error in LDAPUserCache layer: %s", err.Error())
		os.Exit(1)
	}

	serverHandler := server.New(ldapCache, credentialsService, config.AWS.DefaultRole, stats, ldapServer,
		config.LDAP.UserAttr, config.LDAP.BaseDN, config.LDAP.EnableLDAPRoles, config.LDAP.DefaultRoleAttr,
		config.LDAP.PubKeysAttr, config.LDAP.RoleTimeoutAttr)
	server, err := remote.NewServer(config.Listen, serverHandler.HandleConnection)

	// Wait for a signal from the OS to shutdown.
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool)

	// SIGUSR1 and SIGUSR2 should make Hologram enable and disable debug logging, respectively.
	debugEnable := make(chan os.Signal, 1)
	debugDisable := make(chan os.Signal, 1)
	signal.Notify(debugEnable, syscall.SIGUSR1)
	signal.Notify(debugDisable, syscall.SIGUSR2)

	// SIGHUP should make Hologram server reload its cache of user information from LDAP.
	reloadCacheSigHup := make(chan os.Signal, 1)
	signal.Notify(reloadCacheSigHup, syscall.SIGHUP)

	// Reload the cache based on time set in configuration
	cacheTimeoutTicker := time.NewTicker(time.Duration(config.CacheTimeout) * time.Second)

	log.Info("Hologram server is online, waiting for termination.")

	// Handle termination
	go func() {
		s := <-terminate
		log.Info("Signal received by termination handler: %+v", s)
		done <- true
	}()

	// Handle dynamic settings changes
	go func() {
		for {
			select {
			case <-debugEnable:
				log.Info("Enabling debug mode.")
				log.DebugMode(true)
			case <-debugDisable:
				log.Info("Disabling debug mode.")
				log.DebugMode(false)
			case <-reloadCacheSigHup:
				log.Info("Force-reloading user cache.")
				ldapCache.Update()
			case <-cacheTimeoutTicker.C:
				log.Info("Cache timeout. Reloading user cache.")
				ldapCache.Update()
			}
		}
	}()

	<-done
	log.Info("Caught signal; shutting down now.")
	server.Close()
}
