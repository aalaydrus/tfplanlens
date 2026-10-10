<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 0 |
| update | 0 |
| replace | 3 |
| delete | 0 |
| Drift | 0 |
| Known after apply | 21 attribute(s) |

### Danger

| Action | Resource | Why | Target account |
|---|---|---|---|
| **REPLACE** | `local_file.alt` | `Attribute change forces replacement: content` | `111111111111` (source: apply_account) |
| **REPLACE** | `local_file.root` | `Attribute change forces replacement: content` | `111111111111` (source: apply_account) |
| **REPLACE** | `module.net.local_file.inner` | `Attribute change forces replacement: content` | `111111111111` (source: apply_account) |

### Changes by account

<details><summary><b>local</b> &mdash; target account 111111111111 (source: apply_account) &mdash; replace 1</summary>

- `local_file.root` (replace, 7 known after apply)
  - `content`: `"root-v1"` → `"root-v2"` (forces replacement)

</details>

<details><summary><b>local.alt</b> &mdash; target account 111111111111 (source: apply_account) &mdash; replace 2</summary>

- `local_file.alt` (replace, 7 known after apply)
  - `content`: `"alt-v1"` → `"alt-v2"` (forces replacement)
- `module.net.local_file.inner` (replace, 7 known after apply)
  - `content`: `"inner-v1"` → `"inner-v2"` (forces replacement)

</details>

### Changes by module

<details><summary><b>module.net</b> &mdash; 1 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| replace, 7 known after apply | `module.net.local_file.inner` | `111111111111` (apply_account) |

</details>

<details><summary><b>root</b> &mdash; 2 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| replace, 7 known after apply | `local_file.alt` | `111111111111` (apply_account) |
| replace, 7 known after apply | `local_file.root` | `111111111111` (apply_account) |

</details>

