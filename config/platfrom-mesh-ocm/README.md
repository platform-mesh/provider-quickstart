# `config/platfrom-mesh-ocm` — ManagedProvider via OCM

OCM variant of [`config/platfrom-mesh-flux`](../platfrom-mesh-flux). Deploys the same Wild West provider, but the
**controller** and **portal** are sourced from the published **OCM component**
`github.com/platform-mesh/provider-quickstart` instead of directly from Helm OCI charts.

```sh
kubectl apply -k config/platfrom-mesh-ocm
```

> Apply **either** `config/platfrom-mesh-flux` **or** `config/platfrom-mesh-ocm`, not both — they share the
> ManagedProvider `wildwest` in `platform-mesh-system`.

## What's different

| | `config/platfrom-mesh-flux` (`flux:`) | `config/platfrom-mesh-ocm` (`ocm:`) |
|---|---|---|
| Source | Helm OCI chart, pulled directly | OCM component descriptor, resolved by the ocm-controller |
| Operator emits | `OCIRepository` + `HelmRelease` | `Repository` + `Component` + `Resource` → (resolved) `OCIRepository` + `HelmRelease` |
| Gets you | plain chart deploy | signatures / references / relocation via OCM |

The `ocm:` source is **self-contained**: you give the OCM coordinates inline
(`registry` + `component` + `version` + `resourceName`) and the operator creates the
`delivery.ocm.software` `Repository`, `Component` and `Resource` objects for you — no CRs
shipped by hand. The ocm-controller resolves the descriptor and the resolved chart is
deployed via Flux.

```yaml
ocm:
  name: wildwest-controller                                # generated object names
  registry: ghcr.io/platform-mesh                          # → Repository (created by operator)
  component: github.com/platform-mesh/provider-quickstart  # → Component  (created by operator)
  version: "0.0.8"
  resourceName: controller-chart                           # resource within the component
  values: {...}                                            # Helm values (how to configure)
```

`name` is set explicitly because the controller and portal resolve from the **same**
component name and would otherwise collide on the generated object names.

The armament syncer is **not** part of the OCM component, so it stays on `flux:` —
showing the two sources coexisting in one ManagedProvider.

## Prerequisites

1. The ocm-controller (ocm-k8s-toolkit) must be installed in the runtime cluster
   (provides the `delivery.ocm.software` CRDs).
2. The OCM component must be published first:
   ```sh
   make ocm-push            # transfers the component to ghcr.io/platform-mesh
   ```
   Keep the `version:` in each `ocm:` entry in sync with the published `VERSION`.
