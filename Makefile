.PHONY: all deps build clean test lint tidy deploy deb

DEPLOY_HOST ?= testmox
VERSION ?= 0.1.0
ARCH ?= amd64

all: build

deps:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

build:
	mkdir -p bin
	go build -o bin/proxmox-cpu-affinity-hook ./cmd/hook
	go build -o bin/proxmox-cpu-affinity-service ./cmd/service
	go build -o bin/proxmox-cpu-affinity-cpuinfo ./cmd/cpuinfo

clean:
	rm -rf bin dist *.deb

test:
	go clean -testcache
	go test -v ./...

lint:
	golangci-lint run ./...
	gosec ./...

tidy:
	go mod tidy

deploy: deb
	ssh $(DEPLOY_HOST) "rm -rf /tmp/pa-scripts"
	scp -r scripts $(DEPLOY_HOST):/tmp/pa-scripts
	ssh $(DEPLOY_HOST) "sudo mkdir -p /opt/proxmox-cpu-affinity/scripts"
	ssh $(DEPLOY_HOST) "sudo cp -r /tmp/pa-scripts/* /opt/proxmox-cpu-affinity/scripts/ && sudo chmod +x /opt/proxmox-cpu-affinity/scripts/* && rm -rf /tmp/pa-scripts"

	scp proxmox-cpu-affinity_$(VERSION)_$(ARCH).deb $(DEPLOY_HOST):/tmp/proxmox-cpu-affinity.deb
	ssh $(DEPLOY_HOST) "sudo apt remove proxmox-cpu-affinity -y || true"
	ssh $(DEPLOY_HOST) "sudo dpkg -i /tmp/proxmox-cpu-affinity.deb"
	ssh $(DEPLOY_HOST) "sudo rm -f /tmp/proxmox-cpu-affinity.deb"

deb: build
	./deb/build.sh $(VERSION) $(ARCH)
