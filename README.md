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


# Micro frontend

Follow: https://openmfp.org/documentation/getting-started/installation

```bash
git clone https://github.com/openmfp/create-micro-frontend 
cd create-micro-frontend
npm i
npm run build
```

Once it build you can do:

```bash
npx create-micro-frontend portal -y
mv portal ../
npm install
npm start
```