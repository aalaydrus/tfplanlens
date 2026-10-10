package markdown_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
	"github.com/aalaydrus/tfplanlens/internal/plan"
	"github.com/aalaydrus/tfplanlens/internal/render/markdown"
)

func render(t *testing.T, planJSON string, opts *analyze.Options) string {
	t.Helper()
	p, err := plan.Load(strings.NewReader(planJSON))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := markdown.Render(&buf, analyze.Analyze(p, nil, opts)); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../../../testdata/plans/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// <details> must always be balanced or the rest of a PR comment disappears.
func balanced(t *testing.T, md string) {
	t.Helper()
	open := strings.Count(md, "<details")
	closed := strings.Count(md, "</details>")
	if open != closed {
		t.Errorf("%d <details> but %d </details>", open, closed)
	}
}

func TestShowsApplyAccountAndTargetSources(t *testing.T) {
	md := render(t, fixture(t, "runner-cross-account"), &analyze.Options{ApplyAccount: "999999999999", ApplySource: analyze.SourceFlag})
	for _, want := range []string{
		markdown.Marker,
		"| Executed from (apply account) | `999999999999` (source: flag) |",
		"target account 222222222222 (source: role_arn_variable)",
		"target account 999999999999 (source: apply_account)",
		"differs from apply account",
		"`222222222222` (source: role_arn_variable) |", // Danger table
		"kms_key_arn is in account 111111111111",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q", want)
		}
	}
	balanced(t, md)
	if !strings.Contains(render(t, fixture(t, "runner-cross-account"), nil), "**unresolved**") {
		t.Error("unresolved apply account not shown")
	}
}

func TestDangerExpandedBeforeEverythingInDetails(t *testing.T) {
	md := render(t, fixture(t, "deletes-and-replaces"), nil)
	danger, firstDetails := strings.Index(md, "### Danger"), strings.Index(md, "<details")
	if danger < 0 || firstDetails < 0 || danger > firstDetails {
		t.Errorf("danger=%d details=%d", danger, firstDetails)
	}
	// Findings are table rows, not hidden inside <details>.
	if !strings.Contains(md[danger:firstDetails], "aws_db_instance.main") {
		t.Error("danger findings must be visible without expanding anything")
	}
}

func TestSensitiveNeverPrinted(t *testing.T) {
	md := render(t, fixture(t, "sensitive"), nil)
	for _, secret := range []string{"old-secret-pw", "new-secret-pw", "s3cr3t-value"} {
		if strings.Contains(md, secret) {
			t.Errorf("leaks %q", secret)
		}
	}
	if !strings.Contains(md, "(sensitive value changed)") {
		t.Error("no sensitive marker")
	}
}

func TestLongValueShowsChangedLinesInFence(t *testing.T) {
	md := render(t, fixture(t, "long-user-data"), nil)
	if !strings.Contains(md, "4 lines changed of 412") || !strings.Contains(md, "```diff") || strings.Contains(md, "step 5:") {
		t.Error("long value diff missing or full value printed")
	}
}

// Plan data is untrusted: it must not be able to break tables, close
// <details>, or open fences in a PR comment.
func TestHostileDataCannotBreakStructure(t *testing.T) {
	planJSON := `{"format_version":"1.2","resource_changes":[
	 {"address":"aws_x.y[\"a|b` + "`" + `c</details><b>pwn</b>\"]","mode":"managed","type":"aws_x","name":"y","index":"a|b` + "`" + `c</details><b>pwn</b>",
	  "provider_name":"registry.terraform.io/hashicorp/aws","action_reason":"delete_because_each_key",
	  "change":{"actions":["delete"],"before":{"a":"1"},"after":null}},
	 {"address":"aws_x.z","mode":"managed","type":"aws_x","name":"z","provider_name":"registry.terraform.io/hashicorp/aws",
	  "change":{"actions":["update"],"before":{"policy":"` + "```\\n</details>" + `"},"after":{"policy":"` + "```\\nx" + `"}}}]}`
	md := render(t, planJSON, nil)
	// The only <details> tags are ours: a hostile value always sits inside a
	// code span, which GitHub does not render as HTML.
	for _, line := range strings.Split(md, "\n") {
		if i := strings.Index(line, "</details><b>pwn"); i >= 0 {
			if !strings.Contains(line[:i], "``") {
				t.Errorf("hostile text outside a code span: %q", line)
			}
		}
	}
	if strings.Contains(md, "| a|b") {
		t.Error("unescaped pipe inside a table cell")
	}
	opens, closes := 0, 0
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "<details") {
			opens++
		}
		if strings.TrimSpace(line) == "</details>" {
			closes++
		}
	}
	if opens != closes {
		t.Errorf("structural <details> %d vs %d", opens, closes)
	}
}

func TestCapAndTruncationNote(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"format_version":"1.2","resource_changes":[`)
	for i := 0; i < 5000; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"address":"aws_instance.i[%d]","mode":"managed","type":"aws_instance","name":"i","index":%d,"provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["update"],"before":{"ami":"a","tags":{"n":"%d"}},"after":{"ami":"b","tags":{"n":"%d"}}}}`, i, i, i, i+1)
	}
	sb.WriteString(`]}`)
	md := render(t, sb.String(), nil)
	if len(md) > markdown.MaxChars {
		t.Errorf("%d bytes, cap %d", len(md), markdown.MaxChars)
	}
	if !strings.Contains(md, "Report truncated") {
		t.Error("no truncation note")
	}
	balanced(t, md)

	small := render(t, fixture(t, "basic-create"), nil)
	if strings.Contains(small, "Report truncated") {
		t.Error("small report truncated")
	}
}

func TestTruncationNeverDropsDangerFirst(t *testing.T) {
	// 700 deletes: the Danger table alone nears the cap, and must still be
	// well-formed and come first.
	var sb strings.Builder
	sb.WriteString(`{"format_version":"1.2","resource_changes":[`)
	for i := 0; i < 700; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"address":"aws_s3_bucket.b[%d]","mode":"managed","type":"aws_s3_bucket","name":"b","index":%d,"provider_name":"registry.terraform.io/hashicorp/aws","action_reason":"delete_because_count_index","change":{"actions":["delete"],"before":{"bucket":"%s"},"after":null}}`, i, i, strings.Repeat("x", 40))
	}
	sb.WriteString(`]}`)
	md := render(t, sb.String(), nil)
	if len(md) > markdown.MaxChars {
		t.Errorf("%d bytes", len(md))
	}
	if strings.Index(md, "### Danger") > strings.Index(md, "<details") && strings.Contains(md, "<details") {
		t.Error("danger not first")
	}
	balanced(t, md)
	if !strings.Contains(md, "aws_s3_bucket.b[0]") {
		t.Error("first finding missing")
	}
}
