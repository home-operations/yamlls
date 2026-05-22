# yamlls

YAML language server in Go. Schema-driven diagnostics, completion, and
hover; pluggable rendering for Flux `HelmRelease` and `Kustomization`
sources via [home-operations/flate][flate].

## Install

```sh
go install github.com/home-operations/yamlls/cmd/yamlls@latest
```

For Flux rendering:

```sh
go install github.com/home-operations/flate/cmd/flate@latest
```

## Editor setup

See [docs/SETUP.md](docs/SETUP.md).

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

# Override the URL template used by Kubernetes apiVersion+kind auto-detect.
# Placeholders: {group}, {groupSeg}, {groupFirst}, {kind}, {kindLower},
# {version}, {versionLower}. Unset = yannh/kubernetes-json-schema layout.
kubernetes:
  schemaUrl: "https://schemas.example.com/{groupSeg}{kindLower}_{versionLower}.json"

renderers:
  flate:
    enabled: true
    binary: flate
```

Same shape works via `initializationOptions` or
`workspace/didChangeConfiguration`. Precedence (low → high):
`.yamlls.yaml` → `initializationOptions` → `didChangeConfiguration`.

## Commands

- `yamlls.showRendered` — returns the renderer's output for a
  `HelmRelease`/`Kustomization` URI.

## Development

```sh
mise install   # toolchain
mise run test
mise run lint
mise run build
```

[flate]: https://github.com/home-operations/flate
