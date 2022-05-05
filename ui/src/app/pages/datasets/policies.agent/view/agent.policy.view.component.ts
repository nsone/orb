import { ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AgentPolicy } from 'app/common/interfaces/orb/agent.policy.interface';
import { PolicyConfig } from 'app/common/interfaces/orb/policy/config/policy.config.interface';
import { AgentPoliciesService } from 'app/common/services/agents/agent.policies.service';
import { PolicyDetailsComponent } from 'app/shared/components/orb/policy/policy-details/policy-details.component';
import { PolicyInterfaceComponent } from 'app/shared/components/orb/policy/policy-interface/policy-interface.component';
import { STRINGS } from 'assets/text/strings';
import { Subscription } from 'rxjs';

@Component({
  selector: 'ngx-agent-view',
  templateUrl: './agent.policy.view.component.html',
  styleUrls: ['./agent.policy.view.component.scss'],
})
export class AgentPolicyViewComponent implements OnInit, OnDestroy {
  strings = STRINGS.agents;

  isLoading: boolean;

  policyId: string;

  policy: AgentPolicy;

  policySubscription: Subscription;

  editMode = {
    details: false, interface: false,
  };

  @ViewChild(PolicyDetailsComponent) detailsComponent: PolicyDetailsComponent;

  @ViewChild(
    PolicyInterfaceComponent) interfaceComponent: PolicyInterfaceComponent;

  constructor(
    private route: ActivatedRoute,
    private policiesService: AgentPoliciesService,
    private cdr: ChangeDetectorRef,
  ) {}

  ngOnInit() {
    this.policyId = this.route.snapshot.paramMap.get('id');
    this.retrievePolicy();
  }

  isEditMode() {
    return Object.values(this.editMode)
      .reduce((prev, cur) => prev || cur, false);
  }

  canSave() {
    const detailsValid = this.editMode.details
      ? this.detailsComponent?.formGroup?.status === 'VALID' : true;

    const interfaceValid = this.editMode.interface
      ? this.interfaceComponent?.formControl?.status === 'VALID' : true;

    return detailsValid && interfaceValid;
  }

  discard() {
    this.editMode.details = false;
    this.editMode.interface = false;
  }

  save() {
    const {
      format, version, name, description, policy, policy_data, id, tags, backend,
    } = this.policy;

    // get values from all modified sections' forms and submit through service.
    const policyDetails = this.detailsComponent.formGroup?.value;
    const policyInterface = this.interfaceComponent.formControl?.value;

    console.table(this.editMode);

    const detailsPartial = !!this.editMode.details && {
      ...policyDetails,
    } || { name, description };

    const interFacePartial = !!this.editMode.interface && (
      format === 'yaml' ? {
        format: 'yaml', // this should be refactored out.
        policy_data: policyInterface,
      } : {
        policy: JSON.parse(policyInterface) as PolicyConfig,
      }
    ) || format === 'yaml' ? { policy_data, format, backend } : { policy, backend };

    const payload = {
      ...detailsPartial,
      ...interFacePartial,
      version, id, tags,
    } as AgentPolicy;

    this.policiesService.editAgentPolicy(payload)
      .subscribe(resp => {
        this.discard();
        this.retrievePolicy();
        this.cdr.markForCheck();
      });
  }

  retrievePolicy() {
    this.isLoading = true;

    this.policySubscription = this.policiesService
      .getAgentPolicyById(this.policyId)
      .subscribe(policy => {
        this.policy = policy;
        this.isLoading = false;
        this.cdr.markForCheck();
      });
  }

  ngOnDestroy() {
    this.policySubscription?.unsubscribe();
  }
}
