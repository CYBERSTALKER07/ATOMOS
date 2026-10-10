package events

import "testing"

func TestB4EventConstants(t *testing.T) {
	if EventSupplierCreditProgramChanged != "SUPPLIER_CREDIT_PROGRAM_CHANGED" {
		t.Fatalf("program event = %q", EventSupplierCreditProgramChanged)
	}
	if EventSupplierCreditTermsChanged != "SUPPLIER_CREDIT_TERMS_CHANGED" {
		t.Fatalf("terms event = %q", EventSupplierCreditTermsChanged)
	}
	if EventControlTowerPlaybookChanged != "CONTROL_TOWER_PLAYBOOK_CHANGED" {
		t.Fatalf("playbook event = %q", EventControlTowerPlaybookChanged)
	}
	if EventControlTowerRunCreated != "CONTROL_TOWER_RUN_CREATED" {
		t.Fatalf("run created = %q", EventControlTowerRunCreated)
	}
	if EventControlTowerRunUpdated != "CONTROL_TOWER_RUN_UPDATED" {
		t.Fatalf("run updated = %q", EventControlTowerRunUpdated)
	}
	if AggregateControlTower != "ControlTower" {
		t.Fatalf("aggregate = %q", AggregateControlTower)
	}
	if EventPlanningScenarioPublished != "planning.scenario.published.v1" {
		t.Fatalf("scenario publish = %q", EventPlanningScenarioPublished)
	}
}
