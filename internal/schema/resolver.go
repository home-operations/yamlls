package schema

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-yaml/ast"
	"github.com/home-operations/yamlls/internal/config"
)

// Resolver picks the file-level schema URL for a document: modeline,
// workspace `schemas:` glob, JSON Schema Store catalog. Kubernetes
// apiVersion+kind detection is per-document and lives on K8sURLForNode
// so multi-doc files with mixed kinds resolve correctly.
type Resolver struct {
	mu         sync.RWMutex
	settings   config.Settings
	catalog    *Catalog
	reloadHook func()
}

func NewResolver() *Resolver {
	r := &Resolver{}
	r.SetSettings(config.Settings{})
	return r
}

// SetReloadHook registers a callback invoked when an asynchronously loaded
// catalog becomes available, so the server can re-publish diagnostics for
// documents whose schema is resolved via the catalog.
func (r *Resolver) SetReloadHook(fn func()) {
	r.mu.Lock()
	r.reloadHook = fn
	r.mu.Unlock()
}

func (r *Resolver) SetSettings(s config.Settings) {
	r.mu.Lock()
	r.settings = s
	var toLoad *Catalog
	if s.CatalogEnabled() {
		if r.catalog == nil || r.catalog.URL != effectiveCatalogURL(s) {
			r.catalog = NewCatalog(s.CatalogURL)
			toLoad = r.catalog
		}
	} else {
		r.catalog = nil
	}
	hook := r.reloadHook
	r.mu.Unlock()

	if toLoad != nil {
		toLoad.Load(hook)
	}
}

func effectiveCatalogURL(s config.Settings) string {
	if s.CatalogURL != "" {
		return s.CatalogURL
	}
	return DefaultCatalogURL
}

// WaitForCatalog blocks until the background catalog load has finished, or
// returns immediately if the catalog is disabled. One-shot callers use this
// so filename-based matches resolve on the first pass.
func (r *Resolver) WaitForCatalog() {
	r.mu.RLock()
	catalog := r.catalog
	r.mu.RUnlock()
	if catalog != nil {
		catalog.Wait()
	}
}

func (r *Resolver) Resolve(text, docPath string) string {
	if ref := FindModelineSchema(text); ref != "" {
		return ref
	}
	r.mu.RLock()
	schemas := r.settings.Schemas
	catalog := r.catalog
	r.mu.RUnlock()

	if ref := matchSettings(schemas, docPath); ref != "" {
		return ref
	}
	if isYamllsConfigPath(docPath) {
		return EmbeddedYamllsSchemaURL
	}
	// Match never blocks: the catalog loads in the background, so this
	// returns "" until it's ready rather than fetching under the read lock.
	if catalog != nil {
		if ref := catalog.Match(docPath); ref != "" {
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

// K8sURLForNode renders the configured template for the apiVersion+kind
// found in body, or "" when the document isn't a Kubernetes manifest.
func (r *Resolver) K8sURLForNode(body ast.Node) string {
	gvk, ok := DetectGVK(body)
	if !ok {
		return ""
	}
	return r.K8sURL(gvk)
}

func (r *Resolver) K8sURL(gvk GVK) string {
	r.mu.RLock()
	tmpl := ""
	if r.settings.Kubernetes != nil {
		tmpl = r.settings.Kubernetes.SchemaURL
	}
	r.mu.RUnlock()
	return BuildK8sURL(tmpl, gvk.Group, gvk.Version, gvk.Kind)
}
