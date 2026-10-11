# Third-party notices

tfplanlens is MIT licensed (see [LICENSE](LICENSE)). The binary links the Go
modules below. Each keeps its own license, which applies to that module only;
their full texts are in the module sources (`go mod download`, or the upstream
repositories). The Go standard library is BSD-3-Clause, by The Go Authors.

| Module | License | Copyright / source |
|---|---|---|
| `github.com/hashicorp/terraform-json` | MPL-2.0 | Copyright IBM Corp.; https://github.com/hashicorp/terraform-json |
| `github.com/hashicorp/go-version` | MPL-2.0 | Copyright IBM Corp.; https://github.com/hashicorp/go-version |
| `github.com/zclconf/go-cty` | MIT | Copyright Martin Atkins; https://github.com/zclconf/go-cty |
| `github.com/apparentlymart/go-textseg/v15` | MIT (its Unicode table generators are Apache-2.0, its grapheme data under the Unicode license) | Copyright Martin Atkins; https://github.com/apparentlymart/go-textseg |
| `github.com/spf13/cobra` | Apache-2.0 | https://github.com/spf13/cobra |
| `github.com/spf13/pflag` | BSD-3-Clause | Copyright Alex Ogier and The Go Authors; https://github.com/spf13/pflag |
| `github.com/inconshreveable/mousetrap` (Windows builds only) | Apache-2.0 | https://github.com/inconshreveable/mousetrap |
| `gopkg.in/yaml.v3` | MIT and Apache-2.0 (see its `LICENSE` and `NOTICE`) | https://github.com/go-yaml/yaml |
| `github.com/pmezard/go-difflib` | BSD-3-Clause | Copyright Patrick Mezard; https://github.com/pmezard/go-difflib |
| `golang.org/x/sys`, `golang.org/x/term`, `golang.org/x/text` | BSD-3-Clause | Copyright The Go Authors; https://pkg.go.dev/golang.org/x |

## MPL-2.0 modules

`terraform-json` and `go-version` are licensed under the Mozilla Public License
2.0. tfplanlens uses them unmodified. Their source code is available at the
URLs above, and the MPL-2.0 text is at https://www.mozilla.org/MPL/2.0/.

## Test-only modules

`golang.org/x/net` (BSD-3-Clause) is used only by tests and is not part of the
binary.

## Trademarks

Terraform and HashiCorp are trademarks of HashiCorp, Inc. (an IBM company), and
AWS and Amazon Web Services are trademarks of Amazon.com, Inc. tfplanlens is an
independent project and is not affiliated with, endorsed by or sponsored by
either.
