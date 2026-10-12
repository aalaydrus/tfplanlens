# Wishlist and edge cases to dig into

Status: `open`, `exploring` (being looked at now), `hypothesis` (not yet checked), `done`.

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

## Redaction (docs/SECRETS.md)

16. **More credential formats** (`open`): Azure/GCP keys, database URLs without
    a user, Terraform Cloud tokens, `.pem` files without a header.
17. **Entropy check** (`hypothesis`): flag long random-looking strings in
    attributes with innocent names. Likely too noisy; measure on real plans first.
18. **Per-type or per-attribute allow/deny lists** in the config (`open`).
19. **Make hidden values diffable** (`open`): `diff` cannot say a hidden value
    changed; a keyed hash of the value would, at the cost of a new thing to protect.
20. **`verify-offline.sh` in CI** (`open`): needs `strace`/`unshare` on the runner.

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

## From user feedback

Both items came from one reader of the public repo, who described a dev
account where teammates deploy over each other and other teams use different
IaC to refer to shared resources (an RDS or Redshift cluster).

21. **Link each change to the code change behind it** (`exploring`). A plan has
    no git data, so this needs input from the user's side. Two levels:
    - *Run level* (first, small): `--commit <sha>` and `--repo-url <url>` (or
      config keys; CI can pass `GITHUB_SHA` and `GITHUB_REPOSITORY`). The report
      header links the commit, and `diff` can link the range between two plans.
      Stays fully offline.
    - *Per resource* (later, heuristic): `--source-dir` finds the
      `resource "type" "name"` block for each address and runs `git log` or
      `git blame` on its lines. Needs a checkout. Modules and `for_each`
      instances need mapping. A change can come from a variable, a `.tfvars`
      file or a provider upgrade rather than the block, and drift has no code
      cause: say "unknown", never blame a guess.
22. **Show who would be affected by a change to a shared resource**
    (`exploring`). A plan cannot see consumers outside the stack (other teams'
    IaC reading your cluster), so this needs a declared map: config rules
    matching resource type, address or tag to the teams that use it and who to
    contact, shown next to the matching Danger and `diff` entries. Fall back
    to an owner tag in the plan when there is one, otherwise "no owner
    declared". Plans you pass together could add cross-stack references, but
    only for the stacks you supply. Concurrent applies in a shared dev account
    are mostly a state-locking and drift problem; the drift view covers part of
    it, and a plan does not say when the last apply happened (`hypothesis`).

## From more user feedback

A second reader suggested running tfplanlens at plan time, using it in code
review agents (fewer tokens than querying source control), and ranking changes
by how much they matter.

23. **Rank by how many resources depend on the change** (`exploring`). An SSM
    parameter that 12 other resources reference matters more than one nothing
    uses. The plan's `configuration` already carries the reference edges used
    for apply waves. Limits: it only sees references inside this plan; `local.*`
    and other stacks are invisible, so the number is a lower bound ("at least
    12") and must never be shown as exact.
24. **Deletion protection and final snapshot on a delete or replace**
    (`exploring`). Destroying a DB cluster with deletion protection on is
    refused by AWS, without it the data is gone. The flag is in the resource's
    `before` values (for example `deletion_protection` on RDS instances and
    clusters). Needs a per-type table of attribute names; unknown types say
    "unknown". `hypothesis`: attribute names are not yet checked against a real
    plan, and the emulator we use may not produce RDS clusters (see item 13).
25. **Compact, ranked output for review agents** (`open`). The top N changes
    with the reason each matters and nothing else, so an agent does not spend
    tokens on the rest. Builds on items 23 and 24; `-f json` and the markdown
    PR comment already put Danger first.
26. **Run at plan time** (`open`). Terraform has no hook in `plan` itself that
    we know of (not searched exhaustively). Recipes to document, none run yet:
    a wrapper (`terraform show -json p | tfplanlens report -`), a Terragrunt
    `after_hook`, an Atlantis custom plan step (a post-workflow hook does not
    post its output to the PR). An HCP Terraform post-plan run task sends a
    webhook with a plan JSON URL and expects a callback, so it needs a small
    web service in front of the CLI.
