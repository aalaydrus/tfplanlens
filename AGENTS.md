# AGENTS.md

Working on tfplanlens (a Go CLI that reads `terraform show -json` plans offline)?
Start with [`field-notes/Author-Notes/AGENTS_START_HERE.md`](field-notes/Author-Notes/AGENTS_START_HERE.md):
it routes each kind of task to the one file you need. Do not `cat` plan JSON
files (large, and they contain secrets); use the `jq` recipes there.
