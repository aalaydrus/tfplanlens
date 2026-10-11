<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 1 |
| update | 5 |
| replace | 1 |
| delete | 0 |
| Drift | 0 |
| Known after apply | 19 attribute(s) |
| Warnings | 2 |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `module.net.aws_sqs_queue.inner` | `Attribute change forces replacement: name` | `111111111111` (source: role_arn) |

### Changes by account

<details><summary><b>aws</b> &mdash; target account 999999999999 (source: caller_identity) &mdash; <b>differs from apply account</b> &mdash; create 1, update 1</summary>

- `aws_sns_topic_subscription.cross` (create, 6 known after apply)
  - **WARNING:** `topic_arn is in account 111111111111 but provider aws targets 999999999999 (caller_identity)`
- `aws_sqs_queue.runner` (update)
  - `tags.env`: `(absent)` → `"v2"`
  - `tags_all.env`: `(absent)` → `"v2"`

</details>

<details><summary><b>aws.audit</b> &mdash; target account 999999999999 (source: allowed_account_ids) &mdash; <b>differs from apply account</b> &mdash; update 1</summary>

- `aws_sqs_queue.audit` (update)
  - `visibility_timeout_seconds`: `30` → `60`

</details>

<details><summary><b>aws.prod</b> &mdash; target account 111111111111 (source: role_arn) &mdash; update 1, replace 1</summary>

- `aws_sns_topic.prod` (update)
  - `display_name`: `""` → `"prod v2"`
- `module.net.aws_sqs_queue.inner` (replace, 12 known after apply)
  - `kms_master_key_id`: `""` → `null`
  - `name`: `"net-inner-v1"` → `"net-inner-v2"` (forces replacement)
  - `tags`: `{}` → `null`

</details>

<details><summary><b>aws.sandbox</b> &mdash; target account 111111111111 (source: apply_account) &mdash; update 1</summary>

- `aws_ssm_parameter.sandbox` (update, 1 known after apply)
  - **WARNING:** `arn is in account 999999999999 but provider aws.sandbox targets 111111111111 (apply_account)`
  - `value`: (sensitive value changed)

</details>

<details><summary><b>aws.staging</b> &mdash; target account 222222222222 (source: role_arn_variable) &mdash; <b>differs from apply account</b> &mdash; update 1</summary>

- `aws_sqs_queue.staging` (update)
  - `tags.env`: `(absent)` → `"v2"`
  - `tags_all.env`: `(absent)` → `"v2"`

</details>

### Changes by module

<details><summary><b>module.net</b> &mdash; 1 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| replace, 12 known after apply | `module.net.aws_sqs_queue.inner` | `111111111111` (role_arn) |

</details>

<details><summary><b>root</b> &mdash; 6 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| update | `aws_sns_topic.prod` | `111111111111` (role_arn) |
| create, 6 known after apply | `aws_sns_topic_subscription.cross` | `999999999999` (caller_identity) |
| update | `aws_sqs_queue.audit` | `999999999999` (allowed_account_ids) |
| update | `aws_sqs_queue.runner` | `999999999999` (caller_identity) |
| update | `aws_sqs_queue.staging` | `222222222222` (role_arn_variable) |
| update, 1 known after apply | `aws_ssm_parameter.sandbox` | `111111111111` (apply_account) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 6 resource(s)</summary>

- update `aws_sns_topic.prod`
- update `aws_sqs_queue.audit`
- update `aws_sqs_queue.runner`
- update `aws_sqs_queue.staging`
- update `aws_ssm_parameter.sandbox`
- replace `module.net.aws_sqs_queue.inner`

</details>

<details><summary><b>apply wave 2</b> &mdash; 1 resource(s)</summary>

- create `aws_sns_topic_subscription.cross`

</details>

