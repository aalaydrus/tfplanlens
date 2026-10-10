<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 2 |
| update | 2 |
| replace | 0 |
| delete | 0 |
| read | 1 |
| no-op | 1 |
| Drift | 0 |
| Known after apply | 4 attribute(s) |

> **Notice:** Apply account unresolved (set --tfapply-account-id): cross-account flags are off.

### Danger

No deletes or replacements.

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; create 2, update 2, read 1, no-op 1</summary>

- `aws_s3_bucket.logs` **STATEFUL** (create)
- `aws_instance.web` (create)
- `aws_instance.api` (update)
  - `instance_type`: `"t3.micro"` → `"t3.small"`
- `aws_security_group.web` (update)
  - `tags.b`: `(absent)` → `"2"`
- `data.aws_caller_identity.current` (read)

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 5 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| create | `aws_s3_bucket.logs` | `unresolved (alias: default)` (unresolved) |
| create | `aws_instance.web` | `unresolved (alias: default)` (unresolved) |
| update | `aws_instance.api` | `unresolved (alias: default)` (unresolved) |
| update | `aws_security_group.web` | `unresolved (alias: default)` (unresolved) |
| read | `data.aws_caller_identity.current` | `unresolved (alias: default)` (unresolved) |

</details>

