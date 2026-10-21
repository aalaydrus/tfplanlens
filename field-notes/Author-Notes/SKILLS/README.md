# Skills for agents (Claude, Kiro)

Three skills that let an agent use the tfplanlens CLI instead of reading raw
Terraform plans. Each is a folder with a `SKILL.md` (YAML frontmatter with
`name` and `description`, then instructions).

| Skill | Use it when |
|---|---|
| [`review-terraform-plan`](review-terraform-plan/SKILL.md) | "what will this plan do / is it safe to apply?" |
| [`compare-terraform-plans`](compare-terraform-plans/SKILL.md) | "what changed since the last plan?" |
| [`gate-terraform-plans`](gate-terraform-plans/SKILL.md) | "block risky plans in CI" |

## For the agent itself

[`CLAUDE.md`](CLAUDE.md) and [`KIRO.md`](KIRO.md) tell an agent how to find its
way around the repo, contribute from its own fork, and write skills for itself in
its tool's format. Point the agent at the file for its tool.

## Install

The skills **call the `tfplanlens` binary**, which must be on the PATH of the
agent's shell: build it from a clone (`go build -o tfplanlens ./cmd/tfplanlens`,
Go 1.25+) or `go install ./cmd/tfplanlens`. There are no published releases.

- **Claude Code**: copy the three folders into `~/.claude/skills/` (all
  projects) or `<project>/.claude/skills/` (one project). Claude then loads a
  skill when its description matches the task.
- **Kiro**: not verified here. The `SKILL.md` bodies are plain markdown, so they
  can be used as steering documents (Kiro reads markdown steering files from
  `.kiro/steering/`) by copying the body and dropping the frontmatter; check
  Kiro's current docs for its own skill/power format.

## Try one before trusting it

```
cd simulation        # from the repo root
tfplanlens report assets/input/plan.json --no-color
tfplanlens diff assets/input/plan.json assets/input/plan-revised.json
```

(`simulation/rundown/EXAMPLE_RUNDOWN.md` explains the output.) `TestAgentDocs*`
checks that every `tfplanlens ...` command in these skills uses real commands
and flags, so they stay in step with the CLI.

## Not turned into skills (and why)

- **`tui`**: interactive, needs a terminal; an agent cannot drive it.
- **HTML report**: for humans; the skills tell agents to prefer text/JSON.

## Ideas

- A skill that explains one finding in depth (why a replace, what depends on
  it) using `-f json` and `jq`.
- A skill that triages drift into "real" and "provider refresh artifact" once
  the tool can label them ([../../WISHLIST.md](../../WISHLIST.md) item 4).
- A skill for the planned `--aws` checks, if they ship.
