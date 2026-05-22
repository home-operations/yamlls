package schema

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-yaml/ast"
	"github.com/home-operations/yamlls/internal/config"
)

// Resolver picks the FILE-LEVEL schema URL for a document. Order:
// modeline, workspace settings glob, JSON Schema Store catalog.
// Kubernetes apiVersion+kind auto-detection is per-document and lives
// in DetectKubernetesGVKFromNode — callers handle that themselves so
// multi-doc files with mixed kinds resolve correctly.
type Resolver struct {
	mu       sync.RWMutex
	settings config.Settings
	catalog  *Catalog
}

func NewResolver() *Resolver {
	r := &Resolver{}
	r.SetSettings(config.Settings{})
	return r
}

func (r *Resolver) SetSettings(s config.Settings) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings = s
	if s.CatalogEnabled() {
		if r.catalog == nil || r.catalog.URL != effectiveCatalogURL(s) {
			r.catalog = NewCatalog(s.CatalogURL)
		}
	} else {
		r.catalog = nil
	}
}

func effectiveCatalogURL(s config.Settings) string {
	if s.CatalogURL != "" {
		return s.CatalogURL
	}
	return DefaultCatalogURL
}

func (r *Resolver) Resolve(text, docPath string) string {
	if ref := FindModelineSchema(text); ref != "" {
		return ref
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ref := matchSettings(r.settings.Schemas, docPath); ref != "" {
		return ref
	}
	if r.catalog != nil {
		if ref := r.catalog.Match(docPath); ref != "" {
			return ref
		}
	}
	return ""
}

func matchSettings(schemas map[string][]string, docPath string) string {
	if docPath == "" {
		return ""
	}
	norm := strings.ReplaceAll(docPath, string(filepath.Separator), "/")
	for ref, globs := range schemas {
		for _, g := range globs {
			if matchGlob(g, norm) {
				return ref
			}
			// Anchored globs like "k8s/**/*.yaml" should also match an
			// absolute path that ends with the same suffix.
			if !startsAnchored(g) && matchGlob("**/"+g, norm) {
				return ref
			}
		}
	}
	return ""
}

func startsAnchored(g string) bool {
	return len(g) > 0 && (g[0] == '/' || (len(g) >= 2 && g[0] == '*' && g[1] == '*'))
}

// K8sURLForNode runs apiVersion+kind detection on the given document body
// and renders the configured kubernetes.schemaUrl template (yannh default).
// Returns "" when the document isn't a Kubernetes manifest.
func (r *Resolver) K8sURLForNode(body ast.Node) string {
	gvk, ok := DetectGVK(body)
	if !ok {
		return ""
	}
	return r.K8sURL(gvk)
}

// K8sURL renders the configured URL template for an already-parsed GVK.
// Used by the renderer pipeline where the GVK is known up-front.
func (r *Resolver) K8sURL(gvk GVK) string {
	r.mu.RLock()
	tmpl := ""
	if r.settings.Kubernetes != nil {
		tmpl = r.settings.Kubernetes.SchemaURL
	}
	r.mu.RUnlock()
	return BuildK8sURL(tmpl, gvk.Group, gvk.Version, gvk.Kind)
}
