/**
 * Cowboys Service - GraphQL API Client
 *
 * This service demonstrates how to make authenticated GraphQL API calls
 * to the Platform Mesh CRD Gateway from a microfrontend.
 *
 * Key integration points:
 * 1. LuigiContextService - Provides the portal context containing auth token and API URLs
 * 2. context.token - Bearer token for API authentication (from user's session)
 * 3. context.portalContext.crdGatewayApiUrl - GraphQL endpoint for Kubernetes resources
 *
 * The GraphQL schema is auto-generated from your Kubernetes CRDs.
 * Query structure follows: {apiGroup}_{version}_{Kind}
 * Example: wildwest_platform_mesh_io.v1alpha1.Cowboys
 */
import { Injectable, inject } from '@angular/core';
import { LuigiContextService } from '@luigi-project/client-support-angular';
import { from, map, Observable, of, switchMap, catchError, filter } from 'rxjs';

export interface Cowboy {
  metadata: {
    name: string;
    namespace?: string;
    creationTimestamp?: string;
  };
  spec: {
    intent?: string;
  };
  status?: {
    result?: string;
  };
}

export interface Namespace {
  metadata: {
    name: string;
  };
}

export interface CowboyListResponse {
  wildwest_platform_mesh_io: {
    v1alpha1: {
      Cowboys: {
        items: Cowboy[];
      };
    };
  };
}

export interface NamespaceListResponse {
  v1: {
    Namespaces: {
      items: Namespace[];
    };
  };
}

const LIST_COWBOYS_QUERY = `
  query ListCowboys {
    wildwest_platform_mesh_io {
      v1alpha1 {
        Cowboys {
          items {
            metadata {
              name
              namespace
              creationTimestamp
            }
            spec {
              intent
            }
            status {
              result
            }
          }
        }
      }
    }
  }
`;

const LIST_NAMESPACES_QUERY = `
  query ListNamespaces {
    v1 {
      Namespaces {
        items {
          metadata {
            name
          }
        }
      }
    }
  }
`;

const CREATE_COWBOY_MUTATION = `
  mutation CreateCowboy($name: String!, $namespace: String!, $intent: String) {
    wildwest_platform_mesh_io {
      v1alpha1 {
        createCowboy(
          namespace: $namespace
          object: {
            metadata: { name: $name }
            spec: { intent: $intent }
          }
        ) {
          metadata {
            name
            namespace
          }
        }
      }
    }
  }
`;

const DELETE_COWBOY_MUTATION = `
  mutation DeleteCowboy($name: String!) {
    wildwest_platform_mesh_io {
      v1alpha1 {
        deleteCowboy(name: $name)
      }
    }
  }
`;

interface GraphQLConfig {
  endpoint: string;
  token: string | null;
}

@Injectable({ providedIn: 'root' })
export class CowboysService {
  private luigiContextService = inject(LuigiContextService);

  /**
   * Extracts GraphQL endpoint and auth token from the Luigi context.
   *
   * The portal context provides:
   * - token: Bearer token for API authentication
   * - portalContext.crdGatewayApiUrl: Full URL to the GraphQL CRD gateway
   *
   * For local development, we convert the absolute URL to a relative path
   * so Angular's proxy can forward requests (avoids CORS issues).
   */
  private getGraphQLConfig(): Observable<GraphQLConfig> {
    return this.luigiContextService.contextObservable().pipe(
      filter((ctx) => !!ctx?.context),
      map((ctx) => {
        const context = ctx.context as any;

        // Extract auth token from Luigi context
        const token = context.token || null;

        // Get the CRD GraphQL gateway URL from portal context
        let endpoint = context.portalContext?.crdGatewayApiUrl;
        if (!endpoint) {
          console.warn('crdGatewayApiUrl not found in context, falling back to default');
          endpoint = context.portalBaseUrl + '/graphql';
        }

        // LOCAL DEV: Convert absolute URL to relative path for Angular proxy
        // This is only needed when running locally with `ng serve`
        // In production, the microfrontend runs on the same domain as the portal
        try {
          const url = new URL(endpoint);
          const currentOrigin = window.location.origin;
          if (currentOrigin.includes('localhost:4200') || currentOrigin.includes('localhost:8080')) {
            endpoint = url.pathname; // Use path only, proxy handles the rest
          }
        } catch {
          // URL parsing failed, use endpoint as-is
        }

        return { endpoint, token };
      })
    );
  }

