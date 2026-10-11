# Wishlist and edge cases to dig into

Status: `open`, `exploring`, `hypothesis` (not yet checked), `done`.

## From the simulation

1. **Benign replaces** (`open`). A password rotation shows as a Danger REPLACE
   (`aws_secretsmanager_secret_version`). Probably the same shape (`hypothesis`, not
   checked): SSM parameter versions, `aws_s3_object` content, Lambda
   `source_code_hash`. Idea: a built-in
   (and configurable) list of types whose replace is routine, shown below the
   real danger with the reason ("creates a new version"), still matched by
   `--fail-on replace`.
2. **Secret rotation without a replace** (`hypothesis`). The provider has
   write-only arguments (`secret_string_wo` plus `secret_string_wo_version`).
   Check whether bumping the version is an in-place update, and what the plan
   shows when the value is write-only (it will not be in the plan at all).
3. **"Sensitive value changed" for values that did not change** (`open`).
   `secret_binary` was empty before and after a replace but is reported as
   changed. Could say "(sensitive, unchanged)" when both sides are null/empty.
4. **Provider refresh drift** (`open`). Hide or label known provider artifacts
   (`inline_policy`, `versioning`) as "provider normalization", not drift.
5. **Replace noise** (`open`). Drop `x: false -> null` lines on a replace when
   the new value is a computed-optional default.

## Product

6. **HTML for `diff`** (`open`). Text/markdown/JSON only today.
7. **`diff` of merged reports** (`open`). `diff` compares exactly two plans.
8. **Cross-plan dependencies** (`open`). Waves are exact only within a plan.
9. **Plan prefix inside warning text** (`open`). Merged reports prefix
   addresses but not the addresses quoted inside a warning.
10. **`--aws` checks** (`open`, the last roadmap item). Read-only lookups for
    references to resources about to be replaced or deleted (security group
    attachments, role trust, KMS grants, DNS). Needs a design first; floci may
    not emulate every API.
11. **Live dogfooding** (`open`). Host our own report of the smoke plans, and
    run on our own test infrastructure.
12. **AI skill wrapper** around the local binary (Kiro, Claude) (`open`).
13. **A real, non-emulated AWS plan** (`open`). Everything so far is either
    `terraform_data` or floci. A scrubbed plan from a real stack would tell
    us the most.
14. **Large plans** (`open`). The 5,000-resource test exists but performance
    of waves/merge/diff is not measured.
15. **TUI polish** (`open`): mouse, live resize is polled every 200ms.
