# Platform Mesh Provider Quickstart

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
│   └── wild-west/         # Provider operator
├── config/
│   ├── kcp/               # kcp resources (APIExport, APIResourceSchema)
│   └── provider/          # Provider resources (ProviderMetadata, ContentConfiguration, RBAC)
├── pkg/bootstrap/         # Bootstrap logic for applying resources
└── portal/                # Custom UI microfrontend example (Angular + Luigi)
```

## Usage

> **Important:** Providers must live in a dedicated workspace type within a separate tree. This means platform administrators must configure providers using the **admin kubeconfig**. Regular user kubeconfigs will not have the necessary permissions to create provider workspaces. This is bound to change and improve in the future, but for now you must use the admin kubeconfig to set up your provider.

### 1. Set Admin Kubeconfig

You need the admin kubeconfig to create and manage provider workspaces:

```bash
export KUBECONFIG=/path/to/kcp/admin.kubeconfig
```

### 2. Create Provider Workspace Hierarchy

Navigate to the root workspace and create the provider workspace structure:

```bash
# Navigate to root workspace
kubectl ws use :

# Create the providers parent workspace (if it doesn't exist)
kubectl ws create providers --type=root:providers --enter --ignore-existing

# Create your provider workspace
kubectl ws create quickstart --type=root:provider --enter --ignore-existing
```

### 3. Bootstrap Provider Resources

Build and run the bootstrap to register your provider:

```bash
make init
```

This applies all kcp and provider resources to register your provider and created dedicated 
ServiceAccount and RBAC for the provider workspace.

Once this is done, you should be able to access your provider's APIs through the kcp API and see it registered in the Platform Mesh UI.

### 4. Run the Operator

Extract the kubeconfig for your provider workspace and run the operator locally:

```bash
kubectl get secret wildwest-controller-kubeconfig -n default -o jsonpath='{.data.kubeconfig}' | base64 -d > operator.kubeconfig
```

Run the operator from your local machine using the extracted kubeconfig:

```bash
KUBECONFIG=./operator.kubeconfig go run ./cmd/wild-west --endpointslice=wildwest.platform-mesh.io
```

Running in the pod:

```bash
kubectl create namespace provider-cowboys 
kubectl create secret generic wildwest-controller-kubeconfig \
  --from-file=kubeconfig=./operator.kubeconfig -n provider-cowboys

helm install wildwest-controller ./deploy/helm/wildwest-controller \
  --namespace provider-cowboys \
  --set image.tag=0.0.1-rc2
```

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
| `make build` | Build all binaries (operator + init) |
| `make build-operator` | Build the wild-west operator binary |
| `make build-init` | Build the init/bootstrap binary |
| `make run` | Run the wild-west operator locally |
| `make init` | Bootstrap provider resources into workspace (requires KUBECONFIG) |
| `make generate` | Generate code (deepcopy) and kcp resources |
| `make manifests` | Generate CRD manifests from Go types |
| `make apiresourceschemas` | Generate APIResourceSchemas from CRDs |
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
   npx create-micro-frontend my-provider-ui -y
   ```

2. **Run locally:**
   ```bash
   cd my-provider-ui
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