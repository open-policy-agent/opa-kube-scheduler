# rego-scheduler

This project shows how OPA can policy-enable container scheduling in Kubernetes.

## Usage

This project has been built and tested on Go 1.6.x.

### Build

To build rego-scheduler, run the following commands:

```
$ go get ./...
$ go build -o rego-scheduler ./cmd/rego-scheduler/main.go
```

This assumes you have Go installed on your system and GOPATH is exported.

### Running

To run rego-scheduler with the default policy module:

```
$ ./rego-scheduler --cluster_url <Kubernetes API URL>   # e.g., http://localhost:8080/api/v1
```

By default, rego-scheduler will drop into an interactive shell (REPL) that
lets you execute ad-hoc queries against the Kubernetes data that the scheduler
synchronizes.

For example, you can use the REPL to show the nodes that the
scheduler is currently aware of:

```
> data.nodes[id].metadata.name
```

rego-scheduler will watch the policy module for changes and automatically reload it when it detects a chang.

## Future Work

- Server mode: run rego-scheduler in the background and expose an API manage policies and extra data relevant to scheduling.
