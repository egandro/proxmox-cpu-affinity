.PHONY: all deps build clean test coverage lint tidy deploy deb

DEPLOY_HOST ?= testmox
VERSION ?= 0.1.1
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
	go build -o bin/proxmox-cpu-affinity-cpuinfo ./cmd/cpuinfo

clean:
	rm -rf bin dist *.deb coverage.out coverage.html

test:
	go clean -testcache
	go test -v ./...

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
