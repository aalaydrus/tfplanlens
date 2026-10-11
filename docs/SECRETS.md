# Secrets: what tfplanlens hides, and what it cannot

A plan file is as sensitive as Terraform state. It holds secret values in plain
text (variables, sensitive attributes, passwords). tfplanlens reads it offline
and prints a report; this page says what ends up in that report. **It is a
best-effort filter, not a guarantee.** Read the report before you share it.

## What is hidden

| Layer | What | Shown as |
|---|---|---|
| 1. Marked sensitive | Anything Terraform marks sensitive. Never printed, in any format, in `diff` or in the `tui`. | `(sensitive value changed)` |
| 2. Secret-named attributes | An attribute not marked sensitive whose name looks like a secret (`password`, `token`, `api_key`, `secret`, `private_key`, `access_key`, `credential`, ...), including env vars such as `DB_PASSWORD`. Names that describe a secret rather than hold one (`secret_arn`, `token_validity_days`, `password_length`) are left alone. | `(sensitive value changed)` |
| 3. Secret-looking content | In any value, including the **decoded** base64/gzip/JSON of `user_data` and other long values: private key blocks, AWS access key ids, GitHub/Slack/JWT-style tokens, credentials in URLs (`postgres://user:pw@host`), and `NAME=value` / `name: value` / `"name": "value"` lines where the name looks like a secret. References such as `$DB_PASSWORD` or `${var.db_password}` are not secrets and stay. | `[REDACTED]` (the name stays: `export DB_PASSWORD=[REDACTED]`) |

When layers 2 or 3 hide something, the report says so in a notice and in
`summary.redacted` (JSON), for example "Hid secret-looking values in 10 place(s)".

## What is not hidden

- **A secret with an innocent name and no recognizable shape.** A random string
  in an attribute called `value` or `config`, a token format we do not know, a
  secret in a free-text `description`. Layers 2 and 3 are pattern matching.
- **Names and keys:** resource addresses, `for_each` keys, tag keys, import ids.
  A secret used as a key is printed.
- Everything Terraform itself prints, that tfplanlens has no reason to hide:
  account ids, ARNs, resource names, non-secret attribute values.
- Two changed values that both get hidden look identical: tfplanlens cannot tell
  you the (hidden) value changed. Sensitive values are compared the same way.

## Settings

In `tfplanlens.config.yaml`:

```yaml
redact_secrets: true    # default. false shows layers 2 and 3 (layer 1 is always on)
decode_values: true     # default. false stops decoding base64/gzip/JSON in long values
```

`decode_values: false` leaves an encoded `user_data` as the opaque string
Terraform shows. Redaction still applies to whatever is shown. Turn
`redact_secrets` off only for plans you know hold no secrets.

## What the tool does not do

It does not use the network, start other programs, or write files other than the
`--out` target; it reads the plan files and `./tfplanlens.config.yaml`. This is
checked, not assumed: `scripts/verify-offline.sh` runs the tool on the example
plans in an empty network namespace under `strace` and fails on any socket call,
other program started or unexpected write (Linux; needs `strace` and `unshare`).
It has been run on Linux only.

## What you still have to do

- Keep plan files (`plan.json`, `plan.bin`) and state out of version control, CI
  artifacts and chat. The report is safer to share than the plan, not safe by
  default: skim it before posting it as a pull request comment.
- Report a secret format we miss: see [SECURITY.md](../SECURITY.md).

## How it is tested

`TestUnmarkedSecretsNeverReachAnyOutput` (a plan with unmarked secrets: none may
appear in text, markdown, HTML including its "expand full value" data, JSON or
`diff`), `TestRedactionLeavesTheRealPlansAlone` (no false positives on the real
and simulation plans), and unit tests in `internal/redact`.
