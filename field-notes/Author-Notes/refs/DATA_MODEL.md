# Data model (`internal/analyze/model.go`)

The JSON output **is** this struct; field docs are in `docs/json-schema.md`.

```
Analysis
  SchemaVersion, ApplyAccount{ID,Source,Resolved,Via}, Notices[]
  Summary{Mode,Applyable,Errored,TerraformVersion,Total,Actions,ByType,ByModule,
          KnownAfterApply,DriftCount,Warnings,ApplyWaves,DestroyWaves,Hidden,Redacted}
  Findings[]  danger: delete/replace (+ reason text, IndexShift warning, DriftAlso)
  Drift[]     out-of-band changes (+ HighRisk when also replaced/deleted)
  Groups[]    one per provider config key (target account + source + counts)
  Changes[]   every instance: Action, Account(+Source,CrossAccount), Attrs[], Wave*,
              Parent/Key (count/for_each), MovedFrom, Import, StateOnly, Warnings
  Parents[]   count/for_each resources with >1 instance
  Waves[], WaveNotes[], Cycles[]
  Units[]     only in merged reports (Unit label on Change/Finding/Drift)
```

Where each part is filled: everything in `Analyze` (`analyze.go`) except
waves (`waves.go`), parents (`instances.go`), merge (`merge.go`).
Attribute diffs come from `diffAttrs` (`attrs.go`); kinds are `value`,
`sensitive`, `long` (unknown values are only counted in `KnownAfterApply`).

Account sources: `flag`, `env`, `config`, `role_arn`, `role_arn_variable`,
`allowed_account_ids`, `caller_identity`, `apply_account`, `inferred`,
`mixed` (merged), `unresolved`.

Actions: `create update replace delete read no-op forget` (`plan.Action`).
