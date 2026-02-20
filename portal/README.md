# Custom Provider UI - Microfrontend Example

This is an example Angular microfrontend that demonstrates how to build a custom UI for your Platform Mesh provider. It shows how to integrate with the Luigi shell, authenticate API calls, and manage Kubernetes custom resources via GraphQL.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Platform Mesh Portal                         │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Luigi Shell                            │   │
│  │  ┌─────────────────┐  ┌─────────────────────────────┐    │   │
│  │  │   Navigation    │  │      Your Microfrontend     │    │   │
│  │  │                 │  │  ┌─────────────────────────┐│    │   │
│  │  │ - Dashboard     │  │  │  CowboysComponent       ││    │   │
│  │  │ - Settings      │  │  │                         ││    │   │
│  │  │ - Cowboys  ◄────┼──┼──│  Receives context:      ││    │   │
│  │  │   (your MFE)    │  │  │  - auth token           ││    │   │
│  │  │                 │  │  │  - API URLs             ││    │   │
│  │  └─────────────────┘  │  │  - account info         ││    │   │
│  │                       │  └─────────────────────────┘│    │   │
│  │                       └─────────────────────────────────┘    │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              CRD GraphQL Gateway                          │   │
│  │         /api/kubernetes-graphql-gateway/...              │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Key Concepts

### 1. Luigi Context Integration

