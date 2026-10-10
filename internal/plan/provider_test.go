package plan_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

func TestRunnerFixtureProviders(t *testing.T) {
	f, err := os.Open(plansDir + "/runner-cross-account.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := plan.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	prod, staging, sandbox, def := p.Providers["aws.prod"], p.Providers["aws.staging"], p.Providers["aws.sandbox"], p.Providers["aws"]
	if prod.AssumeRoleARN != "arn:aws:iam::111111111111:role/tf-deployer" || prod.RoleARNVia != plan.ViaLiteral {
		t.Errorf("prod: %+v", prod)
	}
	if staging.AssumeRoleARN != "arn:aws:iam::222222222222:role/tf-deployer" || staging.RoleARNVia != plan.ViaVariable {
		t.Errorf("staging: %+v", staging)
	}
	if sandbox.AssumeRole || sandbox.AssumeRoleARN != "" {
		t.Errorf("sandbox should have no assume_role: %+v", sandbox)
	}
	if !reflect.DeepEqual(def.CallerIdentityIDs, []string{"999999999999"}) {
		t.Errorf("default provider caller identity: %+v", def.CallerIdentityIDs)
	}
}

func planWithProvider(expr string, vars string) string {
	return fmt.Sprintf(`{"format_version":"1.2","variables":%s,"resource_changes":[],
	"configuration":{"provider_config":{"aws":{"name":"aws","full_name":"registry.terraform.io/hashicorp/aws","expressions":{%s}}},"root_module":{}}}`, vars, expr)
}

func TestAllowedAccountIDsAndVariables(t *testing.T) {
	cases := []struct {
		name, expr, vars string
		wantIDs          []string
		wantARN, wantVia string
	}{
		{"literal list", `"allowed_account_ids":{"constant_value":["123456789012"]}`, `{}`, []string{"123456789012"}, "", ""},
		{"two ids", `"allowed_account_ids":{"constant_value":["123456789012","210987654321"]}`, `{}`, []string{"123456789012", "210987654321"}, "", ""},
		{"list from variable", `"allowed_account_ids":{"references":["var.ids"]}`, `{"ids":{"value":["123456789012"]}}`, []string{"123456789012"}, "", ""},
		{"variable not in plan", `"allowed_account_ids":{"references":["var.ids"]}`, `{}`, nil, "", ""},
		{"non-variable reference", `"allowed_account_ids":{"references":["local.ids"]}`, `{"ids":{"value":["123456789012"]}}`, nil, "", ""},
		{"role from variable attribute path", `"assume_role":[{"role_arn":{"references":["var.role","var.role"]}}]`, `{"role":{"value":"arn:aws:iam::123456789012:role/x"}}`, nil, "arn:aws:iam::123456789012:role/x", plan.ViaVariable},
		{"role from two variables", `"assume_role":[{"role_arn":{"references":["var.a","var.b"]}}]`, `{"a":{"value":"x"},"b":{"value":"y"}}`, nil, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := plan.Load(strings.NewReader(planWithProvider(c.expr, c.vars)))
			if err != nil {
				t.Fatal(err)
			}
			pr := p.Providers["aws"]
			if !reflect.DeepEqual(pr.AllowedAccountIDs, c.wantIDs) {
				t.Errorf("ids %v, want %v", pr.AllowedAccountIDs, c.wantIDs)
			}
			if pr.AssumeRoleARN != c.wantARN || pr.RoleARNVia != c.wantVia {
				t.Errorf("role %q via %q, want %q via %q", pr.AssumeRoleARN, pr.RoleARNVia, c.wantARN, c.wantVia)
			}
			if strings.Contains(c.expr, "assume_role") && !pr.AssumeRole {
				t.Error("AssumeRole should be true even when the ARN is unresolved")
			}
		})
	}
}
