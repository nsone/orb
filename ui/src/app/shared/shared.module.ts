import {NgModule} from '@angular/core';
import {MatChipsModule} from '@angular/material/chips';
import {MatIconModule} from '@angular/material/icon';

import {ThemeModule} from 'app/@theme/theme.module';
import {
  NbButtonModule,
  NbCardModule,
  NbCheckboxModule,
  NbDatepickerModule,
  NbIconModule,
  NbInputModule,
  NbSelectModule,
} from '@nebular/theme';
import {FormsModule} from '@angular/forms';

import {MapModule} from './components/map/map.module';
import {ConfirmationComponent} from './components/confirmation/confirmation.component';
import {ChartModule} from './components/chart/chart.module';
import {MessageMonitorComponent} from './components/message-monitor/message-monitor.component';
import {MessageValuePipe} from './pipes/message-value.pipe';
import {ToMillisecsPipe} from './pipes/time.pipe';
import {TableComponent} from './components/table/table.component';
import {PaginationComponent} from './components/pagination/pagination.component';

@NgModule({
  imports: [
    ThemeModule,
    NbButtonModule,
    NbCardModule,
    MapModule,
    ChartModule,
    NbSelectModule,
    NbDatepickerModule,
    NbInputModule,
    FormsModule,
    NbIconModule,
    NbCheckboxModule,
    MatChipsModule,
    MatIconModule,
  ],
  declarations: [
    ConfirmationComponent,
    MessageMonitorComponent,
    MessageValuePipe,
    ToMillisecsPipe,
    TableComponent,
    PaginationComponent,
  ],
  exports: [
    ThemeModule,
    NbCardModule,
    NbIconModule,
    MapModule,
    ChartModule,
    ConfirmationComponent,
    MessageMonitorComponent,
    TableComponent,
    PaginationComponent,
  ],
  providers: [
    MessageValuePipe,
    ToMillisecsPipe,
  ],
})

export class SharedModule {
}
