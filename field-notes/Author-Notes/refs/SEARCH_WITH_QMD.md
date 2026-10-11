# Optional: searching these docs with qmd

[qmd](https://github.com/tobi/qmd) is an on-device search engine for markdown
(BM25 keyword search, optional vector search and an LLM reranker, all local,
with `--json` output meant for agents). We tried it on this repo to see whether
it cuts the tokens an agent spends finding things. **Short answer: BM25 search
is cheap per query, but it is not miraculous and not needed here.** The routing
page ([../AGENTS_START_HERE.md](../AGENTS_START_HERE.md)) already gets the cost
down; qmd is an optional extra.

## What we ran (reproducible)

```
npm install @tobilu/qmd                        # in a scratch dir: 159 packages, ~20 s
qmd collection add . --name tfplanlens --mask "**/*.md"    # 59 files, 0.3 s, no models
QMD=path/to/qmd python3 scripts/qmd-experiment.py           # natural-language queries
QMD=path/to/qmd python3 scripts/qmd-experiment.py --tuned   # 1-2 word queries
```

`scripts/qmd-experiment.py` asks ten questions an agent might ask (where is X
documented, which test guards Y, how do I add a flag...) and measures the bytes
of text the agent has to read (about 4 bytes per token) to find the answer by:
`grep`, the guide (start page + the one file it routes to), and `qmd search -n 3`.
"Answer found" means the text read contains the actual answer.

## Results (average per question; 59 markdown files, 176 KB in all)

| Approach | Bytes read | Answer found (of 10) | Notes |
|---|---|---|---|
| Read README + SPEC (the old habit) | 32,309 | 10 | fixed cost per task |
| `grep` the markdown, 2 lines of context | 7,546 | 9 | huge variance: 744 to 31,250 |
| **Guide**: start page + routed file | 6,900 | 10 | the route was chosen from the routing table |
| `qmd search -n 3`, natural-language query | 1,177 | 5 (right file in top 3: 6) | one query (`contribute fork pull request`) returned nothing |
| `qmd search -n 3`, 1-2 word query | 1,301 | 6 (right file in top 3: 8) | queries tuned after seeing the first run |
| qmd search, then read the matching file | 2,864 to 3,344 | up to 8 | |

## What we learned

- **Extra words can make the keyword search return nothing.** `contribute fork
  pull request` returned 0 results while `fork` and `fork OR contribute` worked,
  which fits "every word must match" (I did not read the source to confirm).
  Agents should search one or two specific words, or use `OR`.
- **Snippets are tiny and precise** (about 1.3 KB for three hits, with file and
  line), so a search plus `qmd get file:from:count` can cost far less than
  reading a file. When the right file is found, search + read is about 3 KB
  against 7 KB for the guide.
- **But BM25 misses more often than the guide**: 5 to 6 answers in the snippets
  against 10 for the guide (8 with the right file in the top three, which then
  needs a read). A miss costs a retry, which eats the saving.
- **The guide is already most of the saving.** Going from 32 KB to 7 KB per
  task came from routing, not search. qmd would take it from 7 KB to perhaps
  3 KB when it finds the file, at the price of a Node toolchain.
- It indexes everything you point it at, including generated files
  (`docs/smoke`, `simulation/output`): all 59 markdown files here. A larger or
  noisier corpus would favour search more than this small one does.

## Not tested

- **Hybrid and vector search and the reranker** (`qmd query`, `qmd vsearch`,
  `qmd embed`): they download GGUF models from Hugging Face, which was not
  reachable from the sandbox. The numbers above are BM25 only. Semantic search
  would probably help the questions whose words do not appear in the docs
  ("contribute" vs "fork"), but that is a guess.
- The MCP server, qmd's context annotations (`qmd context add`), and use on a
  corpus much larger than this repo.
- A real agent doing a real task: the questions, answer files and queries are
  ours, written knowing the docs, so the guide is flattered. Treat the table
  as a rough guide.

## Recommendation

Do not add qmd as a dependency or require it. If an agent already has qmd
installed, this is how to use it here:

```
qmd collection add . --name tfplanlens --mask "**/*.md"
qmd search "fork" -n 3 --json            # 1-2 words, or "a OR b"
qmd get "field-notes/Author-Notes/SKILLS/CLAUDE.md:1:40"    # read a slice, not the file
```

Revisit when the docs grow (a hundred files or more), or once someone has run
`qmd query` with models available and measured it.
