package plan_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
)

const plansDir = "../../testdata/plans"

var fixtureNames = []string{
	"basic-create", "deletes-and-replaces", "for-each-shift", "long-user-data",
	"iam-policy-change", "drift", "multi-account", "moved-import-removed", "sensitive", "runner-cross-account", "secret-lookalikes",
}

func load(t *testing.T, name string) (*tfjson.Plan, []byte) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(plansDir, name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var p tfjson.Plan
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return &p, raw
}

// Every fixture must be a loadable plan and nothing else should sit in the
// directory unnoticed.
func TestFixturesParse(t *testing.T) {
	for _, n := range fixtureNames {
		t.Run(n, func(t *testing.T) {
			p, _ := load(t, n)
			if len(p.ResourceChanges) == 0 {
				t.Fatal("no resource_changes")
			}
		})
	}
	files, _ := filepath.Glob(filepath.Join(plansDir, "*.json"))
	if len(files) != len(fixtureNames) {
		t.Errorf("found %d fixture files, expected %d", len(files), len(fixtureNames))
	}
}

func TestDeletesAndReplacesCoverEveryActionReason(t *testing.T) {
	p, _ := load(t, "deletes-and-replaces")
	seen := map[tfjson.ActionReason]bool{}
	for _, rc := range p.ResourceChanges {
		seen[rc.ActionReason] = true
	}
	for _, want := range []tfjson.ActionReason{
		tfjson.ActionReasonDeleteBecauseNoResourceConfig,
		tfjson.ActionReasonDeleteBecauseWrongRepetition,
		tfjson.ActionReasonDeleteBecauseCountIndex,
		tfjson.ActionReasonDeleteBecauseEachKey,
		tfjson.ActionReasonDeleteBecauseNoModule,
		tfjson.ActionReasonReplaceBecauseCannotUpdate,
		tfjson.ActionReasonReplaceBecauseTainted,
		tfjson.ActionReasonReplaceByRequest,
		// Not in the SPEC 5.2 table but emitted by Terraform.
		tfjson.ActionReasonReplaceByTriggers,
		tfjson.ActionReasonDeleteBecauseNoMoveTarget,
		"some_future_reason", // exercises the "show the raw value" fallback
	} {
		if !seen[want] {
			t.Errorf("missing action_reason %q", want)
		}
	}
}

func TestReplacePathsPresentInRawJSON(t *testing.T) {
	// terraform-json has no ReplacePaths field; the loader must read it raw.
	_, raw := load(t, "deletes-and-replaces")
	if !bytes.Contains(raw, []byte(`"replace_paths"`)) {
		t.Error("expected replace_paths in raw fixture")
	}
}

func TestForEachShift(t *testing.T) {
	p, _ := load(t, "for-each-shift")
	var eachKey, countIdx int
	for _, rc := range p.ResourceChanges {
		switch rc.ActionReason {
		case tfjson.ActionReasonDeleteBecauseEachKey:
			eachKey++
		case tfjson.ActionReasonDeleteBecauseCountIndex:
			countIdx++
		}
	}
	if eachKey == 0 || countIdx == 0 {
		t.Errorf("want for_each key and count index deletes, got %d and %d", eachKey, countIdx)
	}
}

func TestLongUserData(t *testing.T) {
	p, _ := load(t, "long-user-data")
	for _, rc := range p.ResourceChanges {
		if rc.Address != "aws_instance.app" {
			continue
		}
		b := rc.Change.Before.(map[string]any)["user_data_base64"].(string)
		dec, err := base64.StdEncoding.DecodeString(b)
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(dec), "\n"); n < 400 {
			t.Errorf("user_data has %d lines, want 400+", n)
		}
		return
	}
	t.Fatal("aws_instance.app not found")
}

func TestIAMPolicyFixturesAreJSONStrings(t *testing.T) {
	p, _ := load(t, "iam-policy-change")
	for _, rc := range p.ResourceChanges {
		for _, side := range []any{rc.Change.Before, rc.Change.After} {
			for _, k := range []string{"policy", "assume_role_policy"} {
				s, ok := side.(map[string]any)[k].(string)
				if !ok {
					continue
				}
				if !json.Valid([]byte(s)) {
					t.Errorf("%s.%s is not valid JSON", rc.Address, k)
				}
			}
		}
	}
}

func TestDriftPopulated(t *testing.T) {
	p, _ := load(t, "drift")
	if len(p.ResourceDrift) == 0 {
		t.Fatal("resource_drift empty")
	}
	// One drifted resource must also be replaced (SPEC 5.3 high-risk case).
	drifted := map[string]bool{}
	for _, d := range p.ResourceDrift {
		drifted[d.Address] = true
	}
	for _, rc := range p.ResourceChanges {
		if drifted[rc.Address] && rc.Change.Actions.Replace() {
			return
		}
	}
	t.Error("no resource is both drifted and replaced")
}

func TestMultiAccountProviders(t *testing.T) {
	p, _ := load(t, "multi-account")
	for _, k := range []string{"aws", "aws.prod", "aws.shared", "aws.mystery"} {
		if p.Config.ProviderConfigs[k] == nil {
			t.Errorf("provider_config %q missing", k)
		}
	}
}

func TestMovedImportRemoved(t *testing.T) {
	p, _ := load(t, "moved-import-removed")
	var moved, imported, forgot bool
	for _, rc := range p.ResourceChanges {
		moved = moved || rc.PreviousAddress != ""
		imported = imported || rc.Change.Importing != nil
		forgot = forgot || rc.Change.Actions.Forget()
	}
	if !moved || !imported || !forgot {
		t.Errorf("moved=%v imported=%v forgot=%v", moved, imported, forgot)
	}
}

func TestSensitiveMarked(t *testing.T) {
	p, _ := load(t, "sensitive")
	for _, rc := range p.ResourceChanges {
		if rc.Change.AfterSensitive == nil {
			t.Errorf("%s: after_sensitive missing", rc.Address)
		}
	}
}
