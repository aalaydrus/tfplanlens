package analyze

import (
	"reflect"
	"strings"
	"testing"
)

func addresses(items []BriefItem) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Address)
	}
	return out
}

// Expected order comes from reading the real plan with jq: the three deletes
// and replaces, then the in-place updates that other resources depend on, most
// dependents first (aws_sqs_queue.orders has 3, aws_secretsmanager_secret.db 2;
// updates nothing depends on, and creates, are not listed).
func TestBriefRanksDangerThenUpdatesOthersDependOn(t *testing.T) {
	a := Analyze(loadSimulation(t), nil, nil)
	b := BriefOf(a, 10)
	want := []string{
		"aws_dynamodb_table.orders", `aws_sqs_queue.orders["bulk"]`, "aws_secretsmanager_secret_version.db",
		`aws_sqs_queue.orders["standard"]`, "aws_secretsmanager_secret.db",
	}
	if got := addresses(b.Items); !reflect.DeepEqual(got, want) {
		t.Errorf("items %v, want %v", got, want)
	}
	if b.Omitted != 0 || b.Total != 5 {
		t.Errorf("total %d omitted %d, want 5 and 0", b.Total, b.Omitted)
	}
	for i, it := range b.Items {
		if it.Rank != i+1 {
			t.Errorf("item %d has rank %d", i, it.Rank)
		}
		if len(it.Why) == 0 {
			t.Errorf("%s: says nothing about why it matters", it.Address)
		}
	}
	first := strings.Join(b.Items[0].Why, "\n")
	for _, want := range []string{"forces replacement", "used by 1+ resources", "deletion protection is off"} {
		if !strings.Contains(first, want) {
			t.Errorf("first item lacks %q: %s", want, first)
		}
	}
}

func TestBriefTruncatesAndCountsWhatIsLeft(t *testing.T) {
	a := Analyze(loadSimulation(t), nil, nil)
	b := BriefOf(a, 2)
	if len(b.Items) != 2 || b.Total != 5 || b.Omitted != 3 {
		t.Errorf("%d items, total %d, omitted %d; want 2, 5, 3", len(b.Items), b.Total, b.Omitted)
	}
	if got := BriefOf(a, 0); len(got.Items) != 5 {
		t.Errorf("top 0 must mean everything, got %d items", len(got.Items))
	}
}

func TestBriefOfAPlanWithNothingToFlagIsEmptyNotNil(t *testing.T) {
	b := BriefOf(Analyze(load(t, "basic-create"), nil, nil), 10)
	if b.Items == nil || len(b.Items) != 0 || b.Total != 0 {
		t.Errorf("%+v", b)
	}
}

func TestBriefFoldsInstancesAndPutsWarningsFirst(t *testing.T) {
	dep := []string{"a.x", "b.x"}
	a := &Analysis{Changes: []Change{
		{Address: `q.x["a"]`, Parent: "q.x", Key: "a", Action: "update", Dependents: dep},
		{Address: `q.x["b"]`, Parent: "q.x", Key: "b", Action: "update", Dependents: dep},
		{Address: "one.x", Action: "update", Dependents: []string{"c.x", "d.x", "e.x"}},
		{Address: "warned.x", Action: "create", Warnings: []Warning{{Kind: "arn_account_mismatch", Message: "topic_arn is in another account"}}},
		{Address: "quiet.x", Action: "update"},                // nothing depends on it
		{Address: "new.x", Action: "create", Dependents: dep}, // a create is not flagged
	}}
	b := BriefOf(a, 0)
	want := []string{"warned.x", "one.x", "q.x"}
	if got := addresses(b.Items); !reflect.DeepEqual(got, want) {
		t.Fatalf("items %v, want %v", got, want)
	}
	if !strings.Contains(b.Items[2].Why[0], "2 instances") {
		t.Errorf("folded entry: %v", b.Items[2].Why)
	}
	if !strings.Contains(strings.Join(b.Items[0].Why, " "), "another account") {
		t.Errorf("warning missing: %v", b.Items[0].Why)
	}
}
