<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 1 |
| update | 4 |
| replace | 1 |
| delete | 3 |
| no-op | 6 |
| forget | 1 |
| Drift | 0 |
| Known after apply | 10 attribute(s) |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `terraform_data.db` | `Attribute change forces replacement: triggers_replace`<br>`used by 1+ resources in this plan: terraform_data.app` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.subnet["b"]` | `for_each key removed: b`<br>**WARNING:** `for_each key removed while sibling instances change (possible key rename: terraform_data.subnet["d"])`<br>`used by 1+ resources in this plan: terraform_data.lb` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.worker[2]` | `count shrank; index 2 no longer exists` | `111111111111` (source: apply_account) |
| **DELETE** | `terraform_data.worker[3]` | `count shrank; index 3 no longer exists` | `111111111111` (source: apply_account) |

### Changes by account

<details><summary><b>terraform</b> &mdash; target account 111111111111 (source: apply_account) &mdash; create 1, update 4, replace 1, delete 3, forget 1, no-op 6</summary>

- `terraform_data.app` (update, 2 known after apply)
- `terraform_data.creds` (update, 1 known after apply)
  - `input`: (sensitive value changed)
- `terraform_data.db` (replace, 2 known after apply)
  - `triggers_replace`: `"a"` → `"b"` (forces replacement)
- `terraform_data.lb` (update, 2 known after apply)
  - `input[1]`: `"c8705e75-c0b1-9ae7-83e7-f552d5ca0cf6"` → `"58541517-227a-6412-d3bf-281d7dc7f79b"`
- `terraform_data.legacy` (removed from state only)
- `terraform_data.subnet["b"]` (delete)
- `terraform_data.subnet["d"]` (create, 2 known after apply)
- `terraform_data.user_data` (update, 1 known after apply)
  - `input: 4 lines changed of 401 (+2 -2)`
    ```diff
    @@ -120,6 +120,6 @@
     echo step-118
     echo step-119
    -echo step-120
    -echo step-121
    +echo CHANGED-120
    +echo CHANGED-121
     echo step-122
     echo step-123
    ```
- `terraform_data.worker[2]` (delete)
- `terraform_data.worker[3]` (delete)
- `module.network_core.terraform_data.route_table` (moved from module.network.terraform_data.route_table)
- `module.network_core.terraform_data.vpc` (moved from module.network.terraform_data.vpc)

</details>

### Changes by module

<details><summary><b>module.network_core</b> &mdash; 2 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| moved from module.network.terraform_data.route_table | `module.network_core.terraform_data.route_table` | `111111111111` (apply_account) |
| moved from module.network.terraform_data.vpc | `module.network_core.terraform_data.vpc` | `111111111111` (apply_account) |

</details>

<details><summary><b>root</b> &mdash; 10 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| update, 2 known after apply | `terraform_data.app` | `111111111111` (apply_account) |
| update, 1 known after apply | `terraform_data.creds` | `111111111111` (apply_account) |
| replace, 2 known after apply | `terraform_data.db` | `111111111111` (apply_account) |
| update, 2 known after apply | `terraform_data.lb` | `111111111111` (apply_account) |
| removed from state only | `terraform_data.legacy` | `111111111111` (apply_account) |
| delete | `terraform_data.subnet["b"]` | `111111111111` (apply_account) |
| create, 2 known after apply | `terraform_data.subnet["d"]` | `111111111111` (apply_account) |
| update, 1 known after apply | `terraform_data.user_data` | `111111111111` (apply_account) |
| delete | `terraform_data.worker[2]` | `111111111111` (apply_account) |
| delete | `terraform_data.worker[3]` | `111111111111` (apply_account) |

</details>

### Waves

Terraform applies in parallel: this shows possible parallelism, not a strict order.

<details><summary><b>apply wave 1</b> &mdash; 4 resource(s)</summary>

- update `terraform_data.creds`
- replace `terraform_data.db`
- delete `terraform_data.subnet` (2 instances)
- update `terraform_data.user_data`

</details>

<details><summary><b>apply wave 2</b> &mdash; 2 resource(s)</summary>

- update `terraform_data.app`
- update `terraform_data.lb`

</details>

<details><summary><b>destroy wave 1</b> &mdash; 1 resource(s)</summary>

- delete `terraform_data.worker` (2 instances)

</details>


> 1 reference(s) to local values are not visible in the plan JSON, so a dependency reached only through a local is missing and a resource can show in an earlier wave than it will run.
### Resources with multiple instances

<details><summary>terraform_data.subnet (for_each) &mdash; 4 instance(s)</summary>

- no-op `a`
- delete `b`
- no-op `c`
- create `d`

</details>

<details><summary>terraform_data.worker (count) &mdash; 4 instance(s)</summary>

- no-op `0`
- no-op `1`
- delete `2`
- delete `3`

</details>

