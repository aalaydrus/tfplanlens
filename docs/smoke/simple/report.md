<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 1 |
| update | 1 |
| replace | 1 |
| delete | 1 |
| Drift | 0 |
| Known after apply | 5 attribute(s) |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `terraform_data.db` | `Attribute change forces replacement: triggers_replace` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.old` | `Removed from config` | `111111111111` (source: apply_account) |

### Changes by account

<details><summary><b>terraform</b> &mdash; target account 111111111111 (source: apply_account) &mdash; create 1, update 1, replace 1, delete 1</summary>

- `terraform_data.cache` (create, 2 known after apply)
- `terraform_data.db` (replace, 2 known after apply)
  - `triggers_replace`: `"a"` → `"b"` (forces replacement)
- `terraform_data.old` (delete)
- `terraform_data.web` (update, 1 known after apply)
  - `input`: `"v1"` → `"v2"`

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 4 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| create, 2 known after apply | `terraform_data.cache` | `111111111111` (apply_account) |
| replace, 2 known after apply | `terraform_data.db` | `111111111111` (apply_account) |
| delete | `terraform_data.old` | `111111111111` (apply_account) |
| update, 1 known after apply | `terraform_data.web` | `111111111111` (apply_account) |

</details>

