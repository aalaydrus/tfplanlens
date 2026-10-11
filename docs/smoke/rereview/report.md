<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 1 |
| update | 1 |
| replace | 1 |
| delete | 0 |
| no-op | 3 |
| Drift | 0 |
| Known after apply | 5 attribute(s) |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `terraform_data.db` | `Attribute change forces replacement: triggers_replace` | `111111111111` (source: apply_account) |

### Changes by account

<details><summary><b>terraform</b> &mdash; target account 111111111111 (source: apply_account) &mdash; create 1, update 1, replace 1, no-op 3</summary>

- `terraform_data.app` (update, 1 known after apply)
  - `input`: `"v1"` → `"v2"`
- `terraform_data.db` (replace, 2 known after apply)
  - `triggers_replace`: `"a"` → `"b"` (forces replacement)
- `terraform_data.queue` (create, 2 known after apply)

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 3 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| update, 1 known after apply | `terraform_data.app` | `111111111111` (apply_account) |
| replace, 2 known after apply | `terraform_data.db` | `111111111111` (apply_account) |
| create, 2 known after apply | `terraform_data.queue` | `111111111111` (apply_account) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 3 resource(s)</summary>

- update `terraform_data.app`
- replace `terraform_data.db`
- create `terraform_data.queue`

</details>

### Resources with multiple instances

<details><summary>terraform_data.cache (count) &mdash; 2 instance(s)</summary>

- no-op `0`
- no-op `1`

</details>

