package plan_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/plan"
)

// A configuration touching every kind of dependency edge.
const depsConfig = `{"format_version":"1.2","resource_changes":[],"configuration":{
 "provider_config":{"aws":{"name":"aws","full_name":"registry.terraform.io/hashicorp/aws","expressions":{"region":{"references":["data.aws_region.current.name","data.aws_region.current"]}}}},
 "root_module":{
  "resources":[
   {"address":"data.aws_region.current","mode":"data","type":"aws_region","name":"current","provider_config_key":"aws"},
   {"address":"aws_vpc.main","mode":"managed","type":"aws_vpc","name":"main","provider_config_key":"aws"},
   {"address":"aws_subnet.a","mode":"managed","type":"aws_subnet","name":"a","provider_config_key":"aws",
    "expressions":{"vpc_id":{"references":["aws_vpc.main.id","aws_vpc.main"]},"tags":[{"zone":{"references":["local.zone"]}}]}},
   {"address":"aws_instance.web","mode":"managed","type":"aws_instance","name":"web","provider_config_key":"aws",
    "count_expression":{"references":["var.n"]},
    "expressions":{"subnet_id":{"references":["aws_subnet.a[0].id","aws_subnet.a"]},"net":{"references":["module.net.subnet_id","module.net"]}}},
   {"address":"aws_eip.ip","mode":"managed","type":"aws_eip","name":"ip","provider_config_key":"aws","depends_on":["aws_instance.web"],
    "expressions":{"each":{"references":["each.key","count.index","path.module","self.id"]}}},
   {"address":"aws_s3_bucket.logs","mode":"managed","type":"aws_s3_bucket","name":"logs","provider_config_key":"aws","depends_on":["module.late"]}
  ],
  "module_calls":{
   "net":{"source":"./net","expressions":{"vpc":{"references":["aws_vpc.main.id","aws_vpc.main"]},"unused":{"constant_value":"x"}},
    "module":{"outputs":{"subnet_id":{"expression":{"references":["aws_subnet.inner.id","aws_subnet.inner"]}}},
     "resources":[{"address":"aws_subnet.inner","mode":"managed","type":"aws_subnet","name":"inner","provider_config_key":"aws","expressions":{"vpc_id":{"references":["var.vpc"]}}}]}},
   "late":{"source":"./late","depends_on":["aws_vpc.main"],
    "module":{"resources":[{"address":"aws_sqs_queue.q","mode":"managed","type":"aws_sqs_queue","name":"q","provider_config_key":"aws"}]}}
  }}}}`

func TestDependencyGraph(t *testing.T) {
	p, err := plan.Load(strings.NewReader(depsConfig))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"data.aws_region.current": {},
		// provider config references data.aws_region, so every resource using aws depends on it
		"aws_vpc.main":                {"data.aws_region.current"},
		"aws_subnet.a":                {"aws_vpc.main", "data.aws_region.current"},
		"aws_instance.web":            {"aws_subnet.a", "data.aws_region.current", "module.net.aws_subnet.inner"}, // var.n at the root is no node; module.net.subnet_id resolves through the output
		"aws_eip.ip":                  {"aws_instance.web", "data.aws_region.current"},                            // each/count/path/self are not dependencies
		"aws_s3_bucket.logs":          {"data.aws_region.current", "module.late.aws_sqs_queue.q"},                 // depends_on a whole module
		"module.net.aws_subnet.inner": {"aws_vpc.main", "data.aws_region.current"},                                // var.vpc resolves to the module argument
		"module.late.aws_sqs_queue.q": {"aws_vpc.main", "data.aws_region.current"},                                // module depends_on applies to everything inside
	}
	for node, deps := range want {
		got, ok := p.Graph.Deps[node]
		if !ok {
			t.Errorf("%s missing from graph", node)
			continue
		}
		if len(deps) == 0 && len(got) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, deps) {
			t.Errorf("%s depends on %v, want %v", node, got, deps)
		}
	}
	if len(p.Graph.Deps) != len(want) {
		t.Errorf("%d nodes, want %d: %v", len(p.Graph.Deps), len(want), p.Graph.Deps)
	}
	if p.Graph.LocalRefs != 1 {
		t.Errorf("LocalRefs = %d, want 1 (local.zone)", p.Graph.LocalRefs)
	}
}

func TestDependencyGraphWithoutConfiguration(t *testing.T) {
	p, err := plan.Load(strings.NewReader(`{"format_version":"1.2","resource_changes":[]}`))
	if err != nil || p.Graph == nil || len(p.Graph.Deps) != 0 {
		t.Fatalf("%v %+v", err, p.Graph)
	}
}

func TestChangeConfigAddress(t *testing.T) {
	p := loadRealForDeps(t, "complex")
	seen := map[string]string{}
	for _, c := range p.Changes {
		seen[c.Address] = c.ConfigAddress
	}
	for addr, want := range map[string]string{
		`terraform_data.subnet["b"]`:                     "terraform_data.subnet",
		"terraform_data.worker[2]":                       "terraform_data.worker",
		"module.network_core.terraform_data.route_table": "module.network_core.terraform_data.route_table",
		"terraform_data.db":                              "terraform_data.db",
	} {
		if seen[addr] != want {
			t.Errorf("%s: ConfigAddress %q, want %q", addr, seen[addr], want)
		}
	}
}

// Real Terraform 1.16.5 output.
func TestRealComplexDependencyGraph(t *testing.T) {
	g := loadRealForDeps(t, "complex").Graph
	for node, want := range map[string][]string{
		"terraform_data.app":                             {"terraform_data.db"},
		"terraform_data.lb":                              {"terraform_data.subnet"},
		"terraform_data.db":                              nil,
		"terraform_data.creds":                           nil, // var.secret is a root variable
		"module.network_core.terraform_data.route_table": {"module.network_core.terraform_data.vpc"},
		"module.network_core.terraform_data.vpc":         nil, // var.name is a constant
	} {
		got, ok := g.Deps[node]
		if !ok || (len(want) == 0 && len(got) != 0) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Errorf("%s: %v (present %v), want %v", node, got, ok, want)
		}
	}
	// terraform_data.user_data reads local.user_data_lines: invisible.
	if g.LocalRefs != 1 {
		t.Errorf("LocalRefs = %d, want 1", g.LocalRefs)
	}
}

// replace_triggered_by is not in the configuration JSON at all, so it cannot
// become an edge. Check that the real plan really lacks it.
func TestRealReplaceTriggeredByIsNotInTheGraph(t *testing.T) {
	g := loadRealForDeps(t, "reasons").Graph
	if deps := g.Deps["terraform_data.follower"]; len(deps) != 0 {
		t.Errorf("follower deps = %v: replace_triggered_by is not exposed by Terraform, expected none", deps)
	}
}

func TestRealAWSModuleDependencies(t *testing.T) {
	g := loadRealForDeps(t, "aws-assume-role").Graph
	got := g.Deps["aws_sns_topic_subscription.cross"]
	want := []string{"aws_sns_topic.prod", "aws_sqs_queue.runner"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("subscription deps = %v, want %v", got, want)
	}
	if deps := g.Deps["module.net.aws_sqs_queue.inner"]; len(deps) != 0 {
		t.Errorf("module queue deps = %v (var.suffix is a root variable)", deps)
	}
}
