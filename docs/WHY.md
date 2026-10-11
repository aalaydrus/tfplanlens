# Why not just read `terraform plan`?

You can, and for a small plan you should. It gets hard when the plan is big or noisy:

- **The dangerous lines are buried.** A delete or a forced replacement is one line among hundreds. tfplanlens lists every delete and replace first, with the reason in plain English ("count shrank; index 2 no longer exists", "for_each key removed: b", "Resource is tainted", "Attribute change forces replacement: triggers_replace"), a **STATEFUL** badge for databases, buckets, volumes and the like, and a warning when a `count`/`for_each` change looks like an accidental index shift.
- **Long values bury everything else.** In one real plan I measured (the `complex` scenario in `testdata/real`), Terraform's output was 105 lines, 25,835 characters, with a single line of **14,705 characters**: a base64 `user_data` shown before and after. The tfplanlens text report for the same plan was 58 lines (57 with an apply account given; longest 114 characters) and shows the decoded change as 2 removed and 2 added lines out of 401, with 2 lines of context. That is a summary, not a replacement: it does not list unchanged attributes or every create.
- **Known-after-apply noise.** Computed attributes that will change are counted per resource, not printed.
- **Secrets can leak through copies.** In that same real plan, Terraform marked a variable's `input` sensitive, but the computed `output` copy (the old secret) was printed in clear by `terraform plan` itself. tfplanlens masks any value equal to a sensitive sibling in the same resource. It is a heuristic and only covers copies within one resource.
- **Which account is this going to?** With role-assuming providers, the account a resource lands in is not the one you run from. tfplanlens shows both, and where each answer came from.
- **Drift, moves, imports and `removed` blocks** are called out instead of looking like deletes and creates.