  /**
   * Build HTTP headers for GraphQL requests.
   * Includes Bearer token if available from the portal context.
   */
  private buildHeaders(token: string | null): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  }

  /**
   * List all Cowboys from the Kubernetes cluster.
   * Uses the GraphQL query pattern: {apiGroup}.{version}.{Kind}s
   */
  listCowboys(): Observable<Cowboy[]> {
    return this.getGraphQLConfig().pipe(
      switchMap(({ endpoint, token }) =>
        from(
          fetch(endpoint, {
            method: 'POST',
            headers: this.buildHeaders(token),
            credentials: 'include',
            body: JSON.stringify({
              query: LIST_COWBOYS_QUERY,
            }),
          }).then((res) => res.json())
        )
      ),
      map((response: { data: CowboyListResponse }) => {
        return response.data?.wildwest_platform_mesh_io?.v1alpha1?.Cowboys?.items || [];
      }),
      catchError((error) => {
        console.error('Error fetching cowboys:', error);
        return of([]);
      })
    );
  }

  /**
   * List all Namespaces available to the user.
   * Core K8s resources use pattern: v1.{Kind}s
   */
  listNamespaces(): Observable<Namespace[]> {
    return this.getGraphQLConfig().pipe(
      switchMap(({ endpoint, token }) =>
        from(
          fetch(endpoint, {
            method: 'POST',
            headers: this.buildHeaders(token),
            credentials: 'include',
            body: JSON.stringify({
              query: LIST_NAMESPACES_QUERY,
            }),
          }).then((res) => res.json())
        )
      ),
      map((response: { data: NamespaceListResponse }) => {
        return response.data?.v1?.Namespaces?.items || [];
      }),
      catchError((error) => {
        console.error('Error fetching namespaces:', error);
        return of([]);
      })
    );
  }

  /**
   * Create a new Cowboy resource in the specified namespace.
   * Uses GraphQL mutation pattern: create{Kind}(namespace, object)
   */
  createCowboy(name: string, namespace: string, intent?: string): Observable<boolean> {
    return this.getGraphQLConfig().pipe(
      switchMap(({ endpoint, token }) =>
        from(
          fetch(endpoint, {
            method: 'POST',
            headers: this.buildHeaders(token),
            credentials: 'include',
            body: JSON.stringify({
              query: CREATE_COWBOY_MUTATION,
              variables: { name, namespace, intent },
            }),
          }).then((res) => res.json())
        )
      ),
      map((response: any) => {
        return !!response.data?.wildwest_platform_mesh_io?.v1alpha1?.createCowboy;
      }),
      catchError((error) => {
        console.error('Error creating cowboy:', error);
        return of(false);
      })
    );
  }

  /**
   * Delete a Cowboy resource by name.
   * Uses GraphQL mutation pattern: delete{Kind}(name)
   */
  deleteCowboy(name: string): Observable<boolean> {
    return this.getGraphQLConfig().pipe(
      switchMap(({ endpoint, token }) =>
        from(
          fetch(endpoint, {
            method: 'POST',
            headers: this.buildHeaders(token),
            credentials: 'include',
            body: JSON.stringify({
              query: DELETE_COWBOY_MUTATION,
              variables: { name },
            }),
          }).then((res) => res.json())
        )
      ),
      map((response: any) => {
        return !!response.data?.wildwest_platform_mesh_io?.v1alpha1?.deleteCowboy;
      }),
      catchError((error) => {
        console.error('Error deleting cowboy:', error);
        return of(false);
      })
    );
  }
}
