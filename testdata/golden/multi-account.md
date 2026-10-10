<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 3 |
| update | 1 |
| replace | 0 |
| delete | 1 |
| Drift | 0 |
| Known after apply | 0 attribute(s) |

> **Notice:** Apply account unresolved (set --tfapply-account-id): cross-account flags are off.

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **DELETE** | `aws_db_instance.prod` **STATEFUL** | `Removed from config` | `111111111111` (source: role_arn) |

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; create 1</summary>

- `aws_s3_bucket.default` **STATEFUL** (create)

</details>

<details><summary><b>aws.mystery</b> &mdash; target account unresolved (alias: mystery) (source: unresolved) &mdash; create 1</summary>

- `aws_sqs_queue.eu` (create)

</details>

<details><summary><b>aws.prod</b> &mdash; target account 111111111111 (source: role_arn) &mdash; create 1, delete 1</summary>

- `aws_s3_bucket.prod` **STATEFUL** (create)
- `aws_db_instance.prod` **STATEFUL** (delete)

</details>

<details><summary><b>aws.shared</b> &mdash; target account 222222222222 (source: role_arn) &mdash; update 1</summary>

- `aws_s3_bucket.shared` **STATEFUL** (update)
  - `tags.x`: `(absent)` → `"y"`

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 5 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| create | `aws_s3_bucket.default` | `unresolved (alias: default)` (unresolved) |
| create | `aws_s3_bucket.prod` | `111111111111` (role_arn) |
| update | `aws_s3_bucket.shared` | `222222222222` (role_arn) |
| delete | `aws_db_instance.prod` | `111111111111` (role_arn) |
| create | `aws_sqs_queue.eu` | `unresolved (alias: mystery)` (unresolved) |

</details>

