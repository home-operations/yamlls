package render

import (
	"encoding/json"
	"strings"
	"sync"

	yaml "github.com/goccy/go-yaml"
	"github.com/home-operations/yamlls/internal/yamlast"
)

type Configurable interface {
	Configure(raw json.RawMessage) error
}

type Enableable interface {
	IsEnabled() bool
}

type Registry struct {
	mu        sync.RWMutex
	providers []Renderer
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) Register(p Renderer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = append(r.providers, p)
}

func (r *Registry) For(doc *SourceDocument) Renderer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.providers {
		if en, ok := p.(Enableable); ok && !en.IsEnabled() {
			continue
		}
		if p.Matches(doc) {
			return p
		}
	}
	return nil
}

func (r *Registry) Configure(configs map[string]json.RawMessage) {
	r.mu.RLock()
	providers := append([]Renderer(nil), r.providers...)
	r.mu.RUnlock()
	for _, p := range providers {
		raw, ok := configs[p.Name()]
		if !ok {
			continue
		}
		if c, ok := p.(Configurable); ok {
			_ = c.Configure(raw)
		}
	}
}

func (r *Registry) All() []Renderer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Renderer, len(r.providers))
	copy(out, r.providers)
	return out
}

func AnalyzeDocument(uri, path, text string) *SourceDocument {
	parsed := yamlast.Parse([]byte(text))
	if parsed.File == nil || len(parsed.File.Docs) == 0 {
		return nil
	}
	doc := parsed.File.Docs[0]
	if doc.Body == nil {
		return nil
	}
	var head struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}
	if err := yaml.NodeToValue(doc.Body, &head); err != nil {
		return nil
	}
	if head.Kind == "" {
		return nil
	}
	group, version := splitAPIVersion(head.APIVersion)
	return &SourceDocument{
		URI:      uri,
		Path:     path,
		Text:     text,
		AST:      parsed.File,
		Kind:     head.Kind,
		APIGroup: group + versionSep(group, version),
	}
}

func versionSep(group, version string) string {
	if group == "" {
		return version
	}
	if version == "" {
		return ""
	}
	return "/" + version
}

func splitAPIVersion(v string) (group, version string) {
	if v == "" {
		return "", ""
	}
	if !strings.Contains(v, "/") {
		return "", v
	}
	g, ver, _ := strings.Cut(v, "/")
	return g, ver
}

// MatchesKind matches doc.Kind exactly and doc.APIGroup by prefix so
// "helm.toolkit.fluxcd.io" matches any of v2beta1, v2beta2, v2.
func MatchesKind(doc *SourceDocument, kind, group string) bool {
	if doc == nil {
		return false
	}
	if doc.Kind != kind {
		return false
	}
	return strings.HasPrefix(doc.APIGroup, group)
}
