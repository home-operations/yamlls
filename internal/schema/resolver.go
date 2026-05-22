package schema

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/home-operations/yamlls/internal/config"
)

// Resolver picks the schema URL for a document. Order: modeline,
// workspace settings, Kubernetes apiVersion+kind, JSON Schema Store catalog.
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
	if ref := DetectKubernetesGVK(text); ref != "" {
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
