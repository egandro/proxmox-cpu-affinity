.PHONY: all deps build clean test coverage lint tidy deploy deb

DEPLOY_HOST ?= testmox
VERSION ?= 0.0.4
ARCH ?= amd64

all: build

deps:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

build:
	mkdir -p bin
	go build -o bin/proxmox-cpu-affinity-hook ./cmd/hook
	go build -o bin/proxmox-cpu-affinity-service ./cmd/service
	go build -o bin/proxmox-cpu-affinity ./cmd/cli

clean:
	rm -rf bin dist local *.deb coverage.out coverage.html

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

deb: build
	./deb/build.sh $(VERSION) $(ARCH)

run-service: build
	mkdir -p local
	./bin/proxmox-cpu-affinity-service --socket ./local/proxmox-cpu-affinity.sock --log-file ./local/proxmox-cpu-affinity.log --log-level debug --stdout

run-status: build
	./bin/proxmox-cpu-affinity status --socket ./local/proxmox-cpu-affinity.sock
