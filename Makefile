IMAGE 	:= openpolicyagent/rego-scheduler
BINARY 	:= rego-scheduler
VERSION := 0.1.1

all: build 
	@echo $(TAG)

build:
	go build -o $(BINARY) ./cmd/rego-scheduler/main.go

build-linux:
	GOOS=linux go build -o $(BINARY) ./cmd/rego-scheduler/main.go

image: build-linux
	docker build -t $(IMAGE):$(VERSION) .
