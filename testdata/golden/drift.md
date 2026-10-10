<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 1 |
| update | 1 |
| replace | 1 |
| delete | 0 |
| Drift | 4 |
| Known after apply | 0 attribute(s) |

> **Notice:** Apply account unknown: set tfapply_account_id in tfplanlens.config.yaml (it cannot be read from the plan). Cross-account flags are off.

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `aws_security_group.web` | `Attribute change forces replacement: description`<br>**HIGH RISK:** also drifted outside Terraform | `unresolved (alias: default)` (source: unresolved) |

<details open><summary><b>Drift</b>: 4 changed outside Terraform (1 high risk)</summary>

- `aws_security_group.web` **HIGH RISK**: Also being replaced or deleted by this plan.
  - `ingress[0].cidr_blocks[0]`: `"10.0.0.0/8"` → `"0.0.0.0/0"`
- `aws_instance.api`
  - `instance_type`: `"t3.micro"` → `"t3.large"`
- `aws_s3_bucket.logs`
  - `tags.owner`: `(absent)` → `"bob"`
- `aws_instance.legacy`: Deleted outside Terraform

</details>

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; create 1, update 1, replace 1</summary>

- `aws_security_group.web` (replace)
  - `description`: `"web"` → `"web v2"` (forces replacement)
- `aws_instance.api` (update)
  - `instance_type`: `"t3.large"` → `"t3.small"`
- `aws_instance.legacy` (create)

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 3 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| replace | `aws_security_group.web` | `unresolved (alias: default)` (unresolved) |
| update | `aws_instance.api` | `unresolved (alias: default)` (unresolved) |
| create | `aws_instance.legacy` | `unresolved (alias: default)` (unresolved) |

</details>

