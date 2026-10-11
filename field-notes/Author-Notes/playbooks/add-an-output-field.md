# Playbook: show a new fact in the reports

Order matters: model first, renderers last.

1. **Model**: add the field to `internal/analyze/model.go` with a json tag
   (`omitempty` for optional scalars; **initialize slices to `[]`** in `Analyze`
   so JSON never has `null`). Add new top-level array keys to `arrayKeys` in
   `TestJSONArraysAreNeverNull` (`cmd/tfplanlens/real_plans_test.go`).
2. **Compute it** in `internal/analyze` (a new file for anything over ~50 lines,
   called from `Analyze` in `analyze.go`). Merged reports: if addresses appear
   in it, prefix them in `merge.go`; if the new field is a count, add it there.
3. **Unit-test** it in `internal/analyze` with a hand-built `plan.Plan`, then
   against a real plan in `cmd/tfplanlens/real_plans_test.go`.
4. **Render** it (only if a human should see it):
   - text: `internal/render/text/text.go` (`Render` calls the sections in order;
     Danger must stay first).
   - markdown: `internal/render/markdown/markdown.go` (`Render`; use `w.line`,
     `code`, `cell`, `closeDetails` so the 60,000-byte cap and escaping hold).
   - html: `internal/render/html/report.html` (+ `html.go` `buildView`,
     `app.js` only if interactive). html/template escapes for you; never
     build HTML by string concatenation.
   - json: automatic from the struct.
   - tui: `internal/tui/tui.go` (`changeNode`/`wavesTree`).
5. **Document**: `docs/json-schema.md` (every JSON field), `docs/USAGE.md` if
   user-visible (the README stays a short summary: do not grow it).
6. **Goldens**: `go test ./cmd/... -run TestGolden -update`; read the diff.
   JSON goldens change for every fixture when you add a top-level field
   (expected: a one-line addition each).
7. If it is shown in the simulation: `simulation/demo.sh` and commit
   `simulation/output` (`TestSimulationOutputsAreCurrent` fails otherwise).
8. Never print a value from a sensitive attribute; tests exist, run them.
