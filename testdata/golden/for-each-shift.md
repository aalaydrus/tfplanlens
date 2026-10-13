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

> **Notice:** Apply account unknown: set tfapply_account_id in tfplanlens.config.yaml (it cannot be read from the plan). Cross-account flags are off.

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **DELETE** | `aws_s3_bucket.shards[1]` **STATEFUL** | `count shrank; index 1 no longer exists` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `aws_iam_user.team["carol"]` | `for_each key removed: carol` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `aws_instance.fleet[2]` | `count shrank; index 2 no longer exists`<br>**WARNING:** `likely count index shift: other instances of this resource are also changing (aws_instance.fleet[1])` | `unresolved (alias: default)` (source: unresolved) |

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

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 1 resource(s)</summary>

- delete `aws_instance.fleet` (2 instances)

</details>

<details><summary><b>destroy wave 1</b> &mdash; 2 resource(s)</summary>

- delete `aws_iam_user.team` (no configuration: order unknown)
- delete `aws_s3_bucket.shards` (no configuration: order unknown)

</details>


> Resources removed from the configuration have no dependency information in the plan; their destroy order is unknown and they are shown in wave 1.
### Resources with multiple instances

<details><summary>aws_iam_user.team (for_each) &mdash; 2 instance(s)</summary>

- delete `carol`
- no-op `alice`

</details>

<details><summary>aws_instance.fleet (count) &mdash; 3 instance(s)</summary>

- no-op `0`
- update `1`
- delete `2`

</details>

<details><summary>aws_s3_bucket.shards (count) &mdash; 2 instance(s)</summary>

- delete `1`
- no-op `0`

</details>

