package redact

import (
	"strings"
	"testing"
)

func TestSecretName(t *testing.T) {
	for _, k := range []string{"password", "db_password", "DB_PASSWORD", "master_password_note", "API_TOKEN", "auth_token", "api-key",
		"apikey", "private_key", "access_key", "client_secret", "secret_string", "credentials", "Authorization", "passphrase"} {
		if !SecretName(k) {
			t.Errorf("%q should look like a secret", k)
		}
	}
	for _, k := range []string{"name", "description", "secret_arn", "secret_id", "token_validity_days", "password_length",
		"secret_version", "password_enabled", "log_level", "identifier", "kms_key_id", "access_key_id_arn", "tags"} {
		if SecretName(k) {
			t.Errorf("%q describes a secret, it does not hold one", k)
		}
	}
}

func TestTextHidesSecretShapes(t *testing.T) {
	for name, tc := range map[string]struct{ in, hidden string }{
		"shell export":      {"export DB_PASSWORD=hunter2", "hunter2"},
		"yaml":              {"db_password: hunter2", "hunter2"},
		"json":              {`{"password": "hunter2", "user": "app"}`, "hunter2"},
		"minified json":     {`{"api_key":"hunter2"}`, "hunter2"},
		"single quoted":     {"TOKEN='hunter2'", "hunter2"},
		"aws key id":        {"key AKIAIOSFODNN7EXAMPLE here", "AKIAIOSFODNN7EXAMPLE"},
		"github token":      {"gh=ghp_abcdefghijklmnopqrstuvwxyz0123456789", "ghp_abcdefghijklmnopqrstuvwxyz0123456789"},
		"slack token":       {"xoxb-1234567890-abcdefghij", "xoxb-1234567890-abcdefghij"},
		"jwt":               {"auth eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig_part", "eyJhbGciOiJIUzI1NiJ9"},
		"url credentials":   {"postgres://app:hunter2@db.example.com/x", "hunter2"},
		"private key block": {"a\n-----BEGIN RSA PRIVATE KEY-----\nMIIEfake\nmore\n-----END RSA PRIVATE KEY-----\nb", "MIIEfake"},
		"unterminated key":  {"-----BEGIN PRIVATE KEY-----\nMIIEfake", "MIIEfake"},
	} {
		r := New()
		out := r.Text(tc.in)
		if strings.Contains(out, tc.hidden) || !strings.Contains(out, "[REDACTED]") || r.Count() == 0 {
			t.Errorf("%s: %q -> %q (count %d)", name, tc.in, out, r.Count())
		}
	}
}

func TestTextLeavesOrdinaryContentAlone(t *testing.T) {
	for _, s := range []string{
		"echo step-118", "Rotates the token every 30 days", "secretsmanager:GetSecretValue",
		`"Action": ["secretsmanager:GetSecretValue", "sqs:SendMessage"]`,
		"arn:aws:secretsmanager:us-east-1:111111111111:secret:orders/db-credentials-1LFT0S",
		"Allow secretsmanager:GetSecretValue -> arn:aws:secretsmanager:us-east-1:1:secret:x",
		"export DB_PASSWORD=$DB_PASSWORD", "password = ${var.db_password}", "token: <your token>", "secret_arn = arn:aws:x",
		"token_validity_days: 30", "password_length = 16", "password=", `"password": ""`, "plain text with no assignment",
	} {
		r := New()
		if out := r.Text(s); out != s || r.Count() != 0 {
			t.Errorf("changed %q -> %q", s, out)
		}
	}
}

func TestNilRedactorHidesNothing(t *testing.T) {
	var r *Redactor
	if r.Text("password=hunter2") != "password=hunter2" || r.Count() != 0 || r.Value("x") != "x" {
		t.Error("a nil Redactor must pass values through")
	}
	r.Hit() // must not panic
}

func TestValueOnlyTouchesStrings(t *testing.T) {
	r := New()
	if r.Value(30) != 30 || r.Value(true) != true || r.Value(nil) != nil {
		t.Error("non-strings must be returned unchanged")
	}
	if got := r.Value("password=hunter2"); got != "password=[REDACTED]" {
		t.Errorf("got %v", got)
	}
}
