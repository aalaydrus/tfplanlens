<!-- tfplanlens-diff -->
## tfplanlens plan diff

| | |
|---|---|
| Old plan | create 3, update 3, replace 1, delete 1 |
| New plan | create 3, update 4, replace 0, delete 0 |
| Unchanged | 6 resource(s) |

No longer deleted or replaced:

- `aws_dynamodb_table.orders`
- `aws_sqs_queue.orders["bulk"]`

<details><summary><b>No longer in the plan</b> &mdash; 1</summary>

- **- was delete** `aws_sqs_queue.orders["bulk"]`

</details>

<details><summary><b>Changed</b> &mdash; 1</summary>

- **~ replace -> update** `aws_dynamodb_table.orders`
  - `action: replace -> update`
  - `reason: replace_because_cannot_update -> (none)`
  - `attribute no longer changes: deletion_protection_enabled`
  - `attribute now changes: global_secondary_index[0].hash_key`
  - `attribute now changes: global_secondary_index[0].name`
  - `attribute now changes: global_secondary_index[0].non_key_attributes`
  - `attribute now changes: global_secondary_index[0].on_demand_throughput`
  - `attribute now changes: global_secondary_index[0].projection_type`
  - `attribute now changes: global_secondary_index[0].range_key`
  - `attribute no longer changes: hash_key`
  - `attribute no longer changes: range_key`
  - `attribute no longer changes: stream_enabled`
  - `attribute no longer changes: table_class`
  - `attribute no longer changes: tags`

</details>

