# CLI reference

```
tfplanlens report <plan.json|-> [more plans...]   [-f text|html|markdown|json|brief]... [--top N] [-o file]
              [-c config] [--fail-on rules] [--tfapply-account-id ID] [--account alias=id]... [--no-color]
tfplanlens diff <old.json> <new.json>   [-f text|markdown|json] [-o file] [-c config]
              [--fail-on-new-danger] [--tfapply-account-id ID] [--account alias=id]...
tfplanlens tui <plan.json> [more plans...]   [-c config] [--tfapply-account-id ID] [--account alias=id]...
tfplanlens version
```

Several plans: addresses become `label:address`; label is the file name, the
directory for `plan.json`/`tfplan.json`, or `label=path`. `-` (stdin) works
only with one plan. `tui` needs a terminal.

## Exit codes

`0` ok; `1` a `--fail-on` / `--fail-on-new-danger` rule matched (the report is
still written first); `2` error (bad input, flag, config).

## `--fail-on` rules (comma-separated)

`delete`, `replace`, `drift`, `unresolved` (an account that could not be
determined), or `kind:type` such as `replace:aws_db_instance`. `delete`
matches pure deletes; replacements are `replace`.

## Apply account precedence

flag `--tfapply-account-id` > env `TFPLANLENS_TFAPPLY_ACCOUNT_ID` > config
`tfapply_account_id` > inferred (default unaliased provider without
assume_role) > unknown (reported, never prompted for).

## Config file (`./tfplanlens.config.yaml`, strict)

`tfapply_account_id`, `accounts` (alias or provider key to id), `format`,
`fail_on`, `max_line`, `stateful_types`, `long_value_keys`, `ignore_types`.
Flags override it. Annotated example: `tfplanlens.config.yaml` at the root.
