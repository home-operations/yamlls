# Editor setup

`yamlls` speaks LSP 3.16 over stdio.

## Neovim (nvim-lspconfig)

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

## VSCode

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
// extension.js
const { LanguageClient } = require("vscode-languageclient/node");

let client;
exports.activate = function (ctx) {
  client = new LanguageClient(
    "yamlls",
    "yamlls",
    { command: "yamlls" },
    { documentSelector: [{ scheme: "file", language: "yaml" }] }
  );
  ctx.subscriptions.push(client.start());
};
exports.deactivate = () => client && client.stop();
```

## Helix

```toml
# ~/.config/helix/languages.toml
[language-server.yamlls]
command = "yamlls"

[[language]]
name = "yaml"
language-servers = ["yamlls"]
```

## Zed

```jsonc
{
  "lsp": {
    "yamlls": { "binary": { "path": "yamlls" } }
  },
  "languages": {
    "YAML": { "language_servers": ["yamlls"] }
  }
}
```

## Flux rendering

Install flate:

```sh
go install github.com/home-operations/flate/cmd/flate@latest
```

Open a `HelmRelease` or `Kustomization`. Diagnostics on the source
document carry `[rendered <kind>/<name> @ <jsonptr>]` for schema
violations on the rendered manifests. The `yamlls.showRendered` command
returns the rendered YAML; in Neovim:

```lua
vim.lsp.buf.execute_command({
  command = "yamlls.showRendered",
  arguments = { vim.uri_from_bufnr(0) },
})
```
