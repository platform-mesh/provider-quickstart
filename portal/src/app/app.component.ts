import { Component } from "@angular/core";
import { ExamplePageComponent } from "./example-page/example-page.component";

@Component({
  selector: "app-root",
  standalone: true,
  imports: [ExamplePageComponent],
  template: `<app-example-page></app-example-page>`,
  styles: ``,
})
export class AppComponent {}
