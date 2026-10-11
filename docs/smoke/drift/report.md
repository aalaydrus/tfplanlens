<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 2 |
| update | 1 |
| replace | 0 |
| delete | 0 |
| Drift | 2 |
| Known after apply | 15 attribute(s) |

### Danger

No deletes or replacements.

<details><summary><b>Drift</b>: 2 changed outside Terraform (0 high risk)</summary>

- `local_file.config`: Deleted outside Terraform
- `local_file.keep`: Deleted outside Terraform

</details>

### Changes by account

<details><summary><b>local</b> &mdash; target account 111111111111 (source: apply_account) &mdash; create 2</summary>

- `local_file.config` (create, 7 known after apply)
- `local_file.keep` (create, 7 known after apply)

</details>

<details><summary><b>terraform</b> &mdash; target account 111111111111 (source: apply_account) &mdash; update 1</summary>

- `terraform_data.app` (update, 1 known after apply)
  - `input`: `"v1"` → `"v2"`

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 3 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| create, 7 known after apply | `local_file.config` | `111111111111` (apply_account) |
| create, 7 known after apply | `local_file.keep` | `111111111111` (apply_account) |
| update, 1 known after apply | `terraform_data.app` | `111111111111` (apply_account) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 2 resource(s)</summary>

- create `local_file.config`
- create `local_file.keep`

</details>

<details><summary><b>apply wave 2</b> &mdash; 1 resource(s)</summary>

- update `terraform_data.app`

</details>

