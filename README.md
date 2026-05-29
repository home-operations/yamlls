# yamlls

YAML language server in Go. Schema-driven diagnostics, completion, and
hover; pluggable rendering for Flux `HelmRelease` and `Kustomization`
sources via [home-operations/flate][flate].

Per-document schema resolution, highest priority first:

1. in-file modeline (`# yaml-language-server: $schema=…`)
2. workspace `schemas:` glob in `.yamlls.yaml`
3. JSON Schema Store catalog (filename match)
4. Kubernetes `apiVersion`+`kind` → `kubernetes.schemaUrl` template

Multi-doc files validate each document against its own schema. The
default `kubernetes.schemaUrl` is
`https://k8s-schemas.home-operations.com/{groupSeg}{kindLower}_{versionLower}.json`;
override in `.yamlls.yaml` to point elsewhere. 404s are silently skipped.

## vs. redhat/yaml-language-server

|                                                | yamlls                                       | redhat/yaml-language-server                                  |
| ---------------------------------------------- | -------------------------------------------- | ------------------------------------------------------------ |
| Runtime                                        | static Go binary                             | Node.js ≥ 12                                                 |
| Diagnostics, completion, hover                 | yes                                          | yes                                                          |
| Symbols, folding, links, code actions          | yes                                          | yes                                                          |
| Code lens                                      | rendered output, diff                        | none                                                         |
| Per-doc schema in multi-doc files              | dynamic, per `---` block from apiVersion+kind | pre-declared ordered list via `schemaSequence`              |
| Kubernetes auto-detect                         | URL template from apiVersion+kind            | `yaml.kubernetesCRDStore` ([datreeio/CRDs-catalog][datree])  |
| Workspace config file                          | `.yamlls.yaml`                               | editor settings only                                         |
| Flux `HelmRelease` / `Kustomization` rendering | via [flate][flate]                           | no                                                           |
| Formatting                                     | no                                           | yes (Prettier)                                               |
| Custom YAML tags (`!Ref`, …)                   | no                                           | yes                                                          |
| Diagnostic suppression comments                | no                                           | yes                                                          |
| JSON Schema drafts                             | 04, 06, 07, 2019-09, 2020-12                 | 04, 07, 2019-09, 2020-12                                     |

[datree]: https://github.com/datreeio/CRDs-catalog

## Install

Homebrew:

```sh
brew install home-operations/tap/yamlls
```

Go:

```sh
go install github.com/home-operations/yamlls/cmd/yamlls@latest
```

Prebuilt binaries for linux/darwin/windows (amd64+arm64) are attached to
each [GitHub release](https://github.com/home-operations/yamlls/releases).

For Flux rendering:

```sh
go install github.com/home-operations/flate/cmd/flate@latest
```

## Editor setup

`yamlls` speaks LSP 3.16 over stdio. Put the binary on `$PATH` or pass
an absolute path.

Packaged extensions for **VS Code** and **Zed** live in [`editors/`](editors);
they download the matching `yamlls` (and `flate`) release binary automatically.
The snippets below are for editors with built-in LSP support.

### Neovim (nvim-lspconfig)

```lua
local lspconfig = require("lspconfig")
local configs = require("lspconfig.configs")

if not configs.yamlls then
  configs.yamlls = {
    default_config = {
      cmd = { "yamlls" },
      filetypes = { "yaml" },
      root_dir = lspconfig.util.find_git_ancestor,
      single_file_support = true,
    },
  }
end

lspconfig.yamlls.setup({})
```

### VSCode

Use the extension in [`editors/vscode`](editors/vscode) — it downloads the
`yamlls` binary (and `flate` for Flux rendering) on first activation, and
exposes `yamlls.*` settings. To build and run it locally, press <kbd>F5</kbd>
from that directory; to package a `.vsix`, run `vsce package`. See its
[README](editors/vscode/README.md) for settings and publishing.

### Helix

```toml
# ~/.config/helix/languages.toml
[language-server.yamlls]
command = "yamlls"

[[language]]
name = "yaml"
language-servers = ["yamlls"]
```

### Zed

Use the extension in [`editors/zed`](editors/zed) — it registers `yamlls` as a
language server for the YAML language and downloads the binary for you (install
it via **zed: install dev extension**). Since Zed bundles its own
`yaml-language-server`, make `yamlls` the only one in
`~/.config/zed/settings.json`:

```jsonc
{
  "languages": {
    "YAML": { "language_servers": ["yamlls", "!yaml-language-server"] }
  }
}
```

Without the extension, Zed's `lsp` key only accepts known language-server
identifiers (`yamlls` as a top-level key triggers `Property yamlls is not
allowed`), so the settings-only alternative is to override the bundled
`yaml-language-server` binary:

```jsonc
// ~/.config/zed/settings.json
{
  "lsp": {
    "yaml-language-server": {
      "binary": {
        "ignore_system_version": true,
        "path": "yamlls"
      },
      "initialization_options": {
        "catalog": true
      }
    }
  }
}
```

### Flux rendering

With [flate][flate] installed, opening a `HelmRelease` or
`Kustomization` surfaces schema violations on rendered manifests as
`[rendered <kind>/<name> @ <jsonptr>]` on the source document. The
`yamlls.showRendered` command returns the rendered YAML; in Neovim:

```lua
vim.lsp.buf.execute_command({
  command = "yamlls.showRendered",
  arguments = { vim.uri_from_bufnr(0) },
})
```

### Debugging

```sh
yamlls --log-file /tmp/yamlls.log -v 2
```

`-v 0` is silent (default), `1` is info, `2+` is debug.

## Configuration

`.yamlls.yaml` in the workspace root:

```yaml
schemas:
  "https://json.schemastore.org/github-workflow.json":
    - ".github/workflows/*.yml"
  "./schemas/local.json":
    - "k8s/**/*.yaml"

catalog: true
catalogUrl: ""

# Optional. Override the URL template for Kubernetes auto-detect.
# Placeholders: {group}, {groupSeg}, {groupFirst}, {groupFirstSeg},
# {kind}, {kindLower}, {version}, {versionLower}.
# kubernetes:
#   schemaUrl: "https://schemas.example.com/{groupSeg}{kindLower}_{versionLower}.json"

# Optional. Defaults shown.
# renderers:
#   flate:
#     enabled: true
#     binary: flate
```

See [`.yamlls.yaml.example`](.yamlls.yaml.example) for a copyable starter.

Same shape works via `initializationOptions` or
`workspace/didChangeConfiguration`. Precedence (low → high):
`.yamlls.yaml` → `initializationOptions` → `didChangeConfiguration`.

## Capabilities

`textDocument/`: diagnostics, completion, hover, foldingRange,
documentLink, documentSymbol, codeAction (enum quick-fix), codeLens.

`workspace/`: didChangeConfiguration, didChangeWorkspaceFolders, executeCommand.

## Commands

- `yamlls.showRendered <uri>` — rendered output for a Flux source.
- `yamlls.showRenderedDiff <uri>` — unified diff between the open-time
  render and the current render.

## CLI flags

```
yamlls --version              print version and exit
yamlls --log-file PATH        append logs to PATH instead of stderr
yamlls -v N                   log verbosity (0=silent, 1=info, 2+=debug)
```

## Development

```sh
mise install   # toolchain
mise run test
mise run lint
mise run build
```

[flate]: https://github.com/home-operations/flate
