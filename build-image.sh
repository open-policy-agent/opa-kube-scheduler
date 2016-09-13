#!/usr/bin/env bash

set -ex

GOOS=linux go build -o rego-scheduler ./cmd/rego-scheduler/main.go
docker build -t open-policy-agent/rego-scheduler:0.1.0 .
rm rego-scheduler
