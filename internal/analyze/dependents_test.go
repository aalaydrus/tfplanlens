package analyze

import (
	"os"
	"reflect"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

func loadSimulation(t *testing.T) *plan.Plan {
	t.Helper()
	f, err := os.Open("../../simulation/assets/input/plan.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := plan.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Expected values come from reading the real plan's configuration with jq
// (references per resource), not from the code under test:
//
//	aws_sqs_queue.orders      <- aws_iam_role_policy.worker, two sns subscriptions
//	aws_dynamodb_table.orders <- aws_iam_role_policy.worker
//	aws_secretsmanager_secret_version.db <- nothing
func TestFindingsCarryWhatDependsOnThem(t *testing.T) {
	a := Analyze(loadSimulation(t), nil, nil)
	got := map[string][]string{}
	for _, f := range a.Findings {
		got[f.Address] = f.Dependents
	}
	want := map[string][]string{
		`aws_sqs_queue.orders["bulk"]`:         {"aws_iam_role_policy.worker", "aws_sns_topic_subscription.audit", "aws_sns_topic_subscription.standard"},
		"aws_dynamodb_table.orders":            {"aws_iam_role_policy.worker"},
		"aws_secretsmanager_secret_version.db": {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependents %v, want %v", got, want)
	}
}

// A stateful replace stays first; among the rest, what more resources depend
// on comes first, and ties keep the plan's order.
func TestDangerIsRankedByStatefulThenDependents(t *testing.T) {
	a := Analyze(loadSimulation(t), nil, nil)
	var got []string
	for _, f := range a.Findings {
		got = append(got, f.Address)
	}
	want := []string{"aws_dynamodb_table.orders", `aws_sqs_queue.orders["bulk"]`, "aws_secretsmanager_secret_version.db"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("order %v, want %v", got, want)
	}
}
