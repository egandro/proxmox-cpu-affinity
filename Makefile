.PHONY: all deps build clean test coverage lint tidy deploy deb get-status-core-ranking get-status-svg run-service run-status

DEPLOY_HOST ?= testmox
VERSION ?= 0.0.9

# Detect architecture (default to amd64)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	DETECTED_ARCH := amd64
else ifeq ($(UNAME_M),aarch64)
	DETECTED_ARCH := arm64
else ifeq ($(UNAME_M),arm64)
	DETECTED_ARCH := arm64
else
	DETECTED_ARCH := amd64
endif

ARCH ?= $(DETECTED_ARCH)
LDFLAGS ?=

all: build

deps:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

build:
	mkdir -p bin/$(ARCH)
	CGO_ENABLED=0 GOARCH=$(ARCH) go build -ldflags="$(LDFLAGS)" -o bin/$(ARCH)/proxmox-cpu-affinity-hook ./cmd/hook
	CGO_ENABLED=0 GOARCH=$(ARCH) go build -ldflags="$(LDFLAGS)" -o bin/$(ARCH)/proxmox-cpu-affinity-service ./cmd/service
	CGO_ENABLED=0 GOARCH=$(ARCH) go build -ldflags="$(LDFLAGS)" -o bin/$(ARCH)/proxmox-cpu-affinity ./cmd/cli

clean:
	rm -rf bin dist local *.deb coverage.out coverage.html pkg/svg/testresult

test:
	go clean -testcache
	go test -v -race ./...

coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

lint:
	golangci-lint run ./...
	gosec ./...
	govulncheck ./...

tidy:
	go mod tidy

# FORCE_PURGE=1 make deploy
deploy: deb
	scp proxmox-cpu-affinity_$(VERSION)_$(ARCH).deb $(DEPLOY_HOST):/tmp/proxmox-cpu-affinity.deb
	ssh $(DEPLOY_HOST) "sudo apt $(if $(FORCE_PURGE),purge,remove) proxmox-cpu-affinity -y || true"
	ssh $(DEPLOY_HOST) "sudo dpkg -i /tmp/proxmox-cpu-affinity.deb"
	ssh $(DEPLOY_HOST) "sudo rm -f /tmp/proxmox-cpu-affinity.deb"

get-status-core-ranking:
	ssh $(DEPLOY_HOST) /usr/bin/proxmox-cpu-affinity status core-ranking --json > ./pkg/svg/testdata/core-ranking.json

get-status-svg:
	mkdir -p ./pkg/svg/testresult
	ssh $(DEPLOY_HOST) /usr/bin/proxmox-cpu-affinity status svg > ./pkg/svg/testresult/status-default.svg
	ssh $(DEPLOY_HOST) /usr/bin/proxmox-cpu-affinity status svg --affinity > ./pkg/svg/testresult/status-affinity.svg

deb:
	GOOS=linux $(MAKE) build
	./deb/build.sh $(VERSION) $(ARCH)

run-service: build
	mkdir -p local
	./bin/proxmox-cpu-affinity-service --socket ./local/proxmox-cpu-affinity.sock --log-file ./local/proxmox-cpu-affinity.log --log-level debug --stdout

run-status: build
	./bin/proxmox-cpu-affinity status --socket ./local/proxmox-cpu-affinity.sock
