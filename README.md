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

Minimal extension:

```jsonc
// package.json
{
  "name": "yamlls",
  "engines": { "vscode": "^1.85.0" },
  "activationEvents": ["onLanguage:yaml"],
  "main": "./extension.js"
}
```

```js
// extension.js  (vscode-languageclient v9+)
const { LanguageClient } = require("vscode-languageclient/node");

let client;
exports.activate = function (ctx) {
  client = new LanguageClient(
    "yamlls",
    "yamlls",
    { command: "yamlls" },
    { documentSelector: [{ scheme: "file", language: "yaml" }] }
  );
  client.start();              // returns Promise<void>, fire-and-forget
  ctx.subscriptions.push(client);
};
exports.deactivate = () => client?.stop();
```

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

Zed's `lsp` key only accepts known language-server identifiers, so
`yamlls` as a top-level key triggers `Property yamlls is not allowed`.
Override Zed's bundled `yaml-language-server` binary instead:

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
