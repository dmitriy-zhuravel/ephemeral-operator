# EphemeralEnvironment Operator

Kubernetes operator that manages the lifetime of temporary (ephemeral) environments.

> **Status:** Work in progress / early development  
> This project is under active development and is not production-ready yet.

## What it does

The operator watches custom resources of kind `EphemeralEnvironment` and automatically cleans up target namespaces when their TTL expires.

Supported actions:

- **ScaleToZero** — scales Deployments and StatefulSets in the target namespace to 0 replicas
- **Delete** — deletes the entire target namespace

Main features currently implemented:

- CRD `EphemeralEnvironment` with `TTL`, `TargetNamespace` and `Action` fields
- Status subresource (`ExpiryTime`, `State`)
- Reconciliation loop with `RequeueAfter` based on remaining TTL
- Helper functions for scaling and namespace deletion

Planned / in progress:

- Finalizers for safe cleanup on early deletion
- Better status updates and error handling
- Kubernetes Events
- Helm chart and improved samples

## Motivation

This is a learning / portfolio project focused on understanding the Kubernetes operator pattern:

- Reconciliation loop
- Resource lifecycle and finalizers
- RBAC
- Working with core Kubernetes resources (Deployments, StatefulSets, Namespaces)

## Quick start (development)

### Prerequisites

- Go 1.22+
- Docker
- `kubectl` configured to access a cluster (kind / minikube / real cluster)
- Access to a Kubernetes cluster

### Install CRDs

```sh
make install
```

### Run the operator locally

```sh
make run
```

The operator will use your current kubeconfig and watch the cluster.

### Create a sample resource

```sh
kubectl apply -f config/samples/
```

Example fields:

```yaml
spec:
  ttl: 10m
  targetNamespace: test-namespace
  action: ScaleToZero   # or Delete
```

### Useful commands

```sh
# Watch the custom resource
kubectl get ephemeralenvironments -A

# Check status
kubectl get ephemeralenvironment <name> -o yaml

# Clean up
make uninstall
make undeploy
```

## Project structure (high level)

```
api/          - CRD types and generated code
controllers/  - Reconcile logic
config/       - CRD manifests, RBAC, samples, kustomize
```

## Notes

- This project is intentionally kept relatively simple to focus on core operator concepts.
- Some parts (finalizers, full status management, events) are still being refined.
- Feedback and suggestions are welcome once the basic flow is more stable.

## License

Apache License 2.0
