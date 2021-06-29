# **[ARCHIVED]** opa-kube-scheduler

This project is no longer active and will be deleted in the future.

This project shows how OPA can policy-enable container scheduling in Kubernetes.

## Build

To build the scheduler binary:

```bash
make
```

To build the Docker image:

```bash
make image
```

## Usage

Kubernetes allows users to run multiple independent schedulers in the cluster. Once an additional scheduler is deployed, pod scheduling can be delegated to it by annotating the pods with the name of the scheduler.

In opa-kube-scheduler, the name of the scheduler is part of the scheduling policy. If annotation does not match the policy, opa-kube-scheduler will ignore the pod:

```ruby
package io.k8s.scheduler

import requested_pod as req

scheduler_name_match :-
	req.metadata.annotations[k8s_scheduler_annotation] = "experimental"

k8s_scheduler_annotation = "scheduler.alpha.kubernetes.io/name"

fit[node] = weight :-
	scheduler_name_match,
	...
```

### Kubernetes

The scheduler can be deployed on Kubernetes. For example, assuming you are using [kubernetes/minikube](https://github.com/kubernetes/minikube) for test purposes, you can try the scheduler as follows:

1. Create a deployment for the scheduler:

	```bash
	kubectl create -f ./examples/deployment.yaml
	```

1. Expose the scheduler's server as as service:

	```bash
	kubectl expose deployment opa-kube-scheduler \
		--port 8181 --target-port 8181 --type NodePort
	```

1.  Obtain the scheduler's URL:

	```bash
	SCHEDULER_URL=$(minikube service opa-kube-scheduler --url)
	```

1. Push the scheduling policy to the scheduler:

	```bash
	curl -X PUT --data-binary \
		@./examples/policy.rego $SCHEDULER_URL/v1/policies/example
	```

1. At this point, the scheduler is deployed and the scheduling policy has been installed. Create a replication controller for nginx as a test:

	```bash
	kubectl create -f ./examples/nginx.yaml
	```

1. If you tail the scheduler's log, you will see that it has scheduled the nginx pods:

	```bash
	kubectl logs $(kubectl get pod | grep opa-kube-scheduler | cut -f 1 -d ' ')
	```

### Development

If you have built the scheduler, you can run it in the development with:

```bash
./opa-kube-scheduler -kubeconfig ~/.kube/config --v 2 --logtostderr 1
```
