<!-- tfplanlens-report -->
## tfplanlens plan report

| | |
|---|---|
| Plan | normal, applyable: yes, errored: no |
| Executed from (apply account) | **unresolved** |
| create | 0 |
| update | 3 |
| replace | 0 |
| delete | 0 |
| Drift | 0 |
| Known after apply | 0 attribute(s) |

> **Notice:** Apply account unresolved (set --tfapply-account-id): cross-account flags are off.

### Danger

No deletes or replacements.

### Changes by account

<details><summary><b>aws</b> &mdash; target account unresolved (alias: default) (source: unresolved) &mdash; update 3</summary>

- `aws_instance.app` (update)
  - `user_data_base64: 4 lines changed of 412 (+2 -2)`
    ```diff
    @@ -99,5 +99,5 @@
     echo 'step 97: configuring subsystem 97' >> /var/log/boot.log
     echo 'step 98: configuring subsystem 98' >> /var/log/boot.log
    -echo 'step 99: configuring subsystem 99' >> /var/log/boot.log
    +echo 'step 99: configuring subsystem 99 (patched)' >> /var/log/boot.log
     echo 'step 100: configuring subsystem 100' >> /var/log/boot.log
     echo 'step 101: configuring subsystem 101' >> /var/log/boot.log
    @@ -249,4 +249,5 @@
     echo 'step 247: configuring subsystem 247' >> /var/log/boot.log
     echo 'step 248: configuring subsystem 248' >> /var/log/boot.log
    +systemctl enable --now node_exporter
     echo 'step 249: configuring subsystem 249' >> /var/log/boot.log
     echo 'step 250: configuring subsystem 250' >> /var/log/boot.log
    @@ -298,5 +299,4 @@
     echo 'step 296: configuring subsystem 296' >> /var/log/boot.log
     echo 'step 297: configuring subsystem 297' >> /var/log/boot.log
    -echo 'step 298: configuring subsystem 298' >> /var/log/boot.log
     echo 'step 299: configuring subsystem 299' >> /var/log/boot.log
     echo 'step 300: configuring subsystem 300' >> /var/log/boot.log
    ```
- `aws_launch_template.gz` (update)
  - `user_data: 4 lines changed of 412 (+2 -2)`
    ```diff
    @@ -99,5 +99,5 @@
     echo 'step 97: configuring subsystem 97' >> /var/log/boot.log
     echo 'step 98: configuring subsystem 98' >> /var/log/boot.log
    -echo 'step 99: configuring subsystem 99' >> /var/log/boot.log
    +echo 'step 99: configuring subsystem 99 (patched)' >> /var/log/boot.log
     echo 'step 100: configuring subsystem 100' >> /var/log/boot.log
     echo 'step 101: configuring subsystem 101' >> /var/log/boot.log
    @@ -249,4 +249,5 @@
     echo 'step 247: configuring subsystem 247' >> /var/log/boot.log
     echo 'step 248: configuring subsystem 248' >> /var/log/boot.log
    +systemctl enable --now node_exporter
     echo 'step 249: configuring subsystem 249' >> /var/log/boot.log
     echo 'step 250: configuring subsystem 250' >> /var/log/boot.log
    @@ -298,5 +299,4 @@
     echo 'step 296: configuring subsystem 296' >> /var/log/boot.log
     echo 'step 297: configuring subsystem 297' >> /var/log/boot.log
    -echo 'step 298: configuring subsystem 298' >> /var/log/boot.log
     echo 'step 299: configuring subsystem 299' >> /var/log/boot.log
     echo 'step 300: configuring subsystem 300' >> /var/log/boot.log
    ```
- `aws_instance.plain` (update)
  - `user_data: 4 lines changed of 412 (+2 -2)`
    ```diff
    @@ -99,5 +99,5 @@
     echo 'step 97: configuring subsystem 97' >> /var/log/boot.log
     echo 'step 98: configuring subsystem 98' >> /var/log/boot.log
    -echo 'step 99: configuring subsystem 99' >> /var/log/boot.log
    +echo 'step 99: configuring subsystem 99 (patched)' >> /var/log/boot.log
     echo 'step 100: configuring subsystem 100' >> /var/log/boot.log
     echo 'step 101: configuring subsystem 101' >> /var/log/boot.log
    @@ -249,4 +249,5 @@
     echo 'step 247: configuring subsystem 247' >> /var/log/boot.log
     echo 'step 248: configuring subsystem 248' >> /var/log/boot.log
    +systemctl enable --now node_exporter
     echo 'step 249: configuring subsystem 249' >> /var/log/boot.log
     echo 'step 250: configuring subsystem 250' >> /var/log/boot.log
    @@ -298,5 +299,4 @@
     echo 'step 296: configuring subsystem 296' >> /var/log/boot.log
     echo 'step 297: configuring subsystem 297' >> /var/log/boot.log
    -echo 'step 298: configuring subsystem 298' >> /var/log/boot.log
     echo 'step 299: configuring subsystem 299' >> /var/log/boot.log
     echo 'step 300: configuring subsystem 300' >> /var/log/boot.log
    ```

</details>

### Changes by module

<details><summary><b>root</b> &mdash; 3 change(s)</summary>

| Action | Resource | Target account |
|---|---|---|
| update | `aws_instance.app` | `unresolved (alias: default)` (unresolved) |
| update | `aws_launch_template.gz` | `unresolved (alias: default)` (unresolved) |
| update | `aws_instance.plain` | `unresolved (alias: default)` (unresolved) |

</details>

