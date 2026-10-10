<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 1 |
| update | 2 |
| replace | 0 |
| delete | 0 |
| Drift | 0 |
| Known after apply | 0 attribute(s) |

> **Notice:** Apply account unknown: set tfapply_account_id in tfplanlens.config.yaml (it cannot be read from the plan). Cross-account flags are off.

### Danger

No deletes or replacements.

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; create 1, update 2</summary>

- `aws_db_instance.main` **STATEFUL** (update)
  - `password`: (sensitive value changed)
- `aws_secretsmanager_secret_version.s` (create)
- `aws_ssm_parameter.mixed` (update)
  - `tags.k`: `"v1"` → `"v2"`
  - `value`: (sensitive value changed)

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 3 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| update | `aws_db_instance.main` | `unresolved (alias: default)` (unresolved) |
| create | `aws_secretsmanager_secret_version.s` | `unresolved (alias: default)` (unresolved) |
| update | `aws_ssm_parameter.mixed` | `unresolved (alias: default)` (unresolved) |

</details>

