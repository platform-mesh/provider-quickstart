import { Component } from "@angular/core";
import { CowboysComponent } from "./cowboys/cowboys.component";

@Component({
  selector: "app-root",
  standalone: true,
  imports: [CowboysComponent],
  template: `<app-cowboys></app-cowboys>`,
  styles: ``,
})
export class AppComponent {}
