# yamlls for VS Code

LSP client for [`yamlls`](https://github.com/home-operations/yamlls). On first
activation it downloads the release binary matching your OS/arch into the
extension's global storage; no manual install needed.

## Develop / test locally

```sh
npm install
npm run compile
```

Then press <kbd>F5</kbd> ("Run Extension") to open an Extension Development Host,
and open any `.yaml` file.

## Package & publish

```sh
npm install -g @vscode/vsce
vsce package          # produces yamlls-<version>.vsix
vsce publish          # requires a Marketplace publisher + PAT
```

## Settings

| Setting | Default | Description |
| ------- | ------- | ----------- |
| `yamlls.path` | `""` | Absolute path to a `yamlls` binary. When set, skips the download. |
| `yamlls.version` | `"latest"` | Release tag to download (e.g. `"0.0.5"`). |
| `yamlls.catalog` | `true` | Enable JSON Schema Store catalog lookups. |
| `yamlls.catalogUrl` | `""` | Override the catalog URL. |
| `yamlls.schemas` | `{}` | Map of schema URI/path → file globs. |
| `yamlls.kubernetes.schemaUrl` | `""` | Override the Kubernetes auto-detect URL template. |

These mirror `.yamlls.yaml`; the workspace file still applies and takes lower
precedence than these editor settings.

## Commands

- **yamlls: Show Rendered Output** — render the active Flux `HelmRelease` /
  `Kustomization` (requires [`flate`](https://github.com/home-operations/flate)).
- **yamlls: Restart Language Server**

## Notes

- Extraction shells out to `tar`, available on macOS, Linux, and Windows 10
  1803+ (its `bsdtar` also handles the Windows `.zip` asset).
