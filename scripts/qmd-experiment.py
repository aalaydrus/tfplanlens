#!/usr/bin/env python3
"""Does qmd (https://github.com/tobi/qmd) save an agent tokens on this repo's docs?

Compares, for ten questions an agent might ask, what it costs (bytes of text
the agent must read; about 4 bytes per token) to find the answer by:

  grep    grep -rniE over the markdown (a common agent move)
  guide   field-notes/Author-Notes/AGENTS_START_HERE.md plus the one file its
          routing table points to
  qmd     `qmd search -n 3 --json` (BM25 keyword search, no models), and
          search plus reading the first matching answer file in full

"ans" means the text the agent would read contains the actual answer (a regex
per question, ANSWERS below). Run from the repo root:

  npm install @tobilu/qmd                       # in a scratch dir
  QMD=/path/to/node_modules/.bin/qmd
  $QMD collection add . --name tfplanlens --mask "**/*.md"
  QMD=$QMD python3 scripts/qmd-experiment.py            # natural-language queries
  QMD=$QMD python3 scripts/qmd-experiment.py --tuned    # 1-2 word queries

The questions, answer files and queries are the author's own, written with
knowledge of the docs: treat the result as a rough guide, not a benchmark.
"""
import json, os, re, subprocess, sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
QMD = os.environ.get("QMD", "qmd")
TUNED = "--tuned" in sys.argv
A = "field-notes/Author-Notes/"

# question, natural query, short query, grep keywords, answer regex,
# files that contain the answer, file the start page routes to
CASES = [
    ("Where are the apply-account precedence rules?", "apply account precedence", "precedence",
     ["precedence", "apply account"], r"precedence", ["docs/ACCOUNTS.md", A + "refs/CLI_REFERENCE.md"], A + "refs/CLI_REFERENCE.md"),
    ("Which test guards 'JSON arrays are never null'?", "JSON arrays never null test", "never null",
     ["never null", "NeverNull"], r"TestJSONArraysAreNeverNull", [A + "AGENTS_START_HERE.md", A + "refs/TESTING_MAP.md"], A + "refs/TESTING_MAP.md"),
    ("How do I add a new CLI flag?", "add a CLI flag", "add flag",
     ["add a flag", "new flag", "CLI flag"], r"reportOptions|newReportCmd", [A + "playbooks/add-a-flag-or-config-key.md"], A + "playbooks/add-a-flag-or-config-key.md"),
    ("How do I list a plan's dependencies without opening it?", "jq references dependencies plan", "jq references",
     ["jq", "references"], r"jq -c '\.configuration", [A + "refs/DO_NOT_READ.md"], A + "refs/DO_NOT_READ.md"),
    ("Why is a password rotation shown as Danger?", "secret rotation replace danger", "secret rotation",
     ["secret_string", "rotat"], r"replace_because_cannot_update|Rotating a secret is a replace",
     ["field-notes/TERRAFORM_GOTCHAS.md", "field-notes/WISHLIST.md", "simulation/rundown/EXAMPLE_RUNDOWN.md"], "field-notes/TERRAFORM_GOTCHAS.md"),
    ("What are the exit codes?", "exit codes", "exit codes",
     ["exit code"], r"`1` a .{0,12}rule matched|`1` a rule matched|1` a `--fail-on", ["docs/USAGE.md", A + "refs/CLI_REFERENCE.md"], A + "refs/CLI_REFERENCE.md"),
    ("How do I contribute from a fork?", "contribute fork pull request", "fork",
     ["fork"], r"gh repo fork", [A + "SKILLS/CLAUDE.md", A + "SKILLS/KIRO.md"], A + "SKILLS/CLAUDE.md"),
    ("Why not Bubble Tea for the TUI?", "Bubble Tea terminal query", "Bubble Tea",
     ["Bubble Tea", "bubbletea"], r"queries the terminal|Bubble Tea.{0,40}init", ["field-notes/TERRAFORM_GOTCHAS.md", "SPEC.md"], "field-notes/TERRAFORM_GOTCHAS.md"),
    ("What does floci support / what are its quirks?", "floci quirks", "floci",
     ["floci"], r"fresh container|restart", ["field-notes/TERRAFORM_GOTCHAS.md", "field-notes/CONTRIBUTING_NOTES.md"], "field-notes/TERRAFORM_GOTCHAS.md"),
    ("Which tests cover the diff command?", "diff command tests", "TestRealPlanDiff",
     ["diff_test", "TestRealPlanDiff"], r"TestRealPlanDiff", [A + "refs/TESTING_MAP.md"], A + "refs/TESTING_MAP.md"),
]

def read(p):
    with open(os.path.join(REPO, p), encoding="utf-8") as f:
        return f.read()

def size(p):
    return os.path.getsize(os.path.join(REPO, p))

rows = []
for q, natural, short, kw, answer, files, route in CASES:
    pat = re.compile(answer, re.S)
    grep = subprocess.run(["grep", "-rniE", "-C2", "|".join(kw), "--include=*.md", "--exclude-dir=node_modules",
                           "--exclude-dir=.git", "."], cwd=REPO, capture_output=True, text=True).stdout
    guide_text = read(A + "AGENTS_START_HERE.md") + read(route)
    guide_bytes = size(A + "AGENTS_START_HERE.md") + size(route)
    out = subprocess.run([QMD, "search", short if TUNED else natural, "-n", "3", "--json"],
                         capture_output=True, text=True).stdout
    try:
        found = [r["file"].replace("qmd://tfplanlens/", "") for r in json.loads(out)]
    except ValueError:
        found = []
    read_extra = next((size(f) for f in found if f in files), 0)
    rows.append(dict(q=q, grep=len(grep), grep_ans=bool(pat.search(grep)),
                     guide=guide_bytes, guide_ans=bool(pat.search(guide_text)),
                     qmd=len(out), qmd_ans=bool(pat.search(out)), qmd_file=any(f in files for f in found),
                     qmd_read=len(out) + read_extra))

def yn(b): return "yes" if b else "no"
print(f"queries: {'short (1-2 words)' if TUNED else 'natural language'}")
print(f"{'question':56} {'grep':>6} {'ans':>3} | {'guide':>6} {'ans':>3} | {'qmd':>5} {'ans':>3} {'file':>4} {'+read':>6}")
for r in rows:
    print(f"{r['q'][:56]:56} {r['grep']:6} {yn(r['grep_ans']):>3} | {r['guide']:6} {yn(r['guide_ans']):>3} | "
          f"{r['qmd']:5} {yn(r['qmd_ans']):>3} {yn(r['qmd_file']):>4} {r['qmd_read']:6}")
n = len(rows)
avg = lambda k: sum(r[k] for r in rows) // n
cnt = lambda k: sum(r[k] for r in rows)
print(f"{'average bytes per question / answers found':56} {avg('grep'):6} {cnt('grep_ans'):>3} | {avg('guide'):6} {cnt('guide_ans'):>3} | "
      f"{avg('qmd'):5} {cnt('qmd_ans'):>3} {cnt('qmd_file'):>4} {avg('qmd_read'):6}")
