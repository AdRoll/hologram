# Common commands for developing and packaging hologram.
#
# Everything under "Packaging" runs inside the adroll/hologram_env container
# (the same thing ./hologram.sh does), so a working Go environment is only
# needed for the "Host Go toolchain" targets.

IMAGE ?= adroll/hologram_env
CONTAINER_DIR := /go/src/github.com/AdRoll/hologram

# Baked into etc/hologram/agent.json for the osx package when set, e.g.
# make build_osx HOLOGRAM_HOST=hologram.example.com:3100
HOLOGRAM_HOST ?=

# Registry for the deployable hologram-server image, e.g.
# make server_image REGISTRY=my.docker.registry.example.com:5000
REGISTRY ?=

# Shared by every packaging target; `console` adds -it for an interactive shell.
DOCKER_RUN := docker run --rm -v "$(CURDIR)":$(CONTAINER_DIR)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

##
## Packaging (runs in Docker)
##

.PHONY: image
image: ## Build the adroll/hologram_env build container
	docker build -t $(IMAGE) .

.PHONY: build_all
build_all: image ## Compile, test, and build the linux and osx packages
	$(DOCKER_RUN) $(IMAGE) build_all "$(HOLOGRAM_HOST)"

.PHONY: build_linux
build_linux: image ## Compile, test, and build the deb/rpm packages
	$(DOCKER_RUN) $(IMAGE) build_linux "$(HOLOGRAM_HOST)"

.PHONY: build_osx
build_osx: image ## Compile, test, and build the osx .pkg installer
	$(DOCKER_RUN) $(IMAGE) build_osx "$(HOLOGRAM_HOST)"

.PHONY: test
test: image ## Cross-compile every binary and run the tests in the container
	$(DOCKER_RUN) $(IMAGE) test

.PHONY: console
console: image ## Open a shell in the build container, repo mounted
	$(DOCKER_RUN) -it $(IMAGE) console

.PHONY: server_image
server_image: ## Build the deployable hologram_server image (needs REGISTRY=, run build_linux first)
	@test -n "$(REGISTRY)" || { echo "REGISTRY is required, e.g. make server_image REGISTRY=my.registry.example.com:5000"; exit 1; }
	cd docker/server && ./build-container.sh $(REGISTRY)

.PHONY: clean
clean: ## Delete the built packages in artifacts/
	rm -f artifacts/*.deb artifacts/*.rpm artifacts/*.pkg

##
## Host Go toolchain (needs Go installed locally)
##

.PHONY: test_local
test_local: ## Run the tests with the local Go toolchain
	go test ./...

.PHONY: build_local
build_local: ## Build every binary for this machine into ./bin
	go build -o bin/ ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Format the source with gofmt
	gofmt -s -w .
