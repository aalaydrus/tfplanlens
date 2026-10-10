<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | refresh-only, applyable: yes, errored: no |
| Executed from (apply account) | `111111111111` (source: flag) |
| create | 0 |
| update | 0 |
| replace | 0 |
| delete | 0 |
| Drift | 2 |
| Known after apply | 0 attribute(s) |

> **Notice:** Refresh-only plan: applying only updates state to match what exists; it creates, changes and destroys nothing.

### Danger

No deletes or replacements.

<details><summary><b>Drift</b>: 2 changed outside Terraform (0 high risk)</summary>

- `local_file.config`: Deleted outside Terraform
- `local_file.keep`: Deleted outside Terraform

</details>

