# Bootstrapping the quickstart provider with `ManagedProvider`

This directory contains a declarative way to bootstrap the quickstart ("Wild West")
provider using the [platform-mesh-operator](https://github.com/platform-mesh/platform-mesh-operator)
`ManagedProvider` machinery — instead of the manual `make init` + local controller
flow documented in the top-level [README](../../README.md).

You apply **one** resource ([`managedprovider.yaml`](./managedprovider.yaml)) to the
runtime cluster and the operator drives the entire lifecycle.

## What `ManagedProvider` does

`ManagedProvider` is a namespaced CRD
(`providers.platform-mesh.io/v1alpha1`) reconciled by the operator through a chain of
subroutines:

| # | Subroutine        | Effect |
|---|-------------------|--------|
| 1 | `WaitPlatformMesh`| Waits for the referenced `PlatformMesh` to be `Ready`. |
| 2 | `ProviderResource`| Creates a kcp `Provider` (default path `root:providers:system`). The Provider controller provisions a dedicated provider **workspace** + a scoped admin kubeconfig `Secret`. |
| 3 | `WaitProvider`    | Waits for the `Provider` to reach `phase=Ready`. |
| 4 | `KubeconfigCopy`  | Copies that kubeconfig into the runtime namespace as `Secret/wildwest-provider-kubeconfig` (key `kubeconfig`). |
| 5 | `Deploy`          | For each `runtimeDeployments[].ocm` entry, creates a Flux `OCIRepository` (`oci://<registry>/<componentName>:<version>`, helm-chart layer) + a `HelmRelease` that installs the chart into the runtime namespace. |

The provider workspace created in step 2 is **empty**. The `wildwest-controller`
chart's **init container** (`init.enabled: true`) bootstraps the workspace content —
the Cowboys `APIResourceSchema`/`APIExport`, the Armaments CRD and the
`ProviderMetadata` — using the copied provider kubeconfig, before the controller's main
container begins reconciling.

## Prerequisites

On the **runtime cluster**:

- The **platform-mesh-operator** is installed and running (it owns the
  `ManagedProvider`/`Provider` CRDs and reconcilers).
- **FluxCD** source-controller + helm-controller are installed (the operator emits
  `OCIRepository` / `HelmRelease` objects — GVKs `source.toolkit.fluxcd.io/v1` and
  `helm.toolkit.fluxcd.io/v2`).
- A **`PlatformMesh`** object named `platform-mesh` exists and is `Ready` (adjust
  `spec.platformMeshRef.name` if yours differs).
- The `platform-mesh-system` namespace exists.

Published artifacts the manifest references:

- OCI **Helm charts** at (version tracks the release tag — latest is `v0.0.7`):
  - `oci://ghcr.io/platform-mesh/provider-quickstart/charts/wildwest-controller:0.0.7`
  - `oci://ghcr.io/platform-mesh/provider-quickstart/charts/wildwest-portal:0.0.7`
- The chart `values.image.tag` digests resolve from `ghcr.io/platform-mesh/provider-quickstart*`.

> **Publishing note.** The operator's `Deploy` subroutine consumes **plain OCI Helm
> chart artifacts** (Flux `OCIRepository` with the helm-chart layer selector), *not* the
> bundled OCM component descriptor. The charts under [`deploy/helm/`](../../deploy/helm/)
> are published as standalone OCI Helm charts under this repo's own GHCR namespace
> (`ghcr.io/platform-mesh/provider-quickstart/charts`, alongside the container images) by
> the `publish-helm` job in [`build-images.yaml`](../../.github/workflows/build-images.yaml)
> on every `v*` tag (via `make helm-push`). CI versions all release artifacts from the git
> tag (`VERSION = ${tag#v}`), so the chart version equals the release tag — latest is
> `v0.0.7`. (The internal `deploy/helm/*/Chart.yaml` version is overridden at publish
> time.) If you publish elsewhere, update `registry` / `componentName` / `version` in
> [`managedprovider.yaml`](./managedprovider.yaml) accordingly.

## Configure the front-proxy IP (required)

kcp advertises the provider's APIExport virtual-workspace endpoint as
`https://root.kcp.localhost:8443/...`. Inside the controller pod that hostname
resolves to `127.0.0.1`, so the controller's endpoint watcher fails with
`dial tcp 127.0.0.1:8443: connect: connection refused`. To fix this the chart
pins those hostnames to the front-proxy service ClusterIP via `hostAliases`.

This is the **one value you must set for your cluster**. Find the IP:

```bash
kubectl -n platform-mesh-system get svc frontproxy-front-proxy \
  -o jsonpath='{.spec.clusterIP}'
```

and set it in [`kustomization.yaml`](./kustomization.yaml) (the `patches:` entry,
`value:`). The IP is stable for the lifetime of the service; update it if the
front-proxy Service is recreated.

## Apply

```bash
# kube context / KUBECONFIG must point at the runtime cluster running the operator
kubectl apply -k config/pm
```

## Observe

```bash
# Lifecycle phase: Pending -> WaitingForPlatformMesh -> WaitingForProvider
#                  -> CopyingKubeconfig -> Deploying -> Ready
kubectl get managedprovider wildwest -n platform-mesh-system -w

# The Provider created in kcp
kubectl get provider -A

# Copied provider kubeconfig
kubectl get secret wildwest-provider-kubeconfig -n platform-mesh-system

# Flux objects emitted by the Deploy subroutine
kubectl get ocirepository,helmrelease -n platform-mesh-system

# Workloads
kubectl get pods -n platform-mesh-system -l app.kubernetes.io/name=wildwest-controller
```

## Tear down

```bash
kubectl delete -k config/pm
```

Set `spec.cleanupOnDelete: true` in [`managedprovider.yaml`](./managedprovider.yaml)
to also remove the kcp provider workspace on deletion. The `Deploy` finalizer removes
the `HelmRelease`/`OCIRepository` objects (and thus the deployed charts) regardless.