The Portal uses [Luigi](https://luigi-project.io/) as its microfrontend framework. Your microfrontend receives context from Luigi containing:

- **`token`** - Bearer token for API authentication (user's session)
- **`portalContext.crdGatewayApiUrl`** - GraphQL endpoint for Kubernetes resources
- **`accountId`** - Current account context

```typescript
// In your component
import { LuigiContextService } from '@luigi-project/client-support-angular';

@Component({...})
export class MyComponent {
  private luigiContextService = inject(LuigiContextService);

  // Convert to Angular signal for reactivity
  public luigiContext = toSignal(this.luigiContextService.contextObservable());

  ngOnInit() {
    // IMPORTANT: Wait for Luigi handshake before making API calls
    LuigiClient.addInitListener(() => {
      this.loadData();
    });
  }
}
```

### 2. GraphQL API Access

The Platform Mesh CRD Gateway exposes your Kubernetes resources via GraphQL. The schema is auto-generated from your CRDs:

```graphql
# Query pattern: {apiGroup}_{version}_{Kind}s
query ListCowboys {
  wildwest_platform_mesh_io {
    v1alpha1 {
      Cowboys {
        items {
          metadata { name namespace }
          spec { intent }
          status { result }
        }
      }
    }
  }
}

# Mutation pattern: create{Kind}(namespace, object)
mutation CreateCowboy($name: String!, $namespace: String!) {
  wildwest_platform_mesh_io {
    v1alpha1 {
      createCowboy(namespace: $namespace, object: { metadata: { name: $name } }) {
        metadata { name }
      }
    }
  }
}
```

### 3. Content Configuration

Your microfrontend registers with the Portal via a `ContentConfiguration` resource:

```yaml
apiVersion: ui.platform-mesh.io/v1alpha1
kind: ContentConfiguration
metadata:
  labels:
    ui.platform-mesh.io/content-for: wildwest.platform-mesh.io  # Links to your APIExport
  name: cowboys-ui
spec:
  inlineConfiguration:
    contentType: json
    content: |
      {
        "name": "cowboys",
        "luigiConfigFragment": {
          "data": {
            "nodes": [{
              "pathSegment": "cowboys",
              "label": "Cowboys",
              "entityType": "main.core_platform-mesh_io_account:1",
              "url": "https://your-mfe-host/index.html",
              "icon": "person-placeholder",
              "context": {
                "accountId": ":core_platform-mesh_io_accountId"
              }
            }]
          }
        }
      }
```

Key fields:
- **`entityType`**: Where in the navigation tree to show your MFE
  - `main` - Root level
  - `main.core_platform-mesh_io_account:1` - Under account level
- **`url`**: URL where your microfrontend is hosted
- **`context`**: Variables passed to your MFE from the URL

## Project Structure

```
portal/
├── public/
│   └── content-configuration.json   # Luigi navigation config (local dev)
├── src/
│   ├── app/
│   │   ├── app.component.ts         # Root component
│   │   ├── app.config.ts            # Angular DI config (LuigiContextService)
│   │   └── cowboys/
│   │       ├── cowboys.component.ts    # Main UI component
│   │       ├── cowboys.component.html  # Template with UI5 components
│   │       ├── cowboys.component.scss  # Styles
│   │       └── cowboys.service.ts      # GraphQL API service
│   └── styles.scss                  # Global styles
├── proxy.conf.json                  # Dev proxy for API calls
└── angular.json                     # Angular configuration
```

## Local Development

### Prerequisites

- Node.js 18+
- Platform Mesh local setup running (see main repo README)
- Your provider registered and bound to an account

### Setup

```bash
cd portal
npm install
```

### Running Locally

1. **Start the dev server:**
   ```bash
   npm start
   ```
   This serves the microfrontend at `http://localhost:4200`

2. **Enable local development mode in Portal:**
   - Open the Portal UI in your browser
   - Enable local development mode (usually via developer tools or settings)
   - The Portal will load your local MFE instead of the deployed version

3. **Configure proxy for API calls:**

   The `proxy.conf.mjs` routes `/api` requests to the Platform Mesh gateway:
   ```javascript
   export default {
     '/api': {
       target: 'https://bob.portal.localhost:8443',
       secure: false,
       changeOrigin: true
     }
   };
   ```

   Update the `target` to match your local Portal URL.

   **Important:** After changing proxy settings, restart the dev server.

### How Local Development Mode Works

1. Your MFE serves `content-configuration.json` at its root
2. Portal fetches this and merges it with server-side configs
3. Luigi renders your MFE in an iframe
4. Context (token, URLs) flows from Portal to your MFE

## Deployment

For production, you need to:

1. **Build the microfrontend:**
   ```bash
   npm run build
   ```

2. **Host the built files** on a web server accessible to Portal users

3. **Create a ContentConfiguration** resource pointing to your hosted URL:
   ```yaml
   apiVersion: ui.platform-mesh.io/v1alpha1
   kind: ContentConfiguration
   metadata:
     labels:
       ui.platform-mesh.io/content-for: wildwest.platform-mesh.io
     name: cowboys-ui
   spec:
     inlineConfiguration:
       contentType: json
       content: |
         {
           "name": "cowboys",
           "luigiConfigFragment": {
             "data": {
               "nodes": [{
                 "pathSegment": "cowboys",
                 "label": "Cowboys",
                 "url": "https://your-production-host/index.html",
                 ...
               }]
             }
           }
         }
   ```

## UI5 Web Components

This example uses [SAP UI5 Web Components](https://sap.github.io/ui5-webcomponents/) for consistent styling with the Portal. Key components used:

- `ui5-dynamic-page` - Page layout with collapsible header
- `ui5-avatar` - User/resource icons
- `ui5-toolbar` - Action bars
- `ui5-button` - Buttons
- `ui5-dialog` - Modal dialogs
- `ui5-input`, `ui5-select` - Form controls
- `ui5-title`, `ui5-label`, `ui5-text` - Typography

Browse available components: https://sap.github.io/ui5-webcomponents/playground/

## Luigi Client API

Common Luigi client methods:

```typescript
import LuigiClient from '@luigi-project/client';

// Wait for initialization
LuigiClient.addInitListener(() => { ... });

// Show/hide loading indicator
LuigiClient.uxManager().showLoadingIndicator();
LuigiClient.uxManager().hideLoadingIndicator();

// Show alerts
LuigiClient.uxManager().showAlert({
  text: 'Operation successful',
  type: 'success',  // 'success' | 'info' | 'warning' | 'error'
  closeAfter: 3000
});

// Show confirmation dialog
LuigiClient.uxManager().showConfirmationModal({
  header: 'Confirm Delete',
  body: 'Are you sure?',
  buttonConfirm: 'Delete',
  buttonDismiss: 'Cancel'
}).then(() => { /* confirmed */ }).catch(() => { /* cancelled */ });

// Navigate within Portal
LuigiClient.linkManager().navigate('/path/to/page');
```

Full API docs: https://docs.luigi-project.io/docs/luigi-client-api

## Troubleshooting

### CORS Errors

When running locally, you may see CORS errors. Solutions:

1. **Use the Angular proxy** - Ensure `proxy.conf.json` is configured and `ng serve` uses it
2. **Update Portal CORS settings** - Add `http://localhost:4200` to allowed origins

### Context Not Available

If `luigiContext` is empty:

1. Ensure you wait for `LuigiClient.addInitListener()` before accessing context
2. Check that local development mode is enabled in Portal
3. Verify your `content-configuration.json` is being served correctly

### GraphQL Errors

1. Check the GraphQL endpoint URL in browser dev tools
2. Verify the auth token is being sent in the `Authorization` header
3. Use GraphQL introspection to discover available types:
   ```graphql
   query { __schema { types { name } } }
   ```

## Further Reading

- [Luigi Project Documentation](https://docs.luigi-project.io/)
- [OpenMFP Documentation](https://openmfp.org/documentation/)
- [SAP UI5 Web Components](https://sap.github.io/ui5-webcomponents/)
- [Angular Signals](https://angular.dev/guide/signals)
