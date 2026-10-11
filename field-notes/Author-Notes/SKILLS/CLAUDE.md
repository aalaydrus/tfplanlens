# For Claude (Claude Code): navigate, contribute from a fork, write your own skills

You are reading this because you were asked to use, extend or contribute to
tfplanlens. Three jobs, in order of how often they come up.

## 1. Navigate cheaply

Read [`../AGENTS_START_HERE.md`](../AGENTS_START_HERE.md) first (about 4 KB). It
routes your task to one ref or playbook; do not read the rest. Never `cat` plan
JSON files: they are large and contain secrets in plain text. Use the `jq`
recipes in [`../refs/DO_NOT_READ.md`](../refs/DO_NOT_READ.md).

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

Then follow the playbook for your task (the table in the start page), and
before every push:

```
go vet ./... && go test ./... -count=1
GOOS=windows go vet ./...
```

Push to **your fork's** branch and open a PR against `aalaydrus/tfplanlens`'s
default branch, using [`../playbooks/open-a-pr.md`](../playbooks/open-a-pr.md):
what changed, what you verified (commands or CI links), and what you did **not**
verify. Do not open a PR unless the person you work for asked for one. Keep
your fork current with `git fetch upstream` before starting new work.

## 3. Write skills for yourself

A Claude Code skill is a folder with a `SKILL.md`:

```
<skills-dir>/<skill-name>/SKILL.md      # name in the frontmatter == folder name
```

`<skills-dir>` is `~/.claude/skills/` (all your projects) or
`<project>/.claude/skills/` (one project). The file starts with exactly:

```
---
name: <skill-name>
description: <one or two sentences: WHAT it does and WHEN to use it, naming the tool>
---
```

then the instructions. The description is what makes Claude load the skill, so
put the trigger words a person would actually say in it.

Method (the one that produced the three skills here):

1. Start from [`review-terraform-plan/SKILL.md`](review-terraform-plan/SKILL.md)
   as a template; keep its structure (get the tool, get the input safely, run,
   read in order, report back).
2. Every command in it must be one you ran: try it on
   `simulation/assets/input/plan.json` and read the output.
3. Say what to do when the tool cannot tell (unknown accounts, unknown values)
   rather than letting the skill guess.
4. Never put secrets, tokens or real plan files in a skill; tell the user to
   keep plan JSON out of chat and artifacts.
5. Trigger it with a realistic prompt in a fresh session and check it loads and
   works. Tighten the description if it does not.
6. To contribute it back, add the folder under `field-notes/Author-Notes/SKILLS/`
   in your fork and run `go test ./cmd/... -run AgentSkills`, which checks the
   frontmatter and that every `tfplanlens ...` command in it is real.

Skills here assume the `tfplanlens` binary is on PATH; see
[`README.md`](README.md) for how to build it.
