<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 0 |
| update | 1 |
| replace | 0 |
| delete | 0 |
| Drift | 0 |
| Known after apply | 0 attribute(s) |

> **Notice:** Apply account unknown: set tfapply_account_id in tfplanlens.config.yaml (it cannot be read from the plan). Cross-account flags are off.

> **Notice:** Hid secret-looking values in 10 place(s): they are not marked sensitive by Terraform, but have secret-like names, key material, tokens or NAME=value lines. Set redact_secrets: false in tfplanlens.config.yaml to show them.

### Danger

No deletes or replacements.

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; update 1</summary>

- `aws_launch_template.app` (update)
  - `db_password`: (sensitive value changed)
  - `description`: `"Rotates the token every 30 days"` → `"Rotates the token every 60 days"`
  - `endpoint`: `"postgres://deploy:[REDACTED]@db.example.com/x"` → `"postgres://deploy:[REDACTED]@db.example.com/x"`
  - `environment.variables.API_TOKEN`: (sensitive value changed)
  - `environment.variables.LOG_LEVEL`: `"info"` → `"debug"`
  - `marked`: (sensitive value changed)
  - `secret_arn`: `"arn:aws:secretsmanager:us-east-1:111111111111:secret:app-AbCdEf"` → `"arn:aws:secretsmanager:us-east-1:111111111111:secret:app-ZzYyXx"`
  - `token_validity_days`: `30` → `60`
  - `user_data: 2 lines changed of 2 (+1 -1)`
    ```diff
    @@ -1,2 +1,2 @@
     [REDACTED] (private key)
    -echo start v1
    +echo start v2
    ```
  - `user_data_base64: 2 lines changed of 5 (+1 -1)`
    ```diff
    @@ -1,4 +1,4 @@
     #!/bin/sh
    -echo deploying v1
    +echo deploying v2
     export DB_PASSWORD=[REDACTED]
     export AWS_KEY=[REDACTED]
    ```

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 1 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| update | `aws_launch_template.app` | `unresolved (alias: default)` (unresolved) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 1 resource(s)</summary>

- update `aws_launch_template.app`

</details>

