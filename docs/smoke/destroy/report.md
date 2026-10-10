<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | destroy, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 0 |
| update | 0 |
| replace | 0 |
| delete | 4 |
| Drift | 0 |
| Known after apply | 0 attribute(s) |

> **Notice:** Destroy plan: applying destroys every resource listed under Danger.

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **DELETE** | `terraform_data.db` | `Destroy plan: every resource is being destroyed` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.shard[0]` | `Destroy plan: every resource is being destroyed` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.shard[1]` | `Destroy plan: every resource is being destroyed` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.web` | `Destroy plan: every resource is being destroyed` | `111111111111` (source: apply_account) |

### Changes by account

<details><summary><b>terraform</b> &mdash; target account 111111111111 (source: apply_account) &mdash; delete 4</summary>

- `terraform_data.db` (delete)
- `terraform_data.shard[0]` (delete)
- `terraform_data.shard[1]` (delete)
- `terraform_data.web` (delete)

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 4 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| delete | `terraform_data.db` | `111111111111` (apply_account) |
| delete | `terraform_data.shard[0]` | `111111111111` (apply_account) |
| delete | `terraform_data.shard[1]` | `111111111111` (apply_account) |
| delete | `terraform_data.web` | `111111111111` (apply_account) |

</details>

