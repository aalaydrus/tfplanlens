<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `999999999999` (source: flag) |
| create | 2 |
| update | 3 |
| replace | 1 |
| delete | 1 |
| Drift | 0 |
| Known after apply | 4 attribute(s) |
| Warnings | 1 |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **DELETE** | `aws_db_instance.prod` **STATEFUL** | `Removed from config` | `111111111111` (source: role_arn) |
| **REPLACE** | `aws_instance.web` | `Attribute change forces replacement: ami` | `222222222222` (source: role_arn_variable) |

### Changes by account

<details><summary><b>aws</b> &mdash; target account 999999999999 (source: caller_identity) &mdash; create 1</summary>

- `aws_s3_bucket.artifacts` **STATEFUL** (create, 2 known after apply)

</details>

<details><summary><b>aws.prod</b> &mdash; target account 111111111111 (source: role_arn) &mdash; <b>differs from apply account</b> &mdash; update 1, delete 1</summary>

- `aws_iam_role.deployer` (update)
  - `max_session_duration`: `3600` → `7200`
- `aws_db_instance.prod` **STATEFUL** (delete)

</details>

<details><summary><b>aws.sandbox</b> &mdash; target account 999999999999 (source: apply_account) &mdash; create 1, update 1</summary>

- `aws_sqs_queue.jobs` (create, 2 known after apply)
- `aws_sns_topic.alerts` (update)
  - `tags.team`: `(absent)` → `"sre"`

</details>

<details><summary><b>aws.staging</b> &mdash; target account 222222222222 (source: role_arn_variable) &mdash; <b>differs from apply account</b> &mdash; update 1, replace 1</summary>

- `aws_instance.web` (replace)
  - `ami`: `"ami-old"` → `"ami-new"` (forces replacement)
- `aws_ssm_parameter.config` (update)
  - **WARNING:** `kms_key_arn is in account 111111111111 but provider aws.staging targets 222222222222 (role_arn_variable)`
  - `description`: `"old"` → `"new"`

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 7 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| create, 2 known after apply | `aws_s3_bucket.artifacts` | `999999999999` (caller_identity) |
| update | `aws_iam_role.deployer` | `111111111111` (role_arn) |
| delete | `aws_db_instance.prod` | `111111111111` (role_arn) |
| replace | `aws_instance.web` | `222222222222` (role_arn_variable) |
| update | `aws_ssm_parameter.config` | `222222222222` (role_arn_variable) |
| create, 2 known after apply | `aws_sqs_queue.jobs` | `999999999999` (apply_account) |
| update | `aws_sns_topic.alerts` | `999999999999` (apply_account) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 6 resource(s)</summary>

- update `aws_iam_role.deployer`
- replace `aws_instance.web`
- create `aws_s3_bucket.artifacts`
- update `aws_sns_topic.alerts`
- create `aws_sqs_queue.jobs`
- update `aws_ssm_parameter.config`

</details>

<details><summary><b>destroy wave 1</b> &mdash; 1 resource(s)</summary>

- delete `aws_db_instance.prod`

</details>

