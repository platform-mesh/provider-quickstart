/**
 * Cowboys Management Page Component
 *
 * This is an example microfrontend that demonstrates how to:
 * 1. Integrate with the Luigi shell via @luigi-project/client
 * 2. Access the Portal context (auth token, API URLs) via LuigiContextService
 * 3. Make GraphQL API calls to manage Kubernetes custom resources
 * 4. Use SAP UI5 web components for a consistent look and feel
 *
 * Key concepts:
 * - LuigiClient.addInitListener() - Wait for Luigi shell handshake before loading data
 * - LuigiContextService - Angular service that provides the portal context as an Observable
 * - context.token - Bearer token for API authentication
 * - context.portalContext.crdGatewayApiUrl - GraphQL endpoint for K8s resources
 */
import { Component, CUSTOM_ELEMENTS_SCHEMA, effect, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import * as LuigiClient from '@luigi-project/client';
import { ILuigiContextTypes, LuigiContextService } from '@luigi-project/client-support-angular';
import {
  AvatarComponent,
  ButtonComponent,
  DialogComponent,
  DynamicPageComponent,
  DynamicPageHeaderComponent,
  DynamicPageTitleComponent,
  IconComponent,
  InputComponent,
  LabelComponent,
  OptionComponent,
  SelectComponent,
  TextComponent,
  TitleComponent,
  ToolbarButtonComponent,
  ToolbarComponent,
} from '@ui5/webcomponents-ngx';

// Import UI5 icons used in the template
import '@ui5/webcomponents-icons/dist/accept.js';
import '@ui5/webcomponents-icons/dist/add.js';
import '@ui5/webcomponents-icons/dist/calendar.js';
import '@ui5/webcomponents-icons/dist/delete.js';
import '@ui5/webcomponents-icons/dist/error.js';
import '@ui5/webcomponents-icons/dist/pending.js';
import '@ui5/webcomponents-icons/dist/person-placeholder.js';
import '@ui5/webcomponents-icons/dist/refresh.js';

import { Armament, Cowboy, CowboysService, Namespace } from './cowboys.service';

@Component({
  selector: 'app-cowboys',
  standalone: true,
  imports: [
    DynamicPageComponent,
    DynamicPageTitleComponent,
    DynamicPageHeaderComponent,
    AvatarComponent,
    TitleComponent,
    LabelComponent,
    TextComponent,
    ToolbarComponent,
    ToolbarButtonComponent,
    IconComponent,
    InputComponent,
    ButtonComponent,
    DialogComponent,
    SelectComponent,
    OptionComponent,
  ],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  templateUrl: './cowboys.component.html',
  styleUrl: './cowboys.component.scss',
})
export class CowboysComponent {
  private luigiContextService = inject(LuigiContextService);
  private cowboysService = inject(CowboysService);

  public luigiContext = toSignal(this.luigiContextService.contextObservable(), {
    initialValue: { context: {}, contextType: ILuigiContextTypes.INIT },
  });

  public cowboys = signal<Cowboy[]>([]);
  public namespaces = signal<Namespace[]>([]);
  public armaments = signal<Armament[]>([]);
  public loading = signal<boolean>(true);
  public showAddDialog = signal<boolean>(false);
  public newCowboyName = signal<string>('');
  public newCowboyNamespace = signal<string>('');
  public newCowboyIntent = signal<string>('');
  // Empty string means "no armament" — the create dialog leaves this unset
  // by default, and the mutation skips the armamentRef field entirely.
  public newCowboyArmament = signal<string>('');

  // Tracks existence of each referenced Secret. Keyed by `${namespace}/${name}`,
  // value is one of 'loading' | 'exists' | 'missing'.
  public secretStatuses = signal<Record<string, 'loading' | 'exists' | 'missing'>>({});

  constructor() {
    // Debug: Log the full Luigi context whenever it changes
    effect(() => {
      const ctx = this.luigiContext();
      console.log('=== LUIGI CONTEXT ===');
      console.log('Full context:', JSON.stringify(ctx.context, null, 2));
      console.log('Context type:', ctx.contextType);
      console.log('=====================');
    });
  }

  /**
   * Initialize the component after Luigi shell handshake completes.
   *
   * IMPORTANT: Always wait for LuigiClient.addInitListener() before making API calls.
   * This ensures the context (auth token, API URLs) is available.
   */
  public ngOnInit(): void {
    LuigiClient.addInitListener(() => {
      // Show Luigi's loading indicator while fetching data
      LuigiClient.uxManager().showLoadingIndicator();
      this.loadNamespaces();
      this.loadArmaments();
      this.loadCowboys();
    });
  }

  public loadArmaments(): void {
    this.cowboysService.listArmaments().subscribe({
      next: (armaments) => this.armaments.set(armaments),
      error: (err) => console.error('Failed to load armaments:', err),
    });
  }

  public loadNamespaces(): void {
    this.cowboysService.listNamespaces().subscribe({
      next: (namespaces) => {
        this.namespaces.set(namespaces);
        // Pre-select first namespace if available
        if (namespaces.length > 0 && !this.newCowboyNamespace()) {
          this.newCowboyNamespace.set(namespaces[0].metadata.name);
        }
      },
      error: (err) => {
        console.error('Failed to load namespaces:', err);
      },
    });
  }

  public loadCowboys(): void {
    this.loading.set(true);
    this.cowboysService.listCowboys().subscribe({
      next: (cowboys) => {
        this.cowboys.set(cowboys);
        this.loading.set(false);
        LuigiClient.uxManager().hideLoadingIndicator();
        this.refreshSecretStatuses(cowboys);
      },
      error: (err) => {
        console.error('Failed to load cowboys:', err);
        this.loading.set(false);
        LuigiClient.uxManager().hideLoadingIndicator();
        LuigiClient.uxManager().showAlert({
          text: 'Failed to load cowboys',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  public openAddDialog(): void {
    this.newCowboyName.set('');
    this.newCowboyIntent.set('');
    this.newCowboyArmament.set('');
    // Pre-select first namespace if available
    if (this.namespaces().length > 0) {
      this.newCowboyNamespace.set(this.namespaces()[0].metadata.name);
    }
    this.showAddDialog.set(true);
  }

  public closeAddDialog(): void {
    this.showAddDialog.set(false);
  }

  public onNameInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.newCowboyName.set(input.value);
  }

  public onIntentInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.newCowboyIntent.set(input.value);
  }

  public onNamespaceChange(event: Event): void {
    const select = event.target as any;
    this.newCowboyNamespace.set(select.selectedOption?.value || '');
  }

  public onArmamentChange(event: Event): void {
    const select = event.target as any;
    this.newCowboyArmament.set(select.selectedOption?.value || '');
  }

  public confirmAddCowboy(): void {
    const name = this.newCowboyName().trim();
    const namespace = this.newCowboyNamespace().trim();
    const intent = this.newCowboyIntent().trim();

    if (!name) {
      LuigiClient.uxManager().showAlert({
        text: 'Please enter a name for the cowboy',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    if (!namespace) {
      LuigiClient.uxManager().showAlert({
        text: 'Please select a namespace',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    const armament = this.newCowboyArmament().trim();
    this.cowboysService.createCowboy(name, namespace, intent || undefined, armament || undefined).subscribe({
      next: (success) => {
        if (success) {
          LuigiClient.uxManager().showAlert({
            text: `Cowboy "${name}" created successfully`,
            type: 'success',
            closeAfter: 3000,
          });
          this.closeAddDialog();
          this.loadCowboys();
        } else {
          LuigiClient.uxManager().showAlert({
            text: 'Failed to create cowboy',
            type: 'error',
            closeAfter: 3000,
          });
        }
      },
      error: () => {
        LuigiClient.uxManager().showAlert({
          text: 'Failed to create cowboy',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  public deleteCowboy(cowboy: Cowboy): void {
    LuigiClient.uxManager()
      .showConfirmationModal({
        type: 'warning',
        header: 'Delete Cowboy',
        body: `Are you sure you want to delete "${cowboy.metadata.name}"?`,
        buttonConfirm: 'Delete',
        buttonDismiss: 'Cancel',
      })
      .then(() => {
        this.cowboysService.deleteCowboy(cowboy.metadata.name, cowboy.metadata.namespace!).subscribe({
          next: (success) => {
            if (success) {
              LuigiClient.uxManager().showAlert({
                text: `Cowboy "${cowboy.metadata.name}" deleted`,
                type: 'success',
                closeAfter: 3000,
              });
              this.loadCowboys();
            } else {
              LuigiClient.uxManager().showAlert({
                text: 'Failed to delete cowboy',
                type: 'error',
                closeAfter: 3000,
              });
            }
          },
          error: () => {
            LuigiClient.uxManager().showAlert({
              text: 'Failed to delete cowboy',
              type: 'error',
              closeAfter: 3000,
            });
          },
        });
      })
      .catch(() => {
        console.log('Cowboy deletion cancelled');
      });
  }

  public secretKey(namespace: string | undefined, name: string): string {
    return `${namespace ?? ''}/${name}`;
  }

  public getSecretStatus(namespace: string | undefined, name: string): 'loading' | 'exists' | 'missing' {
    return this.secretStatuses()[this.secretKey(namespace, name)] ?? 'loading';
  }

  /**
   * Fan out an existence check per unique (namespace, name) referenced by
   * any cowboy. Statuses start as 'loading' and resolve to 'exists' or 'missing'.
   */
  private refreshSecretStatuses(cowboys: Cowboy[]): void {
    const refs = new Map<string, { name: string; namespace: string }>();
    for (const c of cowboys) {
      const namespace = c.metadata.namespace;
      if (!namespace) continue;
      for (const ref of c.spec.secretRefs ?? []) {
        refs.set(this.secretKey(namespace, ref.name), { name: ref.name, namespace });
      }
    }

    if (refs.size === 0) {
      this.secretStatuses.set({});
      return;
    }

    const initial: Record<string, 'loading' | 'exists' | 'missing'> = {};
    for (const key of refs.keys()) {
      initial[key] = 'loading';
    }
    this.secretStatuses.set(initial);

    for (const [key, { name, namespace }] of refs) {
      this.cowboysService.secretExists(name, namespace).subscribe((exists) => {
        this.secretStatuses.update((prev) => ({
          ...prev,
          [key]: exists ? 'exists' : 'missing',
        }));
      });
    }
  }

  /**
   * Look up the catalog Armament for a cowboy's reference. Returns undefined
   * if the catalog hasn't loaded yet or the referenced armament is gone (the
   * controller might race with a sync that just deleted the catalog entry).
   */
  public armamentFor(name: string | undefined): Armament | undefined {
    if (!name) return undefined;
    return this.armaments().find((a) => a.metadata.name === name);
  }

  public getInitials(name: string): string {
    if (!name) return '??';
    const parts = name.split(/[-_\s]+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.substring(0, 2).toUpperCase();
  }

  public getColorScheme(name: string): 'Accent1' | 'Accent2' | 'Accent3' | 'Accent4' | 'Accent5' | 'Accent6' | 'Accent7' | 'Accent8' | 'Accent9' | 'Accent10' {
    const schemes = ['Accent1', 'Accent2', 'Accent3', 'Accent4', 'Accent5', 'Accent6', 'Accent7', 'Accent8', 'Accent9', 'Accent10'] as const;
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    return schemes[Math.abs(hash) % schemes.length];
  }

  public formatDate(timestamp: string | undefined): string {
    if (!timestamp) return 'Unknown';
    try {
      const date = new Date(timestamp);
      return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return timestamp;
    }
  }
}
