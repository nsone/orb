import {
  Component,
  EventEmitter,
  Input,
  OnChanges,
  OnInit,
  Output,
  SimpleChanges,
} from '@angular/core';
import { AgentPolicy } from 'app/common/interfaces/orb/agent.policy.interface';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Tags } from 'app/common/interfaces/orb/tag';
import { OrbService } from 'app/common/services/orb.service';

@Component({
  selector: 'ngx-policy-details',
  templateUrl: './policy-details.component.html',
  styleUrls: ['./policy-details.component.scss'],
})
export class PolicyDetailsComponent implements OnInit, OnChanges {
  @Input()
  policy: AgentPolicy;

  @Input()
  editMode: boolean;

  @Output()
  editModeChange: EventEmitter<boolean>;

  @Input()
  interfaceEditMode: boolean;

  formGroup: FormGroup;

  selectedTags: Tags;

  constructor(
    private fb: FormBuilder,
    private orb: OrbService,
    ) {
    this.policy = {};
    this.editMode = false;
    this.editModeChange = new EventEmitter<boolean>();
    this.updateForm();
    this.interfaceEditMode = false;
  }

  ngOnInit(): void {
    this.selectedTags = this.policy?.tags || {};
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes?.editMode) {
      this.toggleEdit(changes.editMode.currentValue, false);
    }
    if (changes?.policy) {
      this.selectedTags = this.policy?.tags || {};
    }
  }

  updateForm() {
    if (this.editMode) {
      const { name, description, tags } = this.policy;
      this.formGroup = this.fb.group({
        name: [
          name,
          [
            Validators.required,
            Validators.pattern('^[a-zA-Z_][a-zA-Z0-9_-]*$'),
            Validators.maxLength(64),
            Validators.minLength(2),
          ],
        ],
        description: [description],
      });
      this.selectedTags = {...tags} || {};
    } else {
      this.formGroup = this.fb.group({
        name: null,
        description: null,
      });
    }
  }

  toggleEdit(value, notify = true) {
    this.editMode = value;
    if (this.editMode || this.interfaceEditMode) {
      this.orb.pausePolling();
    } else {
      this.orb.startPolling();
    }
    this.updateForm();
    !!notify && this.editModeChange.emit(this.editMode);
  }
}
