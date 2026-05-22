// Package flate is the Renderer adapter for home-operations/flate.
package flate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"

	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
	"github.com/home-operations/yamlls/internal/render"
)

const providerName = "flate"

type Renderer struct {
	Binary string

	mu         sync.Mutex
	disabled   bool
	resolved   string
	lookupOnce sync.Once
}

func New() *Renderer { return &Renderer{Binary: providerName} }

type fileConfig struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Binary  string `json:"binary,omitempty"`
}

func (r *Renderer) Configure(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var cfg fileConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cfg.Enabled != nil {
		r.disabled = !*cfg.Enabled
	}
	if cfg.Binary != "" && cfg.Binary != r.Binary {
		r.Binary = cfg.Binary
		r.resolved = ""
		r.lookupOnce = sync.Once{}
	}
	return nil
}

func (r *Renderer) IsEnabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.disabled
}

func (r *Renderer) Name() string { return providerName }

func (r *Renderer) Matches(doc *render.SourceDocument) bool {
	return render.MatchesKind(doc, "HelmRelease", "helm.toolkit.fluxcd.io") ||
		render.MatchesKind(doc, "Kustomization", "kustomize.toolkit.fluxcd.io")
}

func (r *Renderer) Render(ctx context.Context, doc *render.SourceDocument) (*render.RenderedOutput, error) {
	sub, dir, err := subcommandFor(doc)
	if err != nil {
		return nil, err
	}
	bin, err := r.resolveBinary()
	if err != nil {
		return &render.RenderedOutput{Provider: r.Name()}, err
	}
	cmd := exec.CommandContext(ctx, bin, "build", sub, "--path", dir, "-o", "yaml")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	out := &render.RenderedOutput{
		Provider: r.Name(),
		Raw:      append([]byte(nil), stdout.Bytes()...),
		Stderr:   append([]byte(nil), stderr.Bytes()...),
	}
	if runErr != nil {
		return out, fmt.Errorf("flate %s: %w (stderr: %s)", sub, runErr, truncate(stderr.String(), 512))
	}
	manifests, err := parseManifests(stdout.Bytes())
	if err != nil {
		return out, fmt.Errorf("flate %s: parse output: %w", sub, err)
	}
	out.Manifests = manifests
	return out, nil
}

func (r *Renderer) resolveBinary() (string, error) {
	r.lookupOnce.Do(func() {
		name := r.Binary
		if name == "" {
			name = providerName
		}
		if abs, err := exec.LookPath(name); err == nil {
			r.resolved = abs
		}
	})
	if r.resolved == "" {
		return "", errors.New("flate binary not found on PATH — install with " +
			"`go install github.com/home-operations/flate/cmd/flate@latest`")
	}
	return r.resolved, nil
}

func subcommandFor(doc *render.SourceDocument) (sub, dir string, err error) {
	if doc == nil {
		return "", "", errors.New("nil doc")
	}
	switch {
	case render.MatchesKind(doc, "HelmRelease", "helm.toolkit.fluxcd.io"):
		sub = "hr"
	case render.MatchesKind(doc, "Kustomization", "kustomize.toolkit.fluxcd.io"):
		sub = "ks"
	default:
		return "", "", fmt.Errorf("flate: unsupported kind %q", doc.Kind)
	}
	if doc.Path == "" {
		return sub, "", errors.New("flate needs an on-disk file path")
	}
	return sub, filepath.Dir(doc.Path), nil
}

func parseManifests(stdout []byte) ([]render.RenderedManifest, error) {
	if len(bytes.TrimSpace(stdout)) == 0 {
		return nil, nil
	}
	f, err := parser.ParseBytes(stdout, 0)
	if err != nil {
		return nil, err
	}
	out := make([]render.RenderedManifest, 0, len(f.Docs))
	for _, d := range f.Docs {
		if d.Body == nil {
			continue
		}
		var head struct {
			APIVersion string `yaml:"apiVersion"`
			Kind       string `yaml:"kind"`
			Metadata   struct {
				Name string `yaml:"name"`
			} `yaml:"metadata"`
		}
		if err := yaml.NodeToValue(d.Body, &head); err != nil {
			continue
		}
		if head.Kind == "" {
			continue
		}
		group, version := splitAPIVersion(head.APIVersion)
		out = append(out, render.RenderedManifest{
			AST:  d,
			GVK:  render.GVK{Group: group, Version: version, Kind: head.Kind},
			Name: head.Metadata.Name,
		})
	}
	return out, nil
}

func splitAPIVersion(v string) (group, version string) {
	if v == "" {
		return "", ""
	}
	if g, ver, ok := splitOnce(v, '/'); ok {
		return g, ver
	}
	return "", v
}

func splitOnce(s string, sep byte) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
