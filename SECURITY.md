# Security policy

## Reporting a vulnerability

Please **do not open a public issue** for a security problem. Use GitHub's
private reporting: on this repository, go to **Security** > **Report a
vulnerability**. If that option is not available, open an issue that says only
"security report, please contact me" and no details, and a maintainer will
arrange a private channel.

Include the tfplanlens version (`tfplanlens version`), what you did, what you
expected and what happened. **Do not attach a real Terraform plan or state
file**: they contain secret values in plain text. A minimal fabricated plan that
reproduces the problem is ideal.

You can expect an acknowledgement, but this is a small volunteer project and
there is no guaranteed response time.

## What counts

tfplanlens reads a plan file offline and prints a report; it never contacts AWS
or Terraform. The issues that matter most:

- **A sensitive value printed** in any output format (text, markdown, HTML,
  JSON, `diff`, `tui`). A value Terraform marks sensitive, or one tfplanlens
  promises to hide (see [docs/SECRETS.md](docs/SECRETS.md)), appearing in a report
  is treated as a vulnerability.
- **A common credential format that is shown instead of hidden.** Report it
  privately; it will be added to the redaction patterns.
- **Script or markup injection** through plan content in the HTML or markdown
  report (resource names, attribute values and so on).
- **A crash, hang or runaway memory use** from a crafted plan file.
- Anything that makes the tool read or write files it was not asked to.

Not a vulnerability: a plan that contains secrets because Terraform put them
there (see [`field-notes/TERRAFORM_GOTCHAS.md`](field-notes/TERRAFORM_GOTCHAS.md));
a secret with an innocent name and no recognizable shape that the best-effort
filter cannot know about (the limits are listed in [docs/SECRETS.md](docs/SECRETS.md));
the report being wrong about a resource (open a normal issue).

## Supported versions

There are no published releases yet. Only the `main` branch is supported.
