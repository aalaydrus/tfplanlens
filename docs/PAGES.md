# Host the HTML report on GitHub Pages

**Live demo:** the repository publishes its example reports (a made-up AWS stack on an emulator, dummy account IDs) with [`.github/workflows/pages.yml`](../.github/workflows/pages.yml). The main page ([`docs/demo/index.html`](demo/index.html)) explains the tool; [`docs/demo/reports.html`](demo/reports.html) lists the example reports.

`tfplanlens report -f html -o index.html` writes one self-contained file (no CDN,
no external requests), so any static host works, GitHub Pages included.
[`examples/publish-report-pages.yml`](examples/publish-report-pages.yml) is a
copy-paste workflow for your own plans: it assumes an AWS role over OIDC, runs
`terraform plan`, renders the report and deploys it. All account IDs and role
names in it are dummy values. It is opt-in and has not been run against a real
Pages deployment; tell us if it needs changes.

## Read this before you publish

A plan report lists resource names, addresses, account IDs and the shape of your
infrastructure. Sensitive values are hidden, and secret-looking values are
redacted on a best-effort basis ([limits](SECRETS.md)), but the rest is shown.

- On a public repository a Pages site is **public**, and so are old versions
  you cached or that were indexed.
- Access-controlled Pages for private repositories depends on your GitHub plan.
  Check GitHub's current documentation rather than assuming.
- If the report must stay inside your team, prefer
  [`actions/upload-artifact`](https://github.com/actions/upload-artifact)
  (people with repo access download the HTML from the run) or the
  `markdown` format as a pull request comment.

## Set up

1. Copy the example to `.github/workflows/` and replace the plan step.
2. In the repository settings, set the Pages source to **GitHub Actions**.
3. Run the workflow, then open the `page_url` it prints.

The example deploys one report to `index.html`, replacing the previous one. For
several plans in one page, pass them all to `report` (see [Usage](USAGE.md)).
