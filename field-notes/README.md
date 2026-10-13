# Field notes

Working notes that do not belong in the user docs: what we have learned from
real plans, what we want to do next, and how to work on this repo. (The folder
name is a placeholder; rename it freely, nothing links to it except the
simulation rundown.)

| File | What is in it |
|---|---|
| [`TERRAFORM_GOTCHAS.md`](TERRAFORM_GOTCHAS.md) | Plan JSON and provider behavior that surprised us, each with how it was verified |
| [`WISHLIST.md`](WISHLIST.md) | Ideas and edge cases to dig into, with a status |
| [`CONTRIBUTING_NOTES.md`](CONTRIBUTING_NOTES.md) | How we build, verify and review changes |
| [`Author-Notes/`](Author-Notes/AGENTS_START_HERE.md) | For agents: a routing page, code/test/CLI maps, task playbooks, and importable skills ([`SKILLS/`](Author-Notes/SKILLS/README.md)) |

Rules for entries: say **what you saw**, **where** (a file in this repo or the
Terraform/provider version), and mark it `verified` only if you ran it. A
guess is a `hypothesis`.

## Log of what has been done

Newest first. Details are in the PR descriptions and the files named.

- **Brief output for agents** (`-f brief --top N`): the top N changes that need
  attention, ranked, with reasons; `internal/analyze/brief.go`.
- **Danger side panel** in the HTML report: open by default, each delete or
  replace collapsed under its resource name.
- **Deletion protection and final snapshot** on deletes and replaces
  (`internal/analyze/safeguards.go`); DynamoDB checked in a real plan, RDS names
  unverified.
- **Dependents and ranking**: each delete or replace names what uses it; Danger
  is ordered stateful first, then by dependents.
- **Go 1.23 kept on purpose** after a Dependabot bump raised `go.mod` to 1.25.0;
  a test now guards it.
- **Release dry run**: `goreleaser check` and a local snapshot build pass
  (6 targets, checksums, Homebrew formula generated). The tag-triggered release
  workflow has not run, and there is no release yet.
- **GitHub Pages demo site**: an About page and the example reports
  (`docs/demo/`, `.github/workflows/pages.yml`), plus a guide and example
  workflow for users who want to publish their own (`docs/PAGES.md`).
- **Public launch**: repository public, branch protection on `main`, actions
  pinned to commit SHAs, Dependabot.
