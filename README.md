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

* A Go development environment on one workstation, with `$GOPATH` already setup. A docker container (adroll/hologram_env) is provided that includes all dependencies and support for cross-compiling linux/osx binaries and building deb/osx packages. This container can be launched from the hologram.sh script, so you don't even need to have a working go environment to build and develop on hologram.
* An LDAP server with `sshPublicKey` attributes on user records to store their SSH public keys.
* `ssh-agent` running on all workstations using Hologram Agent, configured to load the key you have stored in LDAP for that user.
* An AWS account that you can administer IAM permissions in.
* A "developer" IAM role or something similar that has the minimum permissions that you want your developers to have by default.
* Developers using Hologram must be running OS X or Linux machines. The built packages support Debian derivatives. No Windows support is planned, but patches are welcome.

### docker

The hologram build script expects a working docker installation on the local host.

If you're building hologram on OS X, you'll need to use Boot2Docker to run Docker inside of VirtualBox, because Linux. Follow the instructions at https://docs.docker.com/installation/mac/.

## Installation
Hologram currently doesn't ship pre-compiled binaries, so you'll need to build it yourself. A docker container is provided that contains all that's needed to test, compile and build hologram packages for both debian and osx. You just need to invoke the script from the same directory where the hologram source lives. This is a full example of testing and building packages for all supported platforms.
```
➞  ./hologram.sh build_all
    >> Getting package github.com/golang/protobuf/...
    >> Getting package golang.org/x/crypto/ssh
    >> Getting package github.com/aybabtme/rgbterm
    >> Getting package github.com/mitchellh/go-homedir
    >> Getting package github.com/nmcclain/ldap
    >> Getting package github.com/peterbourgon/g2s
    >> Getting package github.com/goamz/goamz/...
    >> Getting package github.com/smartystreets/goconvey/...
    >> Setting github.com/golang/protobuf/... to version a8323e2cd7e8ba8596aeb64a2ae304ddcd7dfbc0
    >> Setting golang.org/x/crypto/ssh to version 88b65fb66346493d43e735adad931bf69dee4297
    >> Setting github.com/nmcclain/ldap to version f4e67fa4cd924fbe6f271611514caf5589e6a6e5
    >> Setting github.com/peterbourgon/g2s to version ec76db4c1ac16400ac0e17ca9c4840e1d23da5dc
    >> Setting github.com/aybabtme/rgbterm to version c07e2f009ed2311e9c35bca12ec00b38ccd48283
    >> Setting github.com/goamz/goamz/... to version 63291cb652bc024bcd52303631afad8f230b8244
    >> Setting github.com/smartystreets/goconvey/... to version 1d9daca83fc3cf35d01b9d0ac2debad3453bf178
    >> Setting github.com/mitchellh/go-homedir to version 7d2d8c8a4e078ce3c58736ab521a40b37a504c52
    >> Building package github.com/golang/protobuf/...
    >> Building package golang.org/x/crypto/ssh
    >> Building package github.com/aybabtme/rgbterm
    >> Building package github.com/mitchellh/go-homedir
    >> Building package github.com/nmcclain/ldap
    >> Building package github.com/peterbourgon/g2s
    >> Building package github.com/goamz/goamz/...
    >> Building package github.com/smartystreets/goconvey/...
    >> All Done
    Running tests...
    === RUN TestCliHandler

      AssumeRole ✔✔✔✔


    4 assertions thus far

    --- PASS: TestCliHandler (0.00s)

    <...>

    === RUN TestSSLWithSelfSignedRootCA

      Given a test server with self-signed SSL certificates ✔
        When a client connects and pings ✔✔
          Then it should get a pong response ✔✔


    5 assertions thus far

    --- PASS: TestSSLWithSelfSignedRootCA (0.33s)
    PASS
    ok      github.com/AdRoll/hologram/transport/remote     0.380s
    Compiling for linux...
    Compiling for osx
    /var/lib/gems/2.1.0/gems/fpm-1.3.3/lib/fpm/util.rb:127: warning: Insecure world writable dir /go/src in PATH, mode 040777
    Created package {:path=>"/go/src/github.com/AdRoll/hologram/artifacts/hologram-1.1.42~23a3e63.deb"}
    /var/lib/gems/2.1.0/gems/fpm-1.3.3/lib/fpm/util.rb:127: warning: Insecure world writable dir /go/src in PATH, mode 040777
    Created package {:path=>"/go/src/github.com/AdRoll/hologram/artifacts/hologram-server-1.1.42~23a3e63.deb"}
    44009 blocks
    2 blocks
    osx package has been built
```

