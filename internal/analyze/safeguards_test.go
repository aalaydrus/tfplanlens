package analyze

import (
	"reflect"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// The real plan (simulation, made by Terraform with the AWS provider) has
// "deletion_protection_enabled": false in the DynamoDB table's before values;
// that is what a reviewer needs to see next to the REPLACE.
func TestRealPlanDynamoDBTableReplaceShowsProtectionOff(t *testing.T) {
	a := Analyze(loadSimulation(t), nil, nil)
	for _, f := range a.Findings {
		if f.Address != "aws_dynamodb_table.orders" {
			continue
		}
		want := []Safeguard{{Name: SafeguardDeletionProtection, State: StateOff}}
		if !reflect.DeepEqual(f.Safeguards, want) {
			t.Errorf("safeguards %+v, want %+v", f.Safeguards, want)
		}
		return
	}
	t.Fatal("no finding for aws_dynamodb_table.orders")
}

func del(typ string, before map[string]any) plan.Change {
	return plan.Change{Address: typ + ".x", Type: typ, Action: plan.ActionDelete, Before: before}
}

func TestSafeguardsFromBeforeValues(t *testing.T) {
	cases := []struct {
		name string
		c    plan.Change
		want []Safeguard
	}{
		{"rds protected, snapshot taken", del("aws_db_instance", map[string]any{"deletion_protection": true, "skip_final_snapshot": false}),
			[]Safeguard{{SafeguardDeletionProtection, StateOn}, {SafeguardFinalSnapshot, StateTaken}}},
		{"rds unprotected, no snapshot", del("aws_db_instance", map[string]any{"deletion_protection": false, "skip_final_snapshot": true}),
			[]Safeguard{{SafeguardDeletionProtection, StateOff}, {SafeguardFinalSnapshot, StateNone}}},
		{"rds cluster, attributes missing from the plan", del("aws_rds_cluster", map[string]any{}),
			[]Safeguard{{SafeguardDeletionProtection, StateUnknown}, {SafeguardFinalSnapshot, StateUnknown}}},
		{"rds, null values", del("aws_db_instance", map[string]any{"deletion_protection": nil, "skip_final_snapshot": nil}),
			[]Safeguard{{SafeguardDeletionProtection, StateUnknown}, {SafeguardFinalSnapshot, StateUnknown}}},
		{"rds, wrong type is unknown, not a guess", del("aws_db_instance", map[string]any{"deletion_protection": "yes"}),
			[]Safeguard{{SafeguardDeletionProtection, StateUnknown}, {SafeguardFinalSnapshot, StateUnknown}}},
		{"dynamodb", del("aws_dynamodb_table", map[string]any{"deletion_protection_enabled": true}),
			[]Safeguard{{SafeguardDeletionProtection, StateOn}}},
		{"type without a known attribute", del("aws_sqs_queue", map[string]any{"deletion_protection": true}), []Safeguard{}},
		{"no before values at all", del("aws_db_instance", nil),
			[]Safeguard{{SafeguardDeletionProtection, StateUnknown}, {SafeguardFinalSnapshot, StateUnknown}}},
	}
	for _, tc := range cases {
		got := safeguardsFor(tc.c)
		if got == nil {
			t.Errorf("%s: nil (JSON arrays are [] not null)", tc.name)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// An unprotected stateful delete is the one that loses data, so it comes before
// a protected one (which AWS should refuse).
func TestUnprotectedDeletesRankBeforeProtectedOnes(t *testing.T) {
	fs := []Finding{
		{Address: "protected", Stateful: true, Safeguards: []Safeguard{{SafeguardDeletionProtection, StateOn}}},
		{Address: "unknown", Stateful: true, Safeguards: []Safeguard{{SafeguardDeletionProtection, StateUnknown}}},
		{Address: "off", Stateful: true, Safeguards: []Safeguard{{SafeguardDeletionProtection, StateOff}}},
	}
	rankFindings(fs)
	var got []string
	for _, f := range fs {
		got = append(got, f.Address)
	}
	if want := []string{"unknown", "off", "protected"}; !reflect.DeepEqual(got, want) {
		t.Errorf("%v, want %v", got, want)
	}
}
