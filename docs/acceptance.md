# v0.1 acceptance criteria (SPEC section 9): how each is verified

Run everything with `go test ./...`. Test names below are the ones that fail if the criterion stops holding.

| # | Criterion | Status | Verified by |
|---|---|---|---|
| 1 | `report` runs on all fixtures with no errors | met | `TestEveryPlanEveryFormat` (`cmd/tfplanlens`): every hand-written fixture and every real Terraform plan, in text, markdown, json and html |
| 2 | Every `action_reason` has a plain-English explanation | met | `TestReasonText`, `TestEveryReasonRenderedInDangerSection` (all hand-written reasons, plus an unknown one shown raw); `TestRealReasonsPlan`, `TestRealSimplePlan`, `TestRealComplexPlan` (real keywords) |
| 3 | A 400-line `user_data` change shows only changed lines plus context | met | `TestGolden/long-user-data`, `TestDiffChangedLinesPlusContext` (`valuediff`), `TestRealComplexPlan` (real plan: `+2 -2` of 401) |
| 4 | Deletes and replaces appear in Danger **before any change listing** in text, html and markdown | met | `TestDangerBeforeAnyChangeListing` (every plan that has findings, in all three formats). A short summary (text, markdown) or the header and any notice (html) may come first; the criterion was amended from "before anything else" to match SPEC 5.8. |
| 5 | Resources grouped by account, with an `unresolved` fallback | met | `TestTargetAccountSources`, `TestUnresolvedApplyAccount`, `TestRealModuleProviderPlan` |
| 6 | `--fail-on delete` exits `1` when any delete exists, `0` otherwise | met | `TestFailOnDelete`, `TestFailOnScopedAndDrift`, `TestRealDestroyPlan`, `TestRealRefreshOnlyPlan` |
| 7 | HTML report opens offline and is a single file | met | `TestEveryFixtureRendersValidSelfContainedHTML` (well-formed, no http(s) or external references); the smoke workflow loads each HTML report from `file://` in headless Chromium and fails on any external request |
| 8 | Markdown stays under 60,000 characters on `large.json` | met | `TestLargePlanPerformance` (generated 5,000-resource plan), `TestCapAndTruncationNote`. The cap is 60,000 bytes, which is never fewer than characters. |
| 9 | README and LICENSE (MIT) present | met | files exist; `LICENSE` is the MIT license |
