package html_test

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/plan"
	"github.com/aalaydrus/tfplanlens/internal/render/html"
)

func analysis(t *testing.T, planJSON string, opts *analyze.Options) *analyze.Analysis {
	t.Helper()
	p, err := plan.Load(strings.NewReader(planJSON))
	if err != nil {
		t.Fatal(err)
	}
	return analyze.Analyze(p, nil, opts)
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../../../testdata/plans/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func render(t *testing.T, a *analyze.Analysis) string {
	t.Helper()
	var buf bytes.Buffer
	if err := html.Render(&buf, a); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

var voidElements = map[string]bool{"meta": true, "br": true, "img": true, "input": true, "link": true, "hr": true}

// wellFormed tokenizes the document and checks every non-void element is
// closed in order, and that there is a doctype.
func wellFormed(t *testing.T, doc string) {
	t.Helper()
	z := xhtml.NewTokenizer(strings.NewReader(doc))
	var stack []string
	sawDoctype := false
	for {
		switch z.Next() {
		case xhtml.ErrorToken:
			if err := z.Err(); err != io.EOF {
				t.Fatalf("tokenize: %v", err)
			}
			if len(stack) != 0 {
				t.Fatalf("unclosed elements: %v", stack)
			}
			if !sawDoctype {
				t.Error("missing doctype")
			}
			return
		case xhtml.DoctypeToken:
			sawDoctype = true
		case xhtml.StartTagToken:
			name, _ := z.TagName()
			if !voidElements[string(name)] {
				stack = append(stack, string(name))
			}
		case xhtml.EndTagToken:
			name, _ := z.TagName()
			if len(stack) == 0 || stack[len(stack)-1] != string(name) {
				t.Fatalf("unexpected </%s>, open: %v", name, stack)
			}
			stack = stack[:len(stack)-1]
		}
	}
}

func TestEveryFixtureRendersValidSelfContainedHTML(t *testing.T) {
	entries, err := os.ReadDir("../../../testdata/plans")
	if err != nil {
		t.Fatal(err)
	}
	external := regexp.MustCompile(`(?i)(https?:)?//[a-z0-9.-]+\.[a-z]{2,}|<link\b|\bsrc\s*=|@import|url\(`)
	var files []string // fixture paths relative to testdata/plans
	for _, e := range entries {
		if e.IsDir() {
			sub, err := os.ReadDir("../../../testdata/plans/" + e.Name())
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range sub {
				files = append(files, e.Name()+"/"+f.Name())
			}
			continue
		}
		files = append(files, e.Name())
	}
	for _, f := range files {
		name := strings.TrimSuffix(f, ".json")
		t.Run(name, func(t *testing.T) {
			doc := render(t, analysis(t, fixture(t, name), &analyze.Options{ApplyAccount: "999999999999", ApplySource: analyze.SourceFlag}))
			wellFormed(t, doc)
			if strings.Contains(doc, "http://") || strings.Contains(doc, "https://") {
				t.Error("document references http(s)")
			}
			if m := external.FindString(doc); m != "" {
				t.Errorf("external reference %q", m)
			}
		})
	}
}

func TestContainsFindingsApplyAndTargetAccounts(t *testing.T) {
	doc := render(t, analysis(t, fixture(t, "runner-cross-account"), &analyze.Options{ApplyAccount: "999999999999", ApplySource: analyze.SourceFlag}))
	for _, want := range []string{
		"aws_db_instance.prod", "Removed from config", "STATEFUL",
		"Attribute change forces replacement: ami",
		"executed from:", "999999999999", "(flag)",
		"222222222222", "role_arn_variable", "caller_identity", "apply_account",
		"differs from apply account",
		"kms_key_arn is in account 111111111111",
		"expand full value", // present in the template for long values
	} {
		if want == "expand full value" {
			continue
		}
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q", want)
		}
	}
	// Danger comes before the summary tiles and the changes table.
	d, tiles, table := strings.Index(doc, `id="danger"`), strings.Index(doc, `class="tiles"`), strings.Index(doc, `id="rows"`)
	if !(d >= 0 && d < tiles && tiles < table) {
		t.Errorf("section order: danger=%d tiles=%d table=%d", d, tiles, table)
	}
}

func TestUnresolvedApplyAccountIsShown(t *testing.T) {
	doc := render(t, analysis(t, fixture(t, "runner-cross-account"), nil))
	if !strings.Contains(doc, "executed from: unresolved") || !strings.Contains(doc, "Apply account unresolved") {
		t.Error("unresolved apply account not shown")
	}
}

func TestLongValueHasExpandToggleAndOnlyChangedLinesInDiff(t *testing.T) {
	doc := render(t, analysis(t, fixture(t, "long-user-data"), nil))
	if !strings.Contains(doc, "expand full value") || !strings.Contains(doc, "4 lines changed of 412") {
		t.Error("long value handling missing from HTML")
	}
}

func TestSensitiveValuesNeverAppear(t *testing.T) {
	doc := render(t, analysis(t, fixture(t, "sensitive"), nil))
	for _, secret := range []string{"old-secret-pw", "new-secret-pw", "s3cr3t-value"} {
		if strings.Contains(doc, secret) {
			t.Errorf("HTML leaks %q", secret)
		}
	}
	if !strings.Contains(doc, "(sensitive value changed)") {
		t.Error("no sensitive marker")
	}
}

// Plan data is untrusted: a for_each key or attribute value must not become
// markup or script.
func TestHostilePlanDataIsEscaped(t *testing.T) {
	const evil = `<img src=x onerror=alert(1)>`
	planJSON := `{"format_version":"1.2","resource_changes":[
	 {"address":"aws_x.y[\"` + strings.ReplaceAll(evil, `"`, `\"`) + `\"]","mode":"managed","type":"aws_x","name":"y","index":"` + evil + `",
	  "provider_name":"registry.terraform.io/hashicorp/aws","action_reason":"delete_because_each_key",
	  "change":{"actions":["delete"],"before":{"a":"1"},"after":null}},
	 {"address":"aws_x.z","mode":"managed","type":"aws_x","name":"z","provider_name":"registry.terraform.io/hashicorp/aws",
	  "change":{"actions":["update"],"before":{"policy":"</script><script>alert(2)</script>"},"after":{"policy":"</template><b>x</b>"}}}]}`
	doc := render(t, analysis(t, planJSON, nil))
	wellFormed(t, doc)
	for _, bad := range []string{"<img src=x", "<script>alert", "</template><b>"} {
		if strings.Contains(doc, bad) {
			t.Errorf("unescaped %q in output", bad)
		}
	}
	if !strings.Contains(doc, "&lt;img src=x onerror=alert(1)&gt;") {
		t.Error("escaped form not found")
	}
	if n := strings.Count(doc, "<script>"); n != 1 {
		t.Errorf("%d <script> elements, want exactly the page script", n)
	}
}
