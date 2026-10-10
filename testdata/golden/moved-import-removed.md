<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 0 |
| update | 2 |
| replace | 0 |
| delete | 0 |
| no-op | 2 |
| forget | 1 |
| Drift | 0 |
| Known after apply | 0 attribute(s) |

> **Notice:** Apply account unresolved (set --tfapply-account-id): cross-account flags are off.

### Danger

No deletes or replacements.

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; update 2, forget 1, no-op 2</summary>

- `aws_s3_bucket.new_name` **STATEFUL** (moved from aws_s3_bucket.old_name)
- `module.net.aws_vpc.main` (update, moved from aws_vpc.main)
  - `tags.m`: `(absent)` → `"1"`
- `aws_iam_role.imported` (import)
- `aws_instance.imported_update` (import + update)
  - `instance_type`: `"t3.micro"` → `"t3.small"`
- `aws_db_instance.forgotten` **STATEFUL** (removed from state only)

</details>

### Changes by module

<details><summary><b>module.net</b> &mdash; 1 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| update, moved from aws_vpc.main | `module.net.aws_vpc.main` | `unresolved (alias: default)` (unresolved) |

</details>

<details><summary><b>root</b> &mdash; 4 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| moved from aws_s3_bucket.old_name | `aws_s3_bucket.new_name` | `unresolved (alias: default)` (unresolved) |
| import | `aws_iam_role.imported` | `unresolved (alias: default)` (unresolved) |
| import + update | `aws_instance.imported_update` | `unresolved (alias: default)` (unresolved) |
| removed from state only | `aws_db_instance.forgotten` | `unresolved (alias: default)` (unresolved) |

</details>

