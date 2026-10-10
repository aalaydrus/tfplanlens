<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 0 |
| update | 0 |
| replace | 6 |
| delete | 6 |
| Drift | 0 |
| Known after apply | 0 attribute(s) |

> **Notice:** Apply account unknown: set tfapply_account_id in tfplanlens.config.yaml (it cannot be read from the plan). Cross-account flags are off.

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **DELETE** | `aws_sqs_queue.removed` | `Removed from config` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `aws_sns_topic.switched` | `Resource switched between count/for_each/single`<br>**WARNING:** `count/for_each was added or removed on this resource; instances are being destroyed rather than moved` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `aws_instance.counted[2]` | `count shrank; index 2 no longer exists` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `aws_iam_user.keyed["bob"]` | `for_each key removed: bob` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `module.gone.aws_s3_bucket.data` **STATEFUL** | `Whole module removed or its call changed` | `unresolved (alias: default)` (source: unresolved) |
| **DELETE** | `aws_lambda_function.orphan` | `Moved to an address that no longer exists` | `unresolved (alias: default)` (source: unresolved) |
| **REPLACE** | `aws_db_instance.main` **STATEFUL** | `Attribute change forces replacement: storage_encrypted` | `unresolved (alias: default)` (source: unresolved) |
| **REPLACE** | `aws_instance.tainted` | `Resource is tainted` | `unresolved (alias: default)` (source: unresolved) |
| **REPLACE** | `aws_ebs_volume.requested` **STATEFUL** | `Replacement requested with -replace` | `unresolved (alias: default)` (source: unresolved) |
| **REPLACE** | `aws_lb.triggered` | `replace_triggered_by dependency changed` | `unresolved (alias: default)` (source: unresolved) |
| **REPLACE** | `aws_eip.mystery` | `some_future_reason` | `unresolved (alias: default)` (source: unresolved) |
| **REPLACE** | `aws_instance.create_before_destroy` | `Attribute change forces replacement: ami` | `unresolved (alias: default)` (source: unresolved) |

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; replace 6, delete 6</summary>

- `aws_sqs_queue.removed` (delete)
- `aws_sns_topic.switched` (delete)
- `aws_instance.counted[2]` (delete)
- `aws_iam_user.keyed["bob"]` (delete)
- `module.gone.aws_s3_bucket.data` **STATEFUL** (delete)
- `aws_lambda_function.orphan` (delete)
- `aws_db_instance.main` **STATEFUL** (replace)
  - `storage_encrypted`: `(absent)` → `true` (forces replacement)
- `aws_instance.tainted` (replace)
  - `name`: `"tainted"` → `"tainted-new"`
- `aws_ebs_volume.requested` **STATEFUL** (replace)
  - `name`: `"requested"` → `"requested-new"`
- `aws_lb.triggered` (replace)
  - `name`: `"triggered"` → `"triggered-new"`
- `aws_eip.mystery` (replace)
  - `name`: `"mystery"` → `"mystery-new"`
- `aws_instance.create_before_destroy` (replace)
  - `ami`: `"ami-0abc1234"` → `"ami-0new"` (forces replacement)

</details>

### Changes by module

<details><summary><b>module.gone</b> &mdash; 1 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| delete | `module.gone.aws_s3_bucket.data` | `unresolved (alias: default)` (unresolved) |

</details>

<details><summary><b>root</b> &mdash; 11 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| delete | `aws_sqs_queue.removed` | `unresolved (alias: default)` (unresolved) |
| delete | `aws_sns_topic.switched` | `unresolved (alias: default)` (unresolved) |
| delete | `aws_instance.counted[2]` | `unresolved (alias: default)` (unresolved) |
| delete | `aws_iam_user.keyed["bob"]` | `unresolved (alias: default)` (unresolved) |
| delete | `aws_lambda_function.orphan` | `unresolved (alias: default)` (unresolved) |
| replace | `aws_db_instance.main` | `unresolved (alias: default)` (unresolved) |
| replace | `aws_instance.tainted` | `unresolved (alias: default)` (unresolved) |
| replace | `aws_ebs_volume.requested` | `unresolved (alias: default)` (unresolved) |
| replace | `aws_lb.triggered` | `unresolved (alias: default)` (unresolved) |
| replace | `aws_eip.mystery` | `unresolved (alias: default)` (unresolved) |
| replace | `aws_instance.create_before_destroy` | `unresolved (alias: default)` (unresolved) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 6 resource(s)</summary>

- replace `aws_db_instance.main`
- replace `aws_ebs_volume.requested`
- replace `aws_eip.mystery`
- replace `aws_instance.create_before_destroy`
- replace `aws_instance.tainted`
- replace `aws_lb.triggered`

</details>

<details><summary><b>destroy wave 1</b> &mdash; 6 resource(s)</summary>

- delete `aws_iam_user.keyed` (no configuration: order unknown)
- delete `aws_instance.counted` (no configuration: order unknown)
- delete `aws_lambda_function.orphan` (no configuration: order unknown)
- delete `aws_sns_topic.switched` (no configuration: order unknown)
- delete `aws_sqs_queue.removed` (no configuration: order unknown)
- delete `module.gone.aws_s3_bucket.data` (no configuration: order unknown)

</details>


> replace_triggered_by is not part of the plan JSON: a resource replaced by a trigger is not ordered after the resource that triggered it.

> Resources removed from the configuration have no dependency information in the plan; their destroy order is unknown and they are shown in wave 1.
