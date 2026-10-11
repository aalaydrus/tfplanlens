<!-- tfplanlens-diff -->
## tfplanlens plan diff

| | |
|---|---|
| Old plan | create 1, update 1, replace 1, delete 0 |
| New plan | create 1, update 1, replace 0, delete 1 |
| Unchanged | 0 resource(s) |

**New danger**: the new plan deletes or replaces what the old one did not:

- `terraform_data.legacy`

No longer deleted or replaced:

- `terraform_data.db`

<details><summary><b>Only in the new plan</b> &mdash; 2</summary>

- **+ create** `terraform_data.cache[2]`
- **+ delete** `terraform_data.legacy`

</details>

<details><summary><b>No longer in the plan</b> &mdash; 2</summary>

- **- was replace** `terraform_data.db`
- **- was create** `terraform_data.queue`

</details>

<details><summary><b>Changed</b> &mdash; 1</summary>

- **~ update -> update** `terraform_data.app`
  - `attribute input: was "v1" => "v2", now "v1" => "v3"`

</details>

