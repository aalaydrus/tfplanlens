---
name: compare-terraform-plans
description: Compare two Terraform plans of the same configuration to see what a new commit changed. Use when the user pushed a fix and asks what changed since the last plan, whether a delete or replace went away or a new one appeared, or wants a re-review of only the difference. Runs `tfplanlens diff old.json new.json`.
---

# Compare two Terraform plans with tfplanlens

Use after a plan was reviewed and the author pushed a change: review the
**difference**, not everything again.

## Run

```
tfplanlens diff old-plan.json new-plan.json
tfplanlens diff old-plan.json new-plan.json -f markdown -o diff.md    # for a PR comment
tfplanlens diff old-plan.json new-plan.json --fail-on-new-danger      # exit 1 if new danger
```

Both files are `terraform show -json` output (see the review-terraform-plan
skill; never print them, they contain secrets). Add
`--tfapply-account-id <12 digits>` if the accounts matter.

## Read it in this order

1. **New danger**: addresses the new plan deletes or replaces that the old one
   did not. Always report these first.
2. **No longer deleted or replaced**: good news; confirm it is what the author
   meant to change.
3. **Only in the new plan / No longer in the plan / Changed**: each entry has
   `differences` (action, reason, target account, attributes that now or no
   longer change, before/after values; a rename done with a `moved` block is one
   changed entry, not removed plus added).
4. Plan-level differences (mode, Terraform version, apply account) and drift
   that appeared or went away.

Resources are matched by address; a plain no-op counts as "not changing", and
`N resource(s) change identically` are not listed. Sensitive values are never
compared: two sensitive changes to one attribute look equal.

## Report back

One line: did danger go up, down, or stay? Then the new danger, then the
changes that need a reviewer's attention, with exact addresses. If the diff is
empty say "no differences between the plans" (not "safe").
