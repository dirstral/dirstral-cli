package breeze

import "testing"

func TestPlannerModelProfileAffectsStrategyAndSynthesis(t *testing.T) {
	largePlan, err := PlanTurn("explain indexing", "mistral-large-latest")
	if err != nil {
		t.Fatalf("large plan error: %v", err)
	}
	if len(largePlan.Steps) != 2 {
		t.Fatalf("expected large model to use two-step plan, got %d", len(largePlan.Steps))
	}
	if largePlan.Steps[0].Tool != "dir2mcp.search" || largePlan.Steps[1].Tool != "dir2mcp.ask" {
		t.Fatalf("unexpected large model tool sequence: %#v", largePlan.Steps)
	}
	if largePlan.Synthesis != "analytical" {
		t.Fatalf("expected analytical synthesis for large model, got %q", largePlan.Synthesis)
	}

	smallPlan, err := PlanTurn("explain indexing", "mistral-small-latest")
	if err != nil {
		t.Fatalf("small plan error: %v", err)
	}
	if len(smallPlan.Steps) != 1 {
		t.Fatalf("expected small model to use one-step plan, got %d", len(smallPlan.Steps))
	}
	if smallPlan.Steps[0].Tool != "dir2mcp.ask" {
		t.Fatalf("expected small model to ask directly, got %q", smallPlan.Steps[0].Tool)
	}
	if smallPlan.Synthesis != "concise" {
		t.Fatalf("expected concise synthesis for small model, got %q", smallPlan.Synthesis)
	}
}

func TestPlannerSupportsClearCommand(t *testing.T) {
	plan, err := PlanTurn("/clear", "mistral-small-latest")
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	if !plan.Clear {
		t.Fatal("expected /clear to set Clear=true")
	}
	if len(plan.Steps) != 0 {
		t.Fatalf("expected no tool steps for /clear, got %d", len(plan.Steps))
	}
}
