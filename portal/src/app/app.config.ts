import { ApplicationConfig } from '@angular/core';
import {
  LuigiContextService,
  LuigiContextServiceImpl,
} from '@luigi-project/client-support-angular';

export const appConfig: ApplicationConfig = {
  providers: [
    {
      provide: LuigiContextService,
      useClass: LuigiContextServiceImpl,
    },
  ],
};

