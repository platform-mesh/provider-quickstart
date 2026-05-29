# Platform Mesh Provider Quickstart

[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/platform-mesh/provider-quickstart/badge)](https://scorecard.dev/viewer/?uri=github.com/platform-mesh/provider-quickstart)

A quickstart template for building Platform Mesh providers. This repo demonstrates how to create a provider that exposes APIs through kcp and integrates with the Platform Mesh UI.

## What This Repo Does

This is an example "Wild West" provider that exposes a `Cowboys` API (`wildwest.platform-mesh.io`). It shows how to:

1. **Define and export APIs via kcp** - Using `APIExport` and `APIResourceSchema` resources
2. **Register as a Platform Mesh provider** - Using `ProviderMetadata` to describe your provider
3. **Configure UI integration** - Using `ContentConfiguration` to add navigation and views

## Key Resources

### ProviderMetadata

Registers your provider with Platform Mesh, including display name, description, contacts, and icons:

```yaml
apiVersion: ui.platform-mesh.io/v1alpha1
kind: ProviderMetadata
metadata:
  name: wildwest.platform-mesh.io  # Must match your APIExport name
spec:
  displayName: Wild West Provider
  description: ...
  contacts: [...]
  icon: {...}
```

### ContentConfiguration

Configures how your resources appear in the Platform Mesh UI. The key label links it to your APIExport:

```yaml
apiVersion: ui.platform-mesh.io/v1alpha1
kind: ContentConfiguration
metadata:
  labels:
    ui.platform-mesh.io/content-for: wildwest.platform-mesh.io  # Links to your APIExport
  name: cowboys-ui
spec:
  inlineConfiguration:
    content: |
      { "luigiConfigFragment": { ... } }
```

The `ui.platform-mesh.io/content-for` label is critical - it associates your UI configuration with your APIExport.

## Project Structure

```
├── cmd/
│   ├── init/              # Bootstrap CLI tool
│   ├── wild-west/         # Provider operator (consumer-workspace controller via APIExport)
│   └── armament-sync/     # Catalog syncer (provider-workspace controller, ticker-driven)
├── config/
│   ├── crds/              # CRDs (armaments CRD also installed in the provider workspace)
│   ├── kcp/               # kcp resources (APIExport, APIResourceSchema, CachedResource)
│   └── provider/          # Provider resources (ProviderMetadata, ContentConfiguration, RBAC)
├── operator/
│   ├── wild-west/         # Cowboy reconciler
│   └── armament-sync/     # Armament catalog reconciler
├── pkg/
│   ├── bootstrap/         # Bootstrap logic for applying resources
│   └── external/          # External-source client interface (+ static dev client)
└── portal/                # Custom UI microfrontend example (Angular + Luigi)
```

## Usage

> **Important:** Providers must live in a dedicated workspace type within a separate tree. This means platform administrators must configure providers using the **admin kubeconfig**. Regular user kubeconfigs will not have the necessary permissions to create provider workspaces. This is bound to change and improve in the future, but for now you must use the admin kubeconfig to set up your provider.

### 1. Set Admin Kubeconfig

You need the admin kubeconfig to create and manage provider workspaces:

```bash
cp ../helm-charts/.secret/kcp/admin.kubeconfig kcp-admin.kubeconfig
export PM_KUBECONFIG="$(realpath kcp-admin.kubeconfig)"
kind export kubeconfig --name platform-mesh --kubeconfig compute.kubeconfig
export COMPUTE_KUBECONFIG="$(realpath compute.kubeconfig)"
```

### 2. Create Provider Workspace Hierarchy and Bootstrap

The init binary can either bootstrap into an already-selected workspace (`make init`)
or seed the full workspace hierarchy from the admin kubeconfig first
(`make init-seed-workspaces`). The latter replaces the manual `kubectl ws create`
steps and lets the same binary run as a Kubernetes initContainer.

Option A — seed everything from root in one step (admin kubeconfig pointed at root):

```bash
KUBECONFIG=$PM_KUBECONFIG make init-seed-workspaces HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443
```

By default this creates `providers` (type `root:providers`) and `quickstart` (type
`root:provider`) and bootstraps into the latter. Override with
`--workspace <name>=<type-path>:<type-name>` (repeatable, parent first) and/or
`--parent-workspace`.

Option B — drive the workspaces yourself with `kubectl ws`, then bootstrap content only:

```bash
KUBECONFIG=$PM_KUBECONFIG kubectl ws use :
KUBECONFIG=$PM_KUBECONFIG kubectl ws create providers --type=root:providers --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG kubectl ws create quickstart --type=root:provider --enter --ignore-existing
KUBECONFIG=$PM_KUBECONFIG make init HOST_OVERRIDE=https://frontproxy-front-proxy.platform-mesh-system:8443
```

Either way, this applies all kcp and provider resources to register your provider and
creates a dedicated ServiceAccount and RBAC for the provider workspace.

Once this is done, you should be able to access your provider's APIs through the kcp API and see it registered in the Platform Mesh UI.

### 3. Extract Operator Kubeconfig

Extract the kubeconfig for your provider workspace:

```bash
KUBECONFIG=$PM_KUBECONFIG kubectl get secret wildwest-controller-kubeconfig -n default -o jsonpath='{.data.kubeconfig}' | base64 -d > operator.kubeconfig
```

### 4. Run the Operator Locally (optional)

For local development, run the operator directly:

```bash
KUBECONFIG=./operator.kubeconfig go run ./cmd/wild-west --endpointslice=wildwest.platform-mesh.io
```

### 5. Build and Load Images

Build container images and load them into the kind cluster:

```bash
export IMAGE_TAG=platform-mesh
make images kind-load-all IMAGE_TAG=$IMAGE_TAG
```

### 6. Deploy to Cluster

Create the namespace and the kubeconfig secret for the operator:

```bash
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create namespace provider-cowboys
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl delete secret wildwest-controller-kubeconfig -n provider-cowboys --ignore-not-found
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create secret generic wildwest-controller-kubeconfig \
  --from-file=kubeconfig=./operator.kubeconfig -n provider-cowboys
```

Deploy the controller:

```bash
KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install wildwest-controller ./deploy/helm/wildwest-controller \
  --namespace provider-cowboys \
  --set image.tag=$IMAGE_TAG \
  --set image.pullPolicy=IfNotPresent \
  --set common.defaults.hostAliases.enabled=true
```

Optionally, enable the bootstrap initContainer to (re)apply provider resources on
each deploy. It needs its own secret containing an admin kubeconfig (separate from
the controller kubeconfig) — pointed at the provider workspace, or at root if you
also want it to seed the workspace hierarchy:

```bash
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl create secret generic wildwest-init-kubeconfig \
  --from-file=kubeconfig=$PM_KUBECONFIG -n provider-cowboys

KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install wildwest-controller ./deploy/helm/wildwest-controller \
  --namespace provider-cowboys \
  --set image.tag=$IMAGE_TAG \
  --set image.pullPolicy=IfNotPresent \
  --set common.defaults.hostAliases.enabled=true \
  --set init.enabled=true \
  --set init.kubeconfig.secretName=wildwest-init-kubeconfig \
  --set init.seedWorkspaces=true \
  --set init.hostOverride=https://frontproxy-front-proxy.platform-mesh-system:8443
```

Deploy the armament-sync controller (runs in the provider workspace and syncs the catalog from an external source — currently a static hardcoded list — into `Armament` CRs that are then exposed read-only to consumer workspaces via a `CachedResource`). It ships as its own image (`provider-quickstart-armament-sync`), built and loaded by `make images kind-load-all`:

```bash
KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install wildwest-armament-sync ./deploy/helm/wildwest-armament-sync \
  --namespace provider-cowboys \
  --set image.tag=$IMAGE_TAG \
  --set image.pullPolicy=IfNotPresent \
  --set common.defaults.hostAliases.enabled=true
```
kui
Deploy the portal microfrontend:

```bash
KUBECONFIG=$COMPUTE_KUBECONFIG helm upgrade --install wildwest-portal ./deploy/helm/wildwest-portal \
  --namespace provider-cowboys \
  --set image.tag=$IMAGE_TAG \
  --set image.pullPolicy=IfNotPresent \
  --set httpRoute.enabled=true \
  --set middleware.enabled=true \
  --set common.defaults.hostAliases.enabled=true
```

To upgrade after rebuilding images:

```bash
make images kind-load-all IMAGE_TAG=$IMAGE_TAG
KUBECONFIG=$COMPUTE_KUBECONFIG kubectl rollout restart deployment -n provider-cowboys
```

### 7. Try It Out: Cowboys with Secret Refs

`Cowboy` is a **cluster-scoped** CRD with an optional `spec.secretRefs[]` field that lists Secrets the cowboy depends on. Because the cowboy itself has no namespace, each reference carries its own `namespace`. The portal microfrontend renders one chip per reference and calls the GraphQL gateway to check whether each Secret actually exists:

- **Green chip** — Secret resolved (`v1.Secret(name, namespace)` returned metadata).
- **Red chip** — Secret missing (NotFound) or inaccessible (RBAC forbidden, network error).
- **Neutral chip** — existence check is in flight (transient on first paint).

Cowboys without `secretRefs` show no chips row at all, so the existing tiles are unchanged.

> **Note:** the snippet below targets a **consumer workspace** that has the `wildwest.platform-mesh.io` APIExport bound — it is **not** the provider workspace from the bootstrap steps above. Today the only supported way to provision and switch into such a workspace is via the **Platform Mesh CLI** (`pm`); plain `kubectl`/`kubectl ws` against the provider workspace will not work because the `Cowboy` API is not served there. Use `pm` to create/select your consumer workspace first, then export its kubeconfig as `KUBECONFIG` and run:

```bash
NAMESPACE=default  # the namespace where the referenced Secret will live

kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: colt-45-permit
  namespace: ${NAMESPACE}
type: Opaque
stringData:
  serial_number: C45-123456
  permit_date: "1881-04-15"
  issued_by: Tombstone Marshal
---
apiVersion: wildwest.platform-mesh.io/v1alpha1
kind: Cowboy
metadata:
  name: billy-the-kid
spec:
  intent: Ride the range and protect the cattle
  secretRefs:
    - name: colt-45-permit          # exists -> green chip
      namespace: ${NAMESPACE}
    - name: missing-saddlebag       # does NOT exist -> red chip
      namespace: ${NAMESPACE}
---
apiVersion: wildwest.platform-mesh.io/v1alpha1
kind: Cowboy
metadata:
  name: lonely-ranger
spec:
  intent: Ride alone
  # no secretRefs -> Secret Refs row is hidden in the UI
EOF
```

Open the Cowboys page in the Portal and refresh. The `billy-the-kid` tile shows one green chip (`colt-45-permit`) and one red chip (`missing-saddlebag`); `lonely-ranger` shows no Secret Refs row.

Clean up:

```bash
kubectl delete cowboy billy-the-kid lonely-ranger
kubectl delete -n "$NAMESPACE" secret colt-45-permit
```

### 8. Try It Out: Armaments Catalog (CachedResource)

`Armament` is a cluster-scoped catalog type populated by the `armament-sync` controller from an external source (currently a static hardcoded list in `pkg/external/static`). The catalog lives in the **provider workspace** and is replicated to consumers read-only via a kcp `CachedResource` bound to the `wildwest.platform-mesh.io` APIExport.

Architecture:

```
External source ──poll(ticker)──> armament-sync ──writes──> Armament CRs (provider workspace)
                                                                  │
                                                           CachedResource
                                                                  │
                                                                  ▼
                                                consumer workspaces (read-only)
                                                                  │
                                                                  ▼
                                              Cowboy.spec.armamentRef → name lookup
```

Two binaries, deployed independently:

| Binary | Workspace | Role |
|--------|-----------|------|
| `wild-west` | consumer (via APIExport endpoint slice) | Reconciles `Cowboy` objects users create |
| `armament-sync` | provider (direct kubeconfig) | Pulls the external catalog on a timer, upserts/deletes `Armament` CRs |

Verify in the **provider workspace** that armaments appear after the syncer's first tick:

```bash
KUBECONFIG=./operator.kubeconfig kubectl get armaments
# NAME              KIND       DAMAGE   RANGE
# bowie-knife       blade      30       2
# colt-saa          revolver   50       50
# lasso             rope       5        10
# winchester-1873   rifle      80       400
```

In a **consumer workspace** (one that has bound the `wildwest.platform-mesh.io` APIExport), the same list is visible read-only and can be referenced from a `Cowboy`:

```bash
kubectl apply -f - <<EOF
apiVersion: wildwest.platform-mesh.io/v1alpha1
kind: Cowboy
metadata:
  name: armed-pete
spec:
  intent: Patrol the canyon
  armamentRef:
    name: winchester-1873
EOF
```

Attempting to `kubectl edit armament` from the consumer workspace will fail — the cached resource is read-only. To change the catalog, modify the external source (today: edit `pkg/external/static/client.go` and rebuild) or swap the static client for a real backend implementing `external.Client`.

## Debugging

Assuming your provider workspace is `quickstart` under the `providers` tree:

### Check Marketplace Entries

View your provider's marketplace entry (combines APIExport + ProviderMetadata):

```bash
kubectl --server="https://localhost:8443/services/marketplace/clusters/root:providers:quickstart" get marketplaceentries -A
kubectl --server="https://localhost:8443/services/marketplace/clusters/root:providers:quickstart" get marketplaceentries -A -o yaml
```

### Check Content Configurations

View available API resources and content configurations:

```bash
kubectl --server="https://localhost:8443/services/contentconfigurations/clusters/root:providers:quickstart" api-resources
kubectl --server="https://localhost:8443/services/contentconfigurations/clusters/root:providers:quickstart" get contentconfigurations -A
kubectl --server="https://localhost:8443/services/contentconfigurations/clusters/root:providers:quickstart" get contentconfigurations -A -o yaml
```

### URL Pattern

The server URL follows this pattern:
```
https://<host>/services/<virtual-workspace>/clusters/root:providers:<provider-workspace>
```

Where:
- `marketplace` - Virtual workspace for marketplace entries
- `contentconfigurations` - Virtual workspace for UI content configurations
- `<provider-workspace>` - Your provider workspace name (e.g., `quickstart`)

## Code Generation Tools

This project uses two key code generation tools:

### controller-gen

[controller-gen](https://github.com/kubernetes-sigs/controller-tools) is a Kubernetes code generator that:
- Generates **CRD manifests** from Go type definitions with `+kubebuilder` markers
- Generates **DeepCopy methods** (`zz_generated.deepcopy.go`) required for all Kubernetes API types
- Part of the controller-tools project from Kubernetes SIGs

### apigen

[apigen](https://github.com/kcp-dev/sdk) is a kcp-specific tool that:
- Converts standard Kubernetes **CRDs into APIResourceSchemas** for kcp
- APIResourceSchemas are kcp's way of defining API types that can be exported via `APIExport`
- Takes CRDs from `config/crds/` and outputs APIResourceSchemas to `config/kcp/`

**Generation flow:**
```
Go types (apis/) → controller-gen → CRDs (config/crds/) → apigen → APIResourceSchemas (config/kcp/)
```

## Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build all binaries (operator + init + armament-sync) |
| `make build-operator` | Build the wild-west operator binary |
| `make build-init` | Build the init/bootstrap binary |
| `make build-armament-sync` | Build the armament-sync controller binary |
| `make run` | Run the wild-west operator locally |
| `make run-armament-sync` | Run the armament-sync controller locally |
| `make init` | Bootstrap provider resources into the workspace pointed to by KUBECONFIG (optional HOST_OVERRIDE) |
| `make init-seed-workspaces` | Create the provider workspace hierarchy from the admin kubeconfig, then bootstrap (optional HOST_OVERRIDE) |
| `make generate` | Generate code (deepcopy) and kcp resources |
| `make manifests` | Generate CRD manifests from Go types |
| `make apiresourceschemas` | Generate APIResourceSchemas from CRDs |
| `make image-build` | Build controller container image |
| `make portal-image-build` | Build portal container image |
| `make armament-sync-image-build` | Build armament-sync container image |
| `make images` | Build all container images (controller + portal + armament-sync) |
| `make kind-load` | Load controller image into kind cluster |
| `make kind-load-portal` | Load portal image into kind cluster |
| `make kind-load-armament-sync` | Load armament-sync image into kind cluster |
| `make kind-load-all` | Load all images into kind cluster |
| `make tools` | Install all required tools (controller-gen, apigen) |
| `make fmt` | Run go fmt |
| `make vet` | Run go vet |
| `make tidy` | Run go mod tidy |
| `make help` | Display help for all targets |

## Creating Your Own Provider

1. Fork this repo
2. Update the API group name (replace `wildwest.platform-mesh.io`)
3. Define your CRD schema in `config/kcp/`
4. Update `ProviderMetadata` with your provider details
5. Configure `ContentConfiguration` for your resource UI
6. Update RBAC to allow binding to your APIExport


## Custom Provider UI (Microfrontend)

Platform Mesh supports custom UIs for providers via microfrontends. This is useful when:

- Table views aren't sufficient for your resource representation
- You need custom wizards or multi-step flows (e.g., VM creation with SSH keys)
- You want to orchestrate multiple resources in a single view
- You need custom visualizations beyond standard lists

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                  Platform Mesh Portal                        │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   Luigi Shell                        │    │
│  │                                                      │    │
│  │   Your MFE receives from Luigi context:             │    │
│  │   - token (Bearer auth for API calls)               │    │
│  │   - portalContext.crdGatewayApiUrl (GraphQL API)    │    │
│  │   - accountId (current account context)             │    │
│  │                                                      │    │
│  │   Your MFE can then:                                │    │
│  │   - Query/mutate K8s resources via GraphQL          │    │
│  │   - Use Luigi UX manager for alerts/dialogs         │    │
│  │   - Navigate within the portal                      │    │
│  │                                                      │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Quick Start

1. **Generate a new microfrontend:**
   ```bash
   git clone https://github.com/openmfp/create-micro-frontend
   cd create-micro-frontend
   npm install && npm run build
   npx create-micro-frontend portal -y
   ```

2. **Run locally:**
   ```bash
   cd portal
   npm install
   npm start
   ```
   This serves your MFE at `http://localhost:4200`

3. **Enable local development mode** in the Portal to load your local MFE

4. **Deploy** by hosting the built files and creating a `ContentConfiguration` resource

### Example Implementation

This repo includes a complete working example in the `portal/` directory showing:

- Luigi context integration for auth and API access
- GraphQL queries/mutations for Kubernetes resources
- SAP UI5 web components for consistent Portal styling
- Local development proxy configuration

See [portal/README.md](portal/README.md) for detailed documentation.

### Key Integration Points

**1. Luigi Context (auth & API URLs):**
```typescript
import { LuigiContextService } from '@luigi-project/client-support-angular';
import LuigiClient from '@luigi-project/client';

// Wait for Luigi handshake before making API calls
LuigiClient.addInitListener(() => {
  const context = luigiContextService.getContext();
  const token = context.token;  // Bearer token
  const apiUrl = context.portalContext.crdGatewayApiUrl;  // GraphQL endpoint
});
```

**2. GraphQL API for K8s resources:**
```graphql
query ListMyResources {
  my_api_group_io {
    v1alpha1 {
      MyResources {
        items { metadata { name } spec { ... } }
      }
    }
  }
}
```

**3. ContentConfiguration (register your MFE):**
```yaml
apiVersion: ui.platform-mesh.io/v1alpha1
kind: ContentConfiguration
metadata:
  labels:
    ui.platform-mesh.io/content-for: my-api.platform-mesh.io  # Links to your APIExport
  name: my-ui
spec:
  inlineConfiguration:
    contentType: json
    content: |
      {
        "name": "my-ui",
        "luigiConfigFragment": {
          "data": {
            "nodes": [{
              "pathSegment": "my-resources",
              "label": "My Resources",
              "entityType": "main.core_platform-mesh_io_account:1",
              "url": "https://your-mfe-host/index.html"
            }]
          }
        }
      }
```

### Navigation Categories

Group your MFE under a category in the sidebar:

```json
{
  "category": { "label": "Providers", "icon": "customize", "collapsible": true },
  "pathSegment": "my-resources",
  "label": "My Resources",
  ...
}
```

See [Luigi navigation docs](https://docs.luigi-project.io/docs/navigation-configuration?section=category) for more options.

### Running as a Provider

As a provider, you are responsible for hosting your microfrontend (similar to running your operator). The MFE needs to be accessible to Portal users. Options include:

- Static hosting (S3, GCS, GitHub Pages, etc.)
- Container deployment alongside your operator
- Any web server that can serve static files

<p align="center"><img alt="Bundesministerium für Wirtschaft und Energie (BMWE)-EU funding logo" src="https://apeirora.eu/assets/img/BMWK-EU.png" width="400"/></p>
