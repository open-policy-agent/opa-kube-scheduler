IMAGE 	:= openpolicyagent/opa-kube-scheduler
BINARY 	:= opa-kube-scheduler
VERSION := 0.1.2
OS 		:= $(shell go env GOOS)
ARCH 	:= $(shell go env GOARCH)

all: build 

build:
	go build -o $(BINARY)_$(OS)_$(ARCH) ./cmd/opa-kube-scheduler/main.go

build-linux:
	GOOS=linux go build -o $(BINARY)_linux_$(ARCH) ./cmd/opa-kube-scheduler/main.go

image: build-linux
	docker build -t $(IMAGE):$(VERSION) .
