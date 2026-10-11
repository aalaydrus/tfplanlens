<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `999999999999` (source: config) |
| create | 3 |
| update | 4 |
| replace | 2 |
| delete | 1 |
| no-op | 7 |
| Drift | 2 |
| Known after apply | 40 attribute(s) |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `aws_dynamodb_table.orders` **STATEFUL** | `Attribute change forces replacement: range_key, hash_key` | `123456789012` (source: allowed_account_ids) |
| **REPLACE** | `aws_secretsmanager_secret_version.db` | `Attribute change forces replacement: secret_string` | `123456789012` (source: allowed_account_ids) |
| **DELETE** | `aws_sqs_queue.orders["bulk"]` | `for_each key removed: bulk`<br>**WARNING:** `for_each key removed while sibling instances change (possible key rename: aws_sqs_queue.orders["audit"], aws_sqs_queue.orders["standard"])` | `123456789012` (source: allowed_account_ids) |

<details><summary><b>Drift</b>: 2 changed outside Terraform (0 high risk)</summary>

- `aws_iam_role.worker`
  - `inline_policy[0].name`: `(absent)` → `"orders-worker"`
  - `inline_policy[0].policy: 32 lines changed of 32 (+32 -0)`
    ```diff
    @@ -1,0 +1,32 @@
    +{
    +  "Statement": [
    +    {
    +      "Action": [
    +        "sqs:ReceiveMessage",
    +        "sqs:DeleteMessage"
    +      ],
    +      "Effect": "Allow",
    +      "Resource": [
    +        "arn:aws:sqs:us-east-1:123456789012:orders-bulk",
    +        "arn:aws:sqs:us-east-1:123456789012:orders-priority",
    +        "arn:aws:sqs:us-east-1:123456789012:orders-standard"
    +      ]
    +    },
    +    {
    +      "Action": [
    +        "secretsmanager:GetSecretValue"
    +      ],
    +      "Effect": "Allow",
    +      "Resource": "arn:aws:secretsmanager:us-east-1:123456789012:secret:orders/db-credentials-1LFT0S"
    +    },
    +    {
    +      "Action": [
    +        "dynamodb:PutItem",
    +        "dynamodb:GetItem"
    +      ],
    +      "Effect": "Allow",
    +      "Resource": "arn:aws:dynamodb:us-east-1:123456789012:table/orders"
    +    }
    +  ],
    +  "Version": "2012-10-17"
    +}
    statements:
    +Allow sqs:DeleteMessage,sqs:ReceiveMessage -> arn:aws:sqs:us-east-1:123456789012:orders-bulk,arn:aws:sqs:us-east-1:123456789012:orders-priority,arn:aws:sqs:us-east-1:123456789012:orders-standard
    +Allow secretsmanager:GetSecretValue -> arn:aws:secretsmanager:us-east-1:123456789012:secret:orders/db-credentials-1LFT0S
    +Allow dynamodb:GetItem,dynamodb:PutItem -> arn:aws:dynamodb:us-east-1:123456789012:table/orders
    ```
- `aws_s3_bucket.receipts`
  - `versioning[0].enabled`: `false` → `true`

</details>

### Changes by account

<details><summary><b>aws</b> &mdash; target account 123456789012 (source: allowed_account_ids) &mdash; <b>differs from apply account</b> &mdash; create 3, update 4, replace 2, delete 1, no-op 7</summary>

- `aws_dynamodb_table.orders` **STATEFUL** (replace, 13 known after apply)
  - `attribute[0].name`: `"order_id"` → `"customer_id"`
  - `attribute[1].name`: `(absent)` → `"order_id"`
  - `attribute[1].type`: `(absent)` → `"S"`
  - `deletion_protection_enabled`: `false` → `null`
  - `hash_key`: `"order_id"` → `"customer_id"` (forces replacement)
  - `range_key`: `null` → `"order_id"` (forces replacement)
  - `stream_enabled`: `false` → `null`
  - `table_class`: `"STANDARD"` → `null`
  - `tags`: `{}` → `null`
- `aws_iam_role_policy.worker` (update, 1 known after apply)
- `aws_s3_bucket_lifecycle_configuration.receipts` (create, 2 known after apply)
- `aws_secretsmanager_secret.db` (update)
  - `description`: `"Database credentials for the orders service"` → `"Database credentials for the orders service (rotated quarterly)"`
- `aws_secretsmanager_secret_version.db` (replace, 6 known after apply)
  - `secret_binary`: (sensitive value changed)
  - `secret_string`: (sensitive value changed) (forces replacement)
- `aws_sns_topic_subscription.audit` (create, 7 known after apply)
- `aws_sqs_queue.orders["audit"]` (create, 10 known after apply)
- `aws_sqs_queue.orders["bulk"]` (delete)
- `aws_sqs_queue.orders["standard"]` (update)
  - `visibility_timeout_seconds`: `30` → `60`
- `aws_ssm_parameter.flags` (update, 1 known after apply)
  - `value`: (sensitive value changed)

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 10 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| replace, 13 known after apply | `aws_dynamodb_table.orders` | `123456789012` (allowed_account_ids) |
| update, 1 known after apply | `aws_iam_role_policy.worker` | `123456789012` (allowed_account_ids) |
| create, 2 known after apply | `aws_s3_bucket_lifecycle_configuration.receipts` | `123456789012` (allowed_account_ids) |
| update | `aws_secretsmanager_secret.db` | `123456789012` (allowed_account_ids) |
| replace, 6 known after apply | `aws_secretsmanager_secret_version.db` | `123456789012` (allowed_account_ids) |
| create, 7 known after apply | `aws_sns_topic_subscription.audit` | `123456789012` (allowed_account_ids) |
| create, 10 known after apply | `aws_sqs_queue.orders["audit"]` | `123456789012` (allowed_account_ids) |
| delete | `aws_sqs_queue.orders["bulk"]` | `123456789012` (allowed_account_ids) |
| update | `aws_sqs_queue.orders["standard"]` | `123456789012` (allowed_account_ids) |
| update, 1 known after apply | `aws_ssm_parameter.flags` | `123456789012` (allowed_account_ids) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 5 resource(s)</summary>

- replace `aws_dynamodb_table.orders`
- create `aws_s3_bucket_lifecycle_configuration.receipts`
- update `aws_secretsmanager_secret.db`
- delete `aws_sqs_queue.orders` (3 instances)
- update `aws_ssm_parameter.flags`

</details>

<details><summary><b>apply wave 2</b> &mdash; 3 resource(s)</summary>

- update `aws_iam_role_policy.worker`
- replace `aws_secretsmanager_secret_version.db`
- create `aws_sns_topic_subscription.audit`

</details>


> 1 reference(s) to local values are not visible in the plan JSON, so a dependency reached only through a local is missing and a resource can show in an earlier wave than it will run.
### Resources with multiple instances

<details><summary>aws_sqs_queue.orders (for_each) &mdash; 4 instance(s)</summary>

- create `audit`
- delete `bulk`
- no-op `priority`
- update `standard`

</details>

