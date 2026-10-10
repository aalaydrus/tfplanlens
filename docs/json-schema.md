# JSON output schema (`schema_version: 1`)

`tfplanlens report --format json` prints one object. Fields are only ever
added within a schema version; removing or retyping a field bumps
`schema_version`. Map keys are sorted and slices are never `null`, so output is
deterministic.

| Field | Type | Notes |
|---|---|---|
| `schema_version` | int | Always `1` for this schema. |
| `summary` | object | See below. |
| `findings` | array | Danger section: one entry per delete or replace, in plan order. |
| `drift` | array | Changes made outside Terraform (`resource_drift`). |
| `groups` | array | One per provider config key, sorted by key. |
| `changes` | array | Every resource change in plan order, including no-ops. |

## `summary`
`mode` (`normal` / `refresh-only` / `destroy`), `applyable`, `errored`,
`terraform_version`, `total`, `actions` (`create`, `update`, `replace`,
`delete`, `read`, `no_op`, `forget`), `by_type` and `by_module` (counts of
changes that do something; `root` is the root module), `known_after_apply`
(attribute count), `drift_count`, `hidden` (changes removed by `ignore_types`).

## `findings[]`
`address`, `type`, `action` (`delete` / `replace`), `provider`, `account`,
`stateful`, `reason` (raw `action_reason`), `reason_text` (plain English),
`replace_paths`, `index_shift` + `warning` (likely count/for_each shift),
`drift_also` (the resource also drifted: high risk).

## `drift[]`
`address`, `type`, `action`, `attrs`, `high_risk` (also replaced or deleted by
this plan), `note`.

## `groups[]`
`key`, `provider`, `alias`, `account` (12-digit ID or `unresolved (alias: x)`),
`resolved`, `source` (`role_arn` / `config` / `none`), `default`,
`cross_account`, `counts` (same shape as `summary.actions`), `addresses`.

## `changes[]`
`address`, `type`, `module`, `mode`, `action`, `provider`, `account`,
`cross_account`, `stateful`, `reason`, `reason_text`, `replace_paths`,
`moved_from`, `import`, `import_id`, `removed_from_state_only`, `attrs`,
`known_after_apply`. `attrs` is filled for `update` and `replace` only.

## `attrs[]`
`path` (`a.b[0].c`), `kind`, and `forces_replacement`.

- `value`: `before` / `after` hold the JSON values (absent = not present).
- `unknown`: the new value is known after apply; `before` may be set.
- `sensitive`: neither value is ever included.
- `long`: `diff` holds `header`, `lines` (changed lines plus 2 lines of
  context, capped at 60), `added`, `removed`, `total_lines`, `truncated_lines`.
