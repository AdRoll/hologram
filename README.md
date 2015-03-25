# Hologram [![Build Status](https://travis-ci.org/AdRoll/hologram.svg?branch=master)](https://travis-ci.org/AdRoll/hologram)

## Overview
Storing your AWS keys in source code is a Real Bad Idea, but few good options exist to mitigate this risk that aren't terribly inconvenient. Hologram aims to change this.

EC2 has a feature called "IAM Roles" where a special endpoint in the instance metadata service (http://169.254.169.254/...) exposes temporary AWS API access credentials that have permissions defined by the instance's Role, configured at launch time. In this way, applications can be designed that do not require secret keys checked into their repositories at all, and the chance of malicious key usage is reduced. This service only exists in EC2, but Hologram brings it to non-EC2 hosts, so that developers can run the same software with the same credentials source as in production.

Hologram exposes an imitation of the EC2 instance metadata service on developer workstations that supports the temporary credentials workflow. It is accessible via the same HTTP endpoint to calling SDKs, so your code can use the same process in both development and production. The keys that Hologram provisions are temporary, so EC2 access can be centrally controlled without direct administrative access to developer workstations.

Hologram comes in three parts:

1. hologram-server that runs on an EC2 host in your AWS account that services requests for credentials by authenticating clients' via SSH against an LDAP database, then requesting temporary credentials using the IAM API
1. hologram-agent runs on OS X and Linux workstations, exposing the metadata service interface and fetching credentials from hologram-server as needed.
1. hologram CLI allows users to switch what IAM role they are currently using.

Your software interacts with Hologram in the same manner that you would a production EC2 instance with an instance profile - you communicate with the same `169.254.169.254` IP and get credentials in the same format. If you use Boto or the AWS Java SDK or GoAMZ you are probably already configured to do this.

## Pre-requisites
Hologram requires the following things to already be setup:

* A Go development environment on one workstation, with `$GOPATH` already setup.
* An LDAP server with `sshPublicKey` attributes on user records to store their SSH public keys.
* `ssh-agent` running on all workstations using Hologram Agent, configured to load the key you have stored in LDAP for that user.
* An AWS account that you can administer IAM permissions in.
* A "developer" IAM role or something similar that has the minimum permissions that you want your developers to have by default.
* Developers using Hologram must be running OS X or Linux machines. The built packages support Debian derivatives. No Windows support is planned, but patches are welcome.

## Installation
Hologram currently doesn't ship pre-compiled binaries, so you'll need to build it yourself. This build process has only been tested on OS X Mavericks and OS X Yosemite. Because it compiles binaries for OS X, you probably can't do the building on OS X. It does, however, cross-compile to Linux just fine using gox.

### Building Packages
Hologram comes with support for packaging for Debian-based servers. To build these, do the following:

1. `mkdir -p $GOPATH/src/github.com/AdRoll`
2. `git clone git@github.com:AdRoll/hologram.git $GOPATH/src/github.com/AdRoll/hologram`
3. `cd $GOPATH/src/github.com/AdRoll/hologram`
4. `make setup`: This will setup dependencies for building Hologram on your workstation.
5. Modify the `config/{agent,server}.json` files included for your particular deployment. If you edit these, they will be included in the compiled packages. You may distribute the files in any other way you may wish, but note that they must be in this format, at `/etc/hologram/agent.json` and `/etc/hologram/server.json` respectively.
6. `make package`: This will build the Hologram programs for OS X and Linux, and build installers for each. You will need to `sudo` during this process.

### Deployment
1. Launch an EC2 instance with an instance profile with permissions detailed in `permissions.json`.
2. Deploy the built `hologram-server.deb` to the server you just launched.

### Rollout to Developers
1. Import each developer's SSH key into LDAP. Hologram will search for their key in the `sshPublicKey` attribute.
2. Install the `hologram-agent.pkg` installer you built before on each developer's workstation.

## Usage

If you use Boto or any of the official AWS SDKs, your code is already able to take advantage of Hologram. Simply delete any explicit references to access keys and secrets in your code, and remove environment variables that you may be using, and the application will detect and use the keys provided by the Hologram agent. No further code modification should be necessary.

### Default Behaviour
By default, and at boot, Hologram is configured to hand out credentials for the "developer" role specified in `server.json`. Credentials will be automatically refreshed by Hologram as needed by calling programs.

### Roles
The role of the hologram server must have assume role permissions.  See permissions.json for an example to grant access to all roles - you can limit the roles here.

For different projects it is recommended that you create IAM roles for each and have your developers assume these roles for testing the software. Hologram supports a command `hologram use <rolename>` which will fetch temporary credentials for this role instead of the default developer one until it is reset or another role is assumed.

You will need to modify the Trusted Entities for each of these roles that you create so that the IAM instance profile you created for the Hologram Server can access them. The hologram user must have permission to assume that role. 

```json
    {
      "Sid": "account",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<aws_account_id>:root"
      },
      "Action": "sts:AssumeRole"
    }
```

## Deployment Suggestions
At AdRoll we have Hologram deployed in a fault-tolerant setup, with the following:

* An AutoScaling Group that keeps at least two Hologram servers online in different AZs.
* An Elastic Load Balancer in front of these instances.
* Security Groups that control access to the ELB to just our office networks and the VPN.

## Gotchas
Here are some issues we've run into running Hologram that you might want to be aware of:

* **Sometimes OS X workstations don't like SSH agent.** Some developers have needed to do `ssh-add -K` to add their key to the keychain; some have needed to do this every time they boot; and some just don't require it at all. Your mileage may vary.
* **If you use an ELB to load-balance between Hologram servers, do not have it terminate the TLS connection.** It's pointless to have your ELB use the SSL certificate compiled into Hologram, when the servers themselves know how to handle it. Let them do their job, and have your ELB just use the TCP protocol.
* **Your LDAP server might not support TLS** In that case, you'll want to set "insecureldap" to true in the server config file which will configure hologram to connect to the LDAP server without using TLS. Otherwise you might just get a (somewhat cryptic) "connection reset by peer" error.

## License

Licensed under the Apache License, Version 2.0 (the "License"); see LICENSE for more details.
