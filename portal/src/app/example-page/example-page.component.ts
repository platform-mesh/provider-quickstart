import { Component, CUSTOM_ELEMENTS_SCHEMA, effect, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import LuigiClient from '@luigi-project/client';
import { ILuigiContextTypes, LuigiContextService } from '@luigi-project/client-support-angular';
import {
  AvatarComponent,
  DynamicPageComponent,
  DynamicPageHeaderComponent,
  DynamicPageTitleComponent,
  IconComponent,
  InputComponent,
  LabelComponent,
  LinkComponent,
  TabComponent,
  TabContainerComponent,
  TableCellComponent,
  TableComponent,
  TableHeaderCellComponent,
  TableHeaderRowComponent,
  TableRowComponent,
  TextComponent,
  TitleComponent,
  ToolbarButtonComponent,
  ToolbarComponent,
  ToolbarSpacerComponent,
} from '@ui5/webcomponents-ngx';
import { delay, Observable, of } from 'rxjs';

import '@ui5/webcomponents-icons/dist/action-settings.js';
import '@ui5/webcomponents-icons/dist/filter.js';
import '@ui5/webcomponents-icons/dist/group-2.js';
import '@ui5/webcomponents-icons/dist/search.js';
import '@ui5/webcomponents-icons/dist/shipping-status.js';
import '@ui5/webcomponents-icons/dist/sort.js';

@Component({
  selector: 'app-example-page',
  standalone: true,
  imports: [
    DynamicPageComponent,
    DynamicPageTitleComponent,
    DynamicPageHeaderComponent,
    AvatarComponent,
    TitleComponent,
    LabelComponent,
    TextComponent,
    LinkComponent,
    ToolbarComponent,
    ToolbarButtonComponent,
    ToolbarSpacerComponent,
    TabContainerComponent,
    TabComponent,
    IconComponent,
    InputComponent,
    TableComponent,
    TableHeaderRowComponent,
    TableHeaderCellComponent,
    TableRowComponent,
    TableCellComponent,
  ],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  templateUrl: './example-page.component.html',
  styleUrl: './example-page.component.scss',
})
export class ExamplePageComponent {
  private luigiContextService = inject(LuigiContextService);

  public luigiContext = toSignal(this.luigiContextService.contextObservable(), {
    initialValue: { context: {}, contextType: ILuigiContextTypes.INIT },
  });

  public product$ = signal<any>(null);

  constructor() {
    effect(() => {
      console.log('context updated');
      console.log(this.luigiContext().context);
    });
  }

  public ngOnInit(): void {
    LuigiClient.addInitListener((initialContext: any) => {
      LuigiClient.uxManager().showLoadingIndicator();
      LuigiClient.uxManager().showAlert({
        text: 'Microfrontend initialized on url: ' + initialContext.portalBaseUrl,
        type: 'success',
        closeAfter: 3000,
      });

      this.loadProduct().subscribe((product) => {
        this.product$.set(product);
        LuigiClient.uxManager().hideLoadingIndicator();
      });
    });
  }

  public deleteProduct(): void {
    LuigiClient.uxManager()
      .showConfirmationModal({
        type: 'warning',
        header: 'Delete Product',
        body: 'Are you sure you want to delete this product?',
        buttonConfirm: 'Delete',
        buttonDismiss: 'Cancel',
      })
      .then(() => {
        LuigiClient.uxManager().showAlert({
          text: 'Product deleted',
          type: 'success',
          closeAfter: 3000,
        });
      })
      .catch(() => {
        console.log('Product not deleted');
      });
  }

  private loadProduct(): Observable<any> {
    return of({
      name: 'Robot Arm Series 9',
      objectId: 'PO-48865',
      manufacturer: 'Robotech',
      factory: 'Orlando, Florida',
      supplier: {
        name: 'Robotech',
        id: '234242343',
        display: 'Robotech (234242343)',
      },
      status: {
        text: 'Delivery',
        state: 'success',
      },
      deliveryTime: '12 Days',
      assemblyOption: {
        text: 'To Be Selected',
        state: 'error',
      },
      monthlyLeasingInstalment: {
        number: '379.99',
        unit: 'USD',
      },
      orderDetails: {
        orderId: '589946637',
        contract: '10045876',
        transactionDate: 'May 6, 2018',
        expectedDeliveryDate: 'June 23, 2018',
        factory: 'Orlando, FL',
        supplier: 'Robotech',
      },
      configurationAccounts: {
        model: 'Robot Arm Series 9',
        color: 'White (default)',
        socket: 'Default Socket 10',
        leasingInstalment: '379.99 USD per month',
        axis: '6 Axis',
      },
      products: [
        {
          docNumber: '10223882001820',
          company: 'Jologa',
          contact: 'Denise Smith',
          date: '11/15/19',
          amount: '12,897.00 EUR',
        },
        {
          docNumber: '10223882001821',
          company: 'TechCorp Industries',
          contact: 'Michael Johnson',
          date: '11/18/19',
          amount: '8,450.50 EUR',
        },
        {
          docNumber: '10223882001822',
          company: 'Global Solutions Ltd',
          contact: 'Sarah Williams',
          date: '11/20/19',
          amount: '15,230.75 EUR',
        },
        {
          docNumber: '10223882001823',
          company: 'Advanced Systems Inc',
          contact: 'Robert Brown',
          date: '11/22/19',
          amount: '9,680.25 EUR',
        },
        {
          docNumber: '10223882001824',
          company: 'Innovation Partners',
          contact: 'Emily Davis',
          date: '11/25/19',
          amount: '22,150.00 EUR',
        },
      ],
      contactInformation: {
        phoneNumbers: {
          home: '+ 1 415-321-1234',
          office: '+ 1 415-321-5555',
        },
        socialAccounts: {
          linkedIn: '/DeniseSmith',
          twitter: '@DeniseSmith',
        },
        addresses: {
          home: '2096 Mission Street',
          mailing: 'PO Box 32114',
        },
        mailingAddress: {
          work: 'DeniseSmith@sap.com',
        },
      },
    }).pipe(delay(1500));
  }
}
