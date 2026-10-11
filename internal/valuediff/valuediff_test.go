package valuediff

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func gz(s string) string {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write([]byte(s))
	w.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestDecode(t *testing.T) {
	cases := []struct {
		name, in, want string
		steps          []string
	}{
		{"plain", "hello world", "hello world", nil},
		{"base64", b64("#!/bin/bash\necho hi\n"), "#!/bin/bash\necho hi\n", []string{"base64"}},
		{"gzip", gz("line1\nline2\n"), "line1\nline2\n", []string{"base64", "gzip"}},
		{"json sorted", `{"b":1,"a":{"d":2,"c":3}}`, "{\n  \"a\": {\n    \"c\": 3,\n    \"d\": 2\n  },\n  \"b\": 1\n}", []string{"json"}},
		{"base64 json", b64(`{"z":1,"a":2}`), "{\n  \"a\": 2,\n  \"z\": 1\n}", []string{"base64", "json"}},
		{"gzip json", gz(`[{"b":1,"a":2}]`), "[\n  {\n    \"a\": 2,\n    \"b\": 1\n  }\n]", []string{"base64", "gzip", "json"}},
		// Valid base64 that decodes to binary garbage must be left alone.
		{"base64-looking word", "abcdefgh", "abcdefgh", nil},
		{"bad json stays", `{"a":`, `{"a":`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, steps := Decode(c.in)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
			if fmt.Sprint(steps) != fmt.Sprint(c.steps) {
				t.Errorf("steps %v, want %v", steps, c.steps)
			}
		})
	}
}

func TestIsLong(t *testing.T) {
	if !IsLong("anything", strings.Repeat("x", 201), nil) {
		t.Error("201 chars should be long")
	}
	if IsLong("anything", strings.Repeat("x", 200), nil) {
		t.Error("200 chars should not be long")
	}
	if !IsLong("user_data", "short", nil) || !IsLong("policy", "x", nil) {
		t.Error("builtin keys are long")
	}
	if IsLong("custom_script", "x", nil) || !IsLong("custom_script", "x", []string{"custom_script"}) {
		t.Error("config keys extend the list")
	}
}

func lines(n int, edit map[int]string) string {
	var out []string
	for i := 0; i < n; i++ {
		l := fmt.Sprintf("line %d", i)
		if e, ok := edit[i]; ok {
			l = e
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n") + "\n"
}

func TestDiffChangedLinesPlusContext(t *testing.T) {
	r := Diff("user_data", lines(412, nil), lines(412, map[int]string{200: "changed"}))
	if r.Header != "user_data: 2 lines changed of 412 (+1 -1)" {
		t.Errorf("header %q", r.Header)
	}
	want := []string{"@@ -199,5 +199,5 @@", " line 198", " line 199", "-line 200", "+changed", " line 201", " line 202"}
	if fmt.Sprint(r.Lines) != fmt.Sprint(want) {
		t.Errorf("lines %q", r.Lines)
	}
}

func TestDiffNeverPrintsFullValue(t *testing.T) {
	r := Diff("user_data", "", lines(400, nil))
	if len(r.Lines) > MaxDiffLines || r.Truncated == 0 {
		t.Errorf("%d lines, truncated %d", len(r.Lines), r.Truncated)
	}
}

func TestDiffSeesThroughEncoding(t *testing.T) {
	r := Diff("user_data_base64", gz(lines(50, nil)), b64(lines(50, map[int]string{3: "x"})))
	if r.Added != 1 || r.Removed != 1 {
		t.Errorf("+%d -%d", r.Added, r.Removed)
	}
}

func TestPolicyStatements(t *testing.T) {
	doc := `{"Version":"2012-10-17","Statement":[
	  {"Effect":"Allow","Action":["s3:PutObject","s3:GetObject"],"Resource":["arn:b/*","arn:b"]},
	  {"Effect":"Deny","NotAction":"iam:*","Resource":"*"},
	  {"Effect":"Allow","Action":"sts:AssumeRole","Principal":{"Service":["b.amazonaws.com","a.amazonaws.com"]}}]}`
	got := PolicyStatements(doc)
	want := []string{
		"Allow s3:GetObject,s3:PutObject -> arn:b,arn:b/*",
		"Deny NotAction:iam:* -> *",
		"Allow sts:AssumeRole -> principal Service=a.amazonaws.com|b.amazonaws.com",
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %q", got)
	}
	if PolicyStatements(`{"not":"a policy"}`) != nil || PolicyStatements("nope") != nil {
		t.Error("non-policies must return nil")
	}
	one := PolicyStatements(`{"Statement":{"Effect":"Allow","Action":"*","Resource":"*"}}`)
	if fmt.Sprint(one) != "[Allow * -> *]" {
		t.Errorf("single statement object: %q", one)
	}
}

func TestDiffWithRedactsDiffLinesAndFullValues(t *testing.T) {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	before, after := enc("echo v1\nexport TOKEN=old-secret\n"), enc("echo v2\nexport TOKEN=new-secret\n")
	hide := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "old-secret", "X"), "new-secret", "X")
	}

	plain := DiffWith("user_data", before, after, Options{})
	if !strings.Contains(strings.Join(plain.Lines, "\n"), "old-secret") {
		t.Fatal("test is vacuous: without redaction the decoded secret is in the diff")
	}
	d := DiffWith("user_data", before, after, Options{Redact: hide})
	all := strings.Join(d.Lines, "\n") + d.FullBefore + d.FullAfter
	if strings.Contains(all, "old-secret") || strings.Contains(all, "new-secret") {
		t.Errorf("secret survived in diff lines or in FullBefore/FullAfter (used by the HTML expand toggle):\n%s", all)
	}
	if !strings.Contains(all, "echo v1") || !strings.Contains(all, "echo v2") {
		t.Errorf("ordinary lines were lost:\n%s", all)
	}
}

func TestDiffWithNoDecodeKeepsTheValueOpaque(t *testing.T) {
	enc := base64.StdEncoding.EncodeToString([]byte("export TOKEN=hunter2\n"))
	d := DiffWith("user_data", enc, enc+"AAAA", Options{NoDecode: true})
	if strings.Contains(strings.Join(d.Lines, "\n")+d.FullBefore, "hunter2") {
		t.Error("NoDecode must not decode base64")
	}
	if !strings.Contains(d.FullBefore, enc) {
		t.Errorf("FullBefore should be the raw value, got %q", d.FullBefore)
	}
}
