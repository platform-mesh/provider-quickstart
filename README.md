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
├── cmd/init/              # Bootstrap CLI tool
├── config/
│   ├── kcp/               # kcp resources (APIExport, APIResourceSchema)
│   └── provider/          # Provider resources (ProviderMetadata, ContentConfiguration, RBAC)
└── pkg/bootstrap/         # Bootstrap logic for applying resources
```

## Usage

1. Set your kubeconfig:
   ```bash
   export KUBECONFIG=/path/to/kcp/admin.kubeconfig
   ```

2. Build and run the bootstrap:
   ```bash
   make build
   bin/wild-west-init
   ```

This applies all kcp and provider resources to register your provider.

## Debugging

Assuming your organization is `bob` and provider account is `quickstart`:

### Check Marketplace Entries

View your provider's marketplace entry (combines APIExport + ProviderMetadata):

```bash
kubectl --server="https://localhost:8443/services/marketplace/clusters/root:orgs:bob:quickstart" get marketplaceentries -A
kubectl --server="https://localhost:8443/services/marketplace/clusters/root:orgs:bob:quickstart" get marketplaceentries -A -o yaml
```

### Check Content Configurations

View available API resources and content configurations:

```bash
kubectl --server="https://localhost:8443/services/contentconfigurations/clusters/root:orgs:bob:quickstart" api-resources
kubectl --server="https://localhost:8443/services/contentconfigurations/clusters/root:orgs:bob:quickstart" get contentconfigurations -A
kubectl --server="https://localhost:8443/services/contentconfigurations/clusters/root:orgs:bob:quickstart" get contentconfigurations -A -o yaml
```

### URL Pattern

The server URL follows this pattern:
```
https://<host>/services/<virtual-workspace>/clusters/root:orgs:<org>:<account>
```

Where:
- `marketplace` - Virtual workspace for marketplace entries
- `contentconfigurations` - Virtual workspace for UI content configurations
- `<org>` - Your organization name (e.g., `bob`)
- `<account>` - Your provider account name (e.g., `quickstart`)

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
