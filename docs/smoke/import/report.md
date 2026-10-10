<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 0 |
| update | 2 |
| replace | 0 |
| delete | 0 |
| Drift | 0 |
| Known after apply | 2 attribute(s) |

### Danger

No deletes or replacements.

### Changes by account

<details><summary><b>terraform</b> &mdash; target account 111111111111 (source: apply_account) &mdash; update 2</summary>

- `terraform_data.imp` (import + update, 1 known after apply)
  - `input`: `null` → `"managed"`
- `terraform_data.other` (update, 1 known after apply)
  - `input`: `"other"` → `"other-changed"`

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 2 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| import + update, 1 known after apply | `terraform_data.imp` | `111111111111` (apply_account) |
| update, 1 known after apply | `terraform_data.other` | `111111111111` (apply_account) |

</details>

