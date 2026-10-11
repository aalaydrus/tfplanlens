# For Kiro: navigate, contribute from a fork, write your own skills

**Not verified:** this repo's author has not tested these files in Kiro. Where
Kiro's own format matters (where files go, what frontmatter it wants), check
Kiro's current documentation first; the steps below say where that applies.

## 1. Navigate cheaply

Read [`../AGENTS_START_HERE.md`](../AGENTS_START_HERE.md) first (about 4 KB). It
routes your task to one ref or playbook; do not read the rest. Never print plan
JSON files: they are large and contain secrets in plain text. Use the `jq`
recipes in [`../refs/DO_NOT_READ.md`](../refs/DO_NOT_READ.md).

To have Kiro read this automatically, link or copy the start page into your
project's Kiro steering (Kiro reads markdown steering files from
`.kiro/steering/`; confirm the current location and any inclusion settings in
Kiro's docs).

## 2. Contribute from your own fork

Do not push to the upstream repository. Work in a fork and open the PR from it.

```
gh repo fork aalaydrus/tfplanlens --clone
cd tfplanlens
git remote -v                       # origin = your fork, upstream = aalaydrus/tfplanlens
git fetch upstream
git checkout -b fix/short-name upstream/main
```

If you have no `gh`, fork in the GitHub UI and clone your fork, then
`git remote add upstream https://github.com/aalaydrus/tfplanlens.git`.

Follow the playbook for your task (the table in the start page). Before every
push:

```
go vet ./... && go test ./... -count=1
GOOS=windows go vet ./...
```

Push to **your fork's** branch and open a PR against `aalaydrus/tfplanlens`'s
default branch, using [`../playbooks/open-a-pr.md`](../playbooks/open-a-pr.md):
what changed, what you verified, and what you did **not** verify. Only open a PR
if the person you work for asked for one.

## 3. Write skills (or steering) for yourself

The skills in this folder are plain markdown with a short frontmatter
(`name`, `description`). Use them as the content source:

1. Pick one ([`review-terraform-plan`](review-terraform-plan/SKILL.md) is the
   best template) and read what Kiro currently expects for skills/powers or
   steering files: file location, required metadata, how it decides when to
   load them.
2. Adapt the **body** (instructions) as is and re-express the frontmatter in
   Kiro's format. Keep the content rules:
   - every command must be one you ran, e.g. on
     `simulation/assets/input/plan.json`;
   - say what to do when the tool cannot tell, rather than guessing;
   - never include secrets or real plan files; plan JSON stays out of chat.
3. Check it triggers on a realistic prompt in a fresh Kiro session.
4. To contribute it back, put your version in a new subfolder here in your fork
   (for example `SKILLS/kiro-<skill-name>/`) with a note on what you verified,
   and open the PR as above. Once Kiro's format is confirmed, record it in this
   file and drop the "not verified" line.

Skills here assume the `tfplanlens` binary is on PATH; see
[`README.md`](README.md) for how to build it.
