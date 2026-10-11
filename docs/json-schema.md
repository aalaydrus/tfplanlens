# JSON output schema (`schema_version: 1`)

`tfplanlens report --format json` prints one object. Fields are only ever
added within a schema version; removing or retyping a field bumps
`schema_version`. Map keys are sorted and slices are never `null`, so output is
deterministic.

| Field | Type | Notes |
|---|---|---|
| `schema_version` | int | Always `1` for this schema. |
| `apply_account` | object | Where terraform apply runs: `id`, `source` (`flag` / `env` / `config` / `inferred` / `unresolved`), `resolved`, and for `inferred` also `via` (`config` / `allowed_account_ids` / `caller_identity`). |
| `notices` | array of string | Plan-wide warnings, e.g. an unresolved apply account. |
| `summary` | object | See below. |
| `findings` | array | Danger section: one entry per delete or replace, in plan order. |
| `drift` | array | Changes made outside Terraform (`resource_drift`). |
| `groups` | array | One per provider config key, sorted by key. |
| `changes` | array | Every resource change in plan order, including no-ops. |
| `parents` | array | count/for_each resources with more than one instance. |
| `waves` | array | Apply and destroy waves (possible parallelism). |
| `wave_notes` | array | Why the waves may be a lower bound. |
| `cycles` | array | Dependency cycles found in the configuration. |
| `units` | array | The plans of a merged report; empty for a single plan. |

## `summary`
`mode` (`normal` / `refresh-only` / `destroy`), `applyable`, `errored`,
`terraform_version`, `total`, `actions` (`create`, `update`, `replace`,
`delete`, `read`, `no_op`, `forget`), `by_type` and `by_module` (counts of
changes that do something; `root` is the root module), `known_after_apply`
(attribute count), `apply_waves` and `destroy_waves` (number of waves), `drift_count`, `warnings` (per-resource warning count), `hidden` (changes removed by `ignore_types`).

## `findings[]`
`address`, `type`, `action` (`delete` / `replace`), `provider`, `account`,
`stateful`, `account_source`, `reason` (raw `action_reason`), `reason_text` (plain
English), `replace_paths`, `index_shift` + `warning` (likely count/for_each shift),
`drift_also` (the resource also drifted: high risk).

## `drift[]`
`address`, `type`, `action`, `attrs`, `high_risk` (also replaced or deleted by
this plan), `note`.

## `groups[]`
One per provider config key. `key`, `provider`, `alias`, `account` (the **target**
account: 12-digit ID or `unresolved (alias: x)`), `resolved`, `source` (see
below), `default`, `cross_account` (target differs from the apply account),
`counts` (same shape as `summary.actions`), `addresses`.

Account `source` values: `flag`, `config`, `role_arn`, `role_arn_variable`,
`allowed_account_ids`, `caller_identity`, `apply_account`, `unresolved`.

## `changes[]`
`address`, `type`, `module`, `mode`, `action`, `provider`, `account` (target),
`account_source`, `cross_account`, `warnings` (`kind`, `path`, `message`;
currently `arn_account_mismatch`), `stateful`, `reason`, `reason_text`, `replace_paths`,
`moved_from`, `import`, `import_id`, `removed_from_state_only`, `attrs`,
`known_after_apply`, `parent` + `key` (count/for_each instances only), `wave` + `wave_phase` (absent when the change is in no wave). `attrs` is filled for `update` and `replace` only, and never lists attributes
that are known after apply: those are only counted in `known_after_apply`.

## `attrs[]`
`path` (`a.b[0].c`), `kind`, and `forces_replacement`.

- `value`: `before` / `after` hold the JSON values (absent = not present).
- `sensitive`: neither value is ever included.
- `long`: `diff` holds `header`, `lines` (changed lines plus 2 lines of
  context, capped at 60), `added`, `removed`, `total_lines`, `truncated_lines`.

## Merged reports (`units[]`)
With several plans, every address (`changes`, `findings`, `drift`, `groups`,
`parents`, `waves`, `cycles`, `by_module` keys) is prefixed `label:`, and
`changes[]`, `findings[]` and `drift[]` carry `unit` (the label). Each
`units[]` entry has `label`, `mode`, `terraform_version`, `apply_account` and
`actions`. `summary.mode` is `mixed` when the plans differ; the top-level
`apply_account` is the shared one, or `{"id": "(differs per plan)", "source":
"mixed"}`. Waves are joined by phase and number across plans.

## `parents[]`
`address` (configuration address without keys), `kind` (`count` / `for_each`),
`counts` (same shape as `summary.actions`), `instances[]` (`key`, `address`, `action`).

## `waves[]`
`phase` (`apply` / `destroy`), `number` (from 1), `resources[]`: `address`
(configuration address), `action` (most significant among the instances),
`actions` (instance counts per action), `instances`, `no_configuration`
(resource removed from the configuration, so its destroy order is unknown).

Waves come from the configuration's references and `depends_on`. A resource
is in wave 1 if it depends on no other changing resource, in wave 2 if it
depends only on wave 1, and so on. Destroy waves are the reverse: dependents
first. Terraform applies in parallel, so this is possible parallelism, not a
strict order. Waves are a lower bound: `local.*` references and
`replace_triggered_by` are not visible in the plan (see `wave_notes`).
Resources that only forget or do nothing are not in a wave.

## `cycles[]`
`resources`: the configuration addresses in one dependency cycle. Terraform
rejects cycles before it plans, so this only appears for incomplete plans.

# `tfplanlens diff` JSON

`tfplanlens diff old.json new.json -f json`. Arrays are never `null`.

| Field | Type | Meaning |
|---|---|---|
| `schema_version` | number | 1 |
| `plan` | array of strings | Plan-wide differences: mode, Terraform version, apply account. |
| `old`, `new` | object | Action counts of each plan (same shape as `summary.actions`). |
| `added` | array | Entries that change in the new plan only. |
| `removed` | array | Entries that changed in the old plan and no longer do. |
| `changed` | array | Entries that change in both, differently. |
| `unchanged` | number | Resources that change identically in both. |
| `new_danger` | array of addresses | Deleted or replaced by the new plan only. |
| `defused` | array of addresses | Deleted or replaced by the old plan only. |
| `drift_new`, `drift_gone` | array of addresses | Drift present in only one plan. |

Entry: `address`, `kind` (`added` / `removed` / `changed`), `old_action` and
`new_action` (absent when the resource is not changing), `differences` (plain
sentences: action, reason, target account, moved/import, and per attribute
"now changes", "no longer changes", kind or forces-replacement changes, and
before/after values), `danger` and `defused` (booleans, omitted when false).
Resources are matched by address; a plain no-op counts as not changing.
