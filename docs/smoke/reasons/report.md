<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 1 |
| update | 1 |
| replace | 3 |
| delete | 3 |
| no-op | 2 |
| Drift | 0 |
| Known after apply | 9 attribute(s) |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `terraform_data.follower` | `replace_triggered_by dependency changed` | `111111111111` (source: apply_account) |
| **REPLACE** | `terraform_data.requested` | `Replacement requested with -replace` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.single` | `Resource switched between count/for_each/single`<br>**WARNING:** `count/for_each was added or removed on this resource; instances are being destroyed rather than moved` | `111111111111` (source: apply_account) |
| **REPLACE** | `terraform_data.tainted` | `Resource is tainted` | `111111111111` (source: apply_account) |
| **DELETE** | `module.gone.terraform_data.inner` | `Removed from config` | `111111111111` (source: apply_account) |
| **DELETE** | `module.mods["b"].terraform_data.inner` | `Whole module removed or its call changed` | `111111111111` (source: apply_account) |

### Changes by account

<details><summary><b>terraform</b> &mdash; target account 111111111111 (source: apply_account) &mdash; create 1, update 1, replace 3, delete 3, no-op 2</summary>

- `terraform_data.counted[0]` (moved from terraform_data.counted)
- `terraform_data.follower` (replace, 2 known after apply)
- `terraform_data.requested` (replace, 2 known after apply)
- `terraform_data.single` (delete)
- `terraform_data.single["a"]` (create, 2 known after apply)
- `terraform_data.tainted` (replace, 2 known after apply)
- `terraform_data.trigger` (update, 1 known after apply)
  - `input`: `"t1"` → `"t2"`
- `module.gone.terraform_data.inner` (delete)
- `module.mods["b"].terraform_data.inner` (delete)

</details>

### Changes by module

<details><summary><b>module.gone</b> &mdash; 1 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| delete | `module.gone.terraform_data.inner` | `111111111111` (apply_account) |

</details>

<details><summary><b>module.mods[&quot;b&quot;]</b> &mdash; 1 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| delete | `module.mods["b"].terraform_data.inner` | `111111111111` (apply_account) |

</details>

<details><summary><b>root</b> &mdash; 7 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| moved from terraform_data.counted | `terraform_data.counted[0]` | `111111111111` (apply_account) |
| replace, 2 known after apply | `terraform_data.follower` | `111111111111` (apply_account) |
| replace, 2 known after apply | `terraform_data.requested` | `111111111111` (apply_account) |
| delete | `terraform_data.single` | `111111111111` (apply_account) |
| create, 2 known after apply | `terraform_data.single["a"]` | `111111111111` (apply_account) |
| replace, 2 known after apply | `terraform_data.tainted` | `111111111111` (apply_account) |
| update, 1 known after apply | `terraform_data.trigger` | `111111111111` (apply_account) |

</details>

