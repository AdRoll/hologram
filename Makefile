# Copyright 2014 AdRoll, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
# http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
REVISION 			:= $(shell git rev-parse HEAD)
BRANCH 				:= $(shell git rev-parse --abbrev-ref HEAD)
GIT_DIRTY 		:= $(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
GIT_TAG				:= $(shell git describe --tags --long | sed 's/-/\./' | sed 's/-g/-/' | sed 's/-/~/')

all: test build package

setup: .setup-complete

.setup-complete:
	@go get github.com/mitchellh/gox
	@go get github.com/jteeuwen/go-bindata/...
	@gox -build-toolchain -osarch="linux/amd64 darwin/amd64"
	@brew install gpm jq protobuf
	@sudo gem install fpm deb-s3
	@touch .setup-complete

package: bin/darwin/Hologram-$(GIT_TAG).pkg bin/linux/hologram-$(GIT_TAG).deb bin/linux/hologram-server-$(GIT_TAG).deb

build: bin/darwin/hologram-server bin/linux/hologram-server bin/darwin/hologram-agent bin/linux/hologram-agent bin/darwin/hologram-cli bin/linux/hologram-cli bin/darwin/hologram-authorize bin/linux/hologram-authorize bin/darwin/hologram-boot

protocol/hologram.pb.go: protocol/hologram.proto
	protoc --go_out=. protocol/hologram.proto

transport/remote/bindata.go: transport/remote/self-signed-ca.cert transport/remote/self-signed.cert transport/remote/self-signed.key
	cd transport/remote; go-bindata -pkg remote self-signed-ca.cert self-signed.cert self-signed.key

agent/bindata.go: agent/test_ssh_key
	cd agent; go-bindata -pkg agent test_ssh_key

%/.deps: %/Godeps
	cd $*; gpm install; touch .deps

bin/%/hologram-agent: protocol/hologram.pb.go agent/.deps agent/*.go agent/*/*.go log/*.go log/.deps transport/remote/*.go transport/local/*.go transport/remote/bindata.go
	@echo "Building agent version $(GIT_TAG)$(GIT_DIRTY)"
	@cd agent/bin; gox -osarch="$*/amd64" -output="../../bin/$*/hologram-agent"

bin/%/hologram-server: protocol/hologram.pb.go server/.deps server/*.go server/*/*.go log/*.go log/.deps transport/remote/*.go transport/remote/bindata.go
	@echo "Building server version $(GIT_TAG)$(GIT_DIRTY)"
	@cd server/bin; gox -osarch="$*/amd64" -output="../../bin/$*/hologram-server"

bin/%/hologram-authorize: protocol/hologram.pb.go tools/install/*.go log/*.go log/.deps transport/remote/*.go transport/remote/bindata.go
	@echo "Building SSH key updater version $(GIT_TAG)$(GIT_DIRTY)"
	@cd tools/install; gox -osarch="$*/amd64" -output="../../bin/$*/hologram-authorize"

bin/%/hologram-cli: protocol/hologram.pb.go cli/*/*.go log/*.go log/.deps transport/local/*.go cli/.deps
	@echo "Building CLI version $(GIT_TAG)$(GIT_DIRTY)"
	@cd cli/bin; gox -osarch="$*/amd64" -output="../../bin/$*/hologram-cli"

bin/darwin/hologram-boot: tools/boot/main.go
	@cd tools/boot/; go build
	@mv tools/boot/boot bin/darwin/hologram-boot

bin/ping: tools/ping/main.go log/*.go log/.deps
	@cd tools/ping; go build
	@mv tools/ping/ping bin/ping

bin/darwin/Hologram-%.pkg: bin/darwin/hologram-agent bin/darwin/hologram-cli bin/darwin/hologram-authorize agent/support/darwin/com.adroll.hologram*.plist agent/support/darwin/postinstall.sh
	@echo "Creating temporary directory for pkgbuild..."
	@mkdir -p pkg/darwin/{root,scripts}
	@mkdir -p ./pkg/darwin/root/{usr/bin,Library/{LaunchDaemons,LaunchAgents},etc/hologram}
	@cp ./bin/darwin/hologram-agent ./pkg/darwin/root/usr/bin/hologram-agent
	@cp ./bin/darwin/hologram-cli ./pkg/darwin/root/usr/bin/hologram
	@cp ./bin/darwin/hologram-authorize ./pkg/darwin/root/usr/bin/hologram-authorize
	@cp ./config/agent.json ./pkg/darwin/root/etc/hologram/agent.json
	@cp ./bin/darwin/hologram-boot ./pkg/darwin/root/usr/bin/hologram-boot
	@cp ./agent/support/darwin/com.adroll.hologram-ip.plist ./pkg/darwin/root/Library/LaunchDaemons
	@cp ./agent/support/darwin/com.adroll.hologram.plist ./pkg/darwin/root/Library/LaunchDaemons
	@cp ./agent/support/darwin/com.adroll.hologram-me.plist ./pkg/darwin/root/Library/LaunchAgents
	@cp ./agent/support/darwin/postinstall.sh ./pkg/darwin/scripts/postinstall
	@chmod a+x ./pkg/darwin/root/usr/bin/hologram*
	@chmod a+x ./pkg/darwin/scripts/postinstall
	@echo "Changing ownership of files - you will need to sudo here!"
	@sudo chown -R root:wheel ./pkg/darwin
	@echo "Building installer package..."
	@pkgbuild --root ./pkg/darwin/root \
		--identifier com.adroll.hologram \
		--version $(GIT_TAG) \
		--ownership recommended \
		--scripts ./pkg/darwin/scripts \
		./bin/darwin/Hologram-$(GIT_TAG).pkg

bin/linux/hologram-server-%.deb: bin/linux/hologram-server server/after-install.sh server/before-remove.sh
	@echo "Creating temporary directory for fpm..."
	@mkdir -p ./pkg/linux/hologram-server/{root,scripts}
	@mkdir -p ./pkg/linux/hologram-server/root/{usr/local/bin,etc/{hologram,init.d}}
	@cp ./config/server.json ./pkg/linux/hologram-server/root/etc/hologram/server.json
	@cp ./server/support/hologram.init.sh ./pkg/linux/hologram-server/root/etc/init.d/hologram
	@cp ./server/after-install.sh ./pkg/linux/hologram-server/scripts/after-install.sh
	@cp ./server/before-remove.sh ./pkg/linux/hologram-server/scripts/before-remove.sh
	@cp ./bin/linux/hologram-server ./pkg/linux/hologram-server/root/usr/local/bin/
	@chmod a+x ./pkg/linux/hologram-server/root/etc/init.d/hologram
	@fpm -s dir -t deb -f                                                        \
		-n hologram-server                                                       \
		-v $(GIT_TAG)                                                            \
		--after-install ./pkg/linux/hologram-server/scripts/after-install.sh     \
		--before-remove ./pkg/linux/hologram-server/scripts/before-remove.sh     \
		--config-files /etc/hologram/server.json \
		-C ./pkg/linux/hologram-server/root                                      \
		-p ./bin/linux/hologram-server-$(GIT_TAG).deb                            \
		-a amd64                                                                 \
		./

bin/linux/hologram-%.deb: bin/linux/hologram-cli bin/linux/hologram-agent bin/linux/hologram-authorize
	@echo "Creating temporary directory for fpm..."
	@mkdir -p ./pkg/linux/hologram/{root,scripts}
	@mkdir -p ./pkg/linux/hologram/root/{usr/local/bin,etc/{hologram,init.d}}
	@cp ./config/agent.json ./pkg/linux/hologram/root/etc/hologram/agent.json
	@cp ./bin/linux/hologram-cli ./pkg/linux/hologram/root/usr/local/bin/hologram
	@cp ./bin/linux/hologram-agent ./pkg/linux/hologram/root/usr/local/bin/hologram-agent
	@cp ./bin/linux/hologram-authorize ./pkg/linux/hologram/root/usr/local/bin/hologram-authorize
	@cp ./agent/support/debian/after-install.sh ./pkg/linux/hologram/scripts/
	@cp ./agent/support/debian/before-remove.sh ./pkg/linux/hologram/scripts/
	@cp ./agent/support/debian/init.sh ./pkg/linux/hologram/root/etc/init.d/hologram-agent
	@chmod a+x ./pkg/linux/hologram/root/etc/init.d/hologram-agent
	@fpm -s dir -t deb                                                   \
		-n hologram-agent                                                \
		-v $(GIT_TAG)                                                    \
		--after-install ./pkg/linux/hologram/scripts/after-install.sh    \
		--before-remove ./pkg/linux/hologram/scripts/before-remove.sh    \
		--config-files /etc/hologram/agent.json \
		-C ./pkg/linux/hologram/root                                     \
		-p ./bin/linux/hologram-$(GIT_TAG).deb                           \
		-a amd64                                                         \
		./

test: protocol/hologram.pb.go server/.deps agent/.deps transport/remote/bindata.go agent/bindata.go
	@echo "Running test suite."
	@go test ./... -v -cover

clean:
	rm -rf ./bin ./build
	sudo rm -rf ./pkg

version:
	@echo "$(GIT_TAG)"

.PHONY: setup all build package clean test version


