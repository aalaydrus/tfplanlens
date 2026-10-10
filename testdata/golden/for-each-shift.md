<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 0 |
| update | 1 |
| replace | 0 |
| delete | 3 |
| no-op | 3 |
| Drift | 0 |
| Known after apply | 0 attribute(s) |

> **Notice:** Apply account unresolved (set --tfapply-account-id): cross-account flags are off.

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **DELETE** | `aws_iam_user.team["carol"]` | `for_each key removed: carol` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `aws_instance.fleet[2]` | `count shrank; index 2 no longer exists`<br>**WARNING:** `likely count/for_each index shift: sibling instances also changing (aws_instance.fleet[1])` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `aws_s3_bucket.shards[1]` **STATEFUL** | `count shrank; index 1 no longer exists` | `unresolved (alias: default)` (source: unresolved) |

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; update 1, delete 3, no-op 3</summary>

- `aws_iam_user.team["carol"]` (delete)
- `aws_instance.fleet[1]` (update)
  - `tags.Name`: `"b"` → `"c"`
- `aws_instance.fleet[2]` (delete)
- `aws_s3_bucket.shards[1]` **STATEFUL** (delete)

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 4 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| delete | `aws_iam_user.team["carol"]` | `unresolved (alias: default)` (unresolved) |
| update | `aws_instance.fleet[1]` | `unresolved (alias: default)` (unresolved) |
| delete | `aws_instance.fleet[2]` | `unresolved (alias: default)` (unresolved) |
| delete | `aws_s3_bucket.shards[1]` | `unresolved (alias: default)` (unresolved) |

</details>

