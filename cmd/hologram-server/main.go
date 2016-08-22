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

const (
	LDAPUserStorage = "ldap"
	FileUserStorage = "file"
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
		// Still here for backwards compatibility
		enableLDAPRoles   = flag.Bool("ldaproles", false, "Enable role support using LDAP directory (DEPRECATED: Use enableServerRoles instead).")
		enableServerRoles = flag.Bool("serverRoles", false, "Enable role support using server directory.")
		roleAttribute     = flag.String("roleattribute", "", "Group attribute to get role from.")
		defaultRoleAttr   = flag.String("defaultroleattr", "", "User attribute to check to determine a user's default role.")
		defaultRole       = flag.String("role", "", "AWS role to assume by default.")
		userStorage       = flag.String("userStorage", LDAPUserStorage, "User storage type (ldap, file)")
		configFile        = flag.String("conf", "/etc/hologram/server.json", "Config file to load.")
		cacheTimeout      = flag.Int("cachetime", 3600, "Time in seconds after which to refresh LDAP user cache.")
		debugMode         = flag.Bool("debug", false, "Enable debug mode.")
		config            Config
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
	if config.UserStorage == "" {
		config.UserStorage = *userStorage
	}

	// Validating user storage value
	if config.UserStorage != LDAPUserStorage && config.UserStorage != FileUserStorage {
		log.Errorf("Invalid user storage value: %s. Possible values (%s, %s)", config.UserStorage, LDAPUserStorage, FileUserStorage)
		os.Exit(1)
	}

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

	if *enableServerRoles || *enableLDAPRoles {
		config.LDAP.EnableLDAPRoles = true
	}

	if config.LDAP.EnableLDAPRoles || *enableLDAPRoles {
		config.EnableServerRoles = true
	}

	if *defaultRoleAttr != "" {
		config.LDAP.DefaultRoleAttr = *defaultRoleAttr
	}

	if *roleAttribute != "" {
		config.LDAP.RoleAttribute = *roleAttribute
	}

	if *cacheTimeout != 3600 {
		config.CacheTimeout = *cacheTimeout
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
	credentialsService := server.NewDirectSessionTokenService(config.AWS.Account, stsConnection)

	var (
		userCache       server.UserCache
		userStorageImpl server.UserStorage
	)

	if config.UserStorage == LDAPUserStorage {
		open := func() (server.LDAPImplementation, error) { return ConnectLDAP(config.LDAP) }
		userStorageImpl, err := server.NewPersistentLDAP(open)
		if err != nil {
			log.Errorf("Fatal error, exiting: %s", err.Error())
			os.Exit(1)
		}

		userCache, err = server.NewLDAPUserCache(userStorageImpl, stats, config.LDAP.UserAttr, config.LDAP.BaseDN, config.EnableServerRoles, config.LDAP.RoleAttribute, config.AWS.DefaultRole, config.LDAP.DefaultRoleAttr)
		if err != nil {
			log.Errorf("Top-level error in LDAPUserCache layer: %s", err.Error())
			os.Exit(1)
		}
	} else if config.UserStorage == FileUserStorage {
		open := func() ([]byte, error) { return ioutil.ReadFile(config.KeysFile.filePath) }
		dump := func(data []byte) error {
			return ioutil.WriteFile(config.KeysFile.filePath, data, os.FileMode(500))
		}
		userStorageImpl := server.NewPersistentKeysFile(open, dump, config.KeysFile.UserAttr, config.KeysFile.RoleAttr)
		userCache, _ = server.NewKeysFileUserCache(userStorageImpl, stats, config.EnableServerRoles, config.KeysFile.UserAttr, config.KeysFile.RoleAttr, config.AWS.DefaultRole, config.KeysFile.DefaultRoleAttr)
	}

	serverHandler := server.New(userCache, credentialsService, config.AWS.DefaultRole, stats, userStorageImpl, config.EnableServerRoles)
	server, _ := remote.NewServer(config.Listen, serverHandler.HandleConnection)

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
	reloadCacheSigHup := make(chan os.Signal)
	signal.Notify(reloadCacheSigHup, syscall.SIGHUP)

	// Reload the cache based on time set in configuration
	cacheTimeoutTicker := time.NewTicker(time.Duration(config.CacheTimeout) * time.Second)

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
		case <-reloadCacheSigHup:
			log.Info("Force-reloading user cache.")
			userCache.Update()
		case <-cacheTimeoutTicker.C:
			log.Info("Cache timeout. Reloading user cache.")
			userCache.Update()
		}
	}

	log.Info("Caught signal; shutting down now.")
	server.Close()
}