To access the full development environment, with all the needed dependencies and cross-compiling support just do:
```
    $ ./hologram.sh console
```
Please note that you'll probably need to update the `config/{agent,server}.json` files included for your particular deployment. If you edit these, they will be included in the compiled packages. You may distribute the files in any other way you may wish, but note that they must be in this format, at `/etc/hologram/agent.json` and `/etc/hologram/server.json` respectively.

### Deployment
1. Launch an EC2 instance with an instance profile with permissions detailed in `permissions.json`.
2. Deploy the built `hologram-server.deb` to the server you just launched.

### Deployment using Docker
A docker base image is provided if you want to deploy hologram inside a docker container, you just need to add the config file for your environment and you'll be good to go. Our image uses ONBUILD to copy the config file so you just need to use it as a base for your own images, where you can do (or not) any other customizations you might want, just ensure there's a config.json file next to it, as the onbuild trigger will do ``` COPY config.json /etc/hologram/server.json```
```
FROM adroll/hologram_server
```
For each new hologram release an updated image will be pushed to https://registry.hub.docker.com/u/adroll/hologram_server/

### Create a new base image
If you're hacking on hologram and want to create your own base image, you can build it yourself. Assuming you have already built a .deb package of hologram-server you want to deploy and that you have also a server.json config file you can use this is what you need to do:
```
cd docker/server
./build-container.sh my.docker.registry.example.com:5000 # Usage is ./build-container.sh CONTAINER_NAME
```
And it will build a base container that can be pushed to your favourite private docker registry, just add the config file and you're good to go!

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

### Account Aliases
The config files can set accountAliases, a dictionary from short name to account iam arn, `arn:aws:iam::ACCOUNT-ID-WITHOUT-HYPHENS`.  If you run `hologram use key/rolename`, it will expand it out to the full arn.  This config param is supported on both the server(org wide accounts), or client(individual accounts).

```json
{
  "host":"localhost:3100",
  "accountAliases":{
    "dev":"arn:aws:iam::123456"
  }
}```

With this config, `hologram use dev/service` would be equivalent to `hologram use arn:aws:iam::123456:role/service`

### Serverless

The hologram agent supports being run without a server, based on long-lived user credentials.  To use, instead of defining host in the config.json file, it uses the go sdk [default credentials provider](https://github.com/aws/aws-sdk-go/#configuring-credentials) on the hologram-agent.
This can be done by adding environment variables to the launch daemon .plist, or set the HOME environment variable in the plist to the directory containing your `.aws` directory.

The user must have permission to iam:GetUser on itself(resource "arn:aws:iam::ACCOUNT-ID-WITHOUT-HYPHENS:user/${aws:username}").  `hologram me` uses getsessiontoken from sts, which has some [limitations](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_control-access_getsessiontoken.html).  With assume role permissions, `hologram use <role>` will assume into any role bypassing those restrictions.

### LDAP Based Roles

Hologram supports assigning roles based on a user's LDAP group. Roles can be turned on by setting the `enableLDAPRoles` key to `true` in `config/server.json`.

An LDAP group attribute will have to be chosen for user roles. By default `businessCategory` is chosen for this role since it is part of the core LDAP schema. The attribute used can be modified by editing the `roleAttribute` key in `config/server.json`. The value of this attribute should be the name of the group's role in AWS.

Users will have to be added to a group giving them access to the default role before they can use Hologram. It is recommended that a group such as `Hologram-Users` be created with attribute `businessCategory` set to the name of the default AWS role.

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

## Related Projects

[Holochrome](https://github.com/Bridgewater/holochrome) is a chrome extension that allows you to easily log in and switch between your AWS accounts using a single key stroke.  As it uses the metadata service, it can use hologram as its credential provider.

## License

Licensed under the Apache License, Version 2.0 (the "License"); see LICENSE for more details.
