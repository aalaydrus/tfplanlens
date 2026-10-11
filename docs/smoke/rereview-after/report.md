<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 1 |
| update | 1 |
| replace | 0 |
| delete | 1 |
| no-op | 3 |
| Drift | 0 |
| Known after apply | 3 attribute(s) |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **DELETE** | `terraform_data.legacy` | `Removed from config` | `111111111111` (source: apply_account) |

### Changes by account

<details><summary><b>terraform</b> &mdash; target account 111111111111 (source: apply_account) &mdash; create 1, update 1, delete 1, no-op 3</summary>

- `terraform_data.app` (update, 1 known after apply)
  - `input`: `"v1"` → `"v3"`
- `terraform_data.cache[2]` (create, 2 known after apply)
- `terraform_data.legacy` (delete)

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 3 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| update, 1 known after apply | `terraform_data.app` | `111111111111` (apply_account) |
| create, 2 known after apply | `terraform_data.cache[2]` | `111111111111` (apply_account) |
| delete | `terraform_data.legacy` | `111111111111` (apply_account) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 2 resource(s)</summary>

- update `terraform_data.app`
- create `terraform_data.cache`

</details>

<details><summary><b>destroy wave 1</b> &mdash; 1 resource(s)</summary>

- delete `terraform_data.legacy` (no configuration: order unknown)

</details>


> Resources removed from the configuration have no dependency information in the plan; their destroy order is unknown and they are shown in wave 1.
### Resources with multiple instances

<details><summary>terraform_data.cache (count) &mdash; 3 instance(s)</summary>

- no-op `0`
- no-op `1`
- create `2`

</details>

