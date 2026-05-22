package config

import (
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/goccy/go-yaml"
)

const WorkspaceConfigFile = ".yamlls.yaml"

// LoadFromWorkspace reads `.yamlls.yaml` from the workspace root. Relative
// schema paths in the file are anchored at the workspace root so they
// don't resolve from the open document's directory.
func LoadFromWorkspace(rootURI string) (Settings, error) {
	root := workspacePath(rootURI)
	if root == "" {
		return Settings{}, nil
	}
	s, err := LoadFile(filepath.Join(root, WorkspaceConfigFile))
	if err != nil {
		return s, err
	}
	expandSchemaPaths(&s, root)
	return s, nil
}

func expandSchemaPaths(s *Settings, root string) {
	if len(s.Schemas) == 0 {
		return
	}
	expanded := make(map[string][]string, len(s.Schemas))
	for ref, globs := range s.Schemas {
		expanded[expandRef(ref, root)] = globs
	}
	s.Schemas = expanded
}

func expandRef(ref, root string) string {
	if ref == "" || filepath.IsAbs(ref) {
		return ref
	}
	for _, prefix := range []string{"http://", "https://", "file://"} {
		if strings.HasPrefix(ref, prefix) {
			return ref
		}
	}
	return filepath.Clean(filepath.Join(root, ref))
}

// LoadFile reads a YAML config file. A missing file returns Settings{}
// and nil — callers should not treat absence as an error.
func LoadFile(path string) (Settings, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, nil
		}
		return Settings{}, err
	}
	jsonBytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(jsonBytes, &s); err != nil {
		return Settings{}, err
	}
	return s, nil
}

func workspacePath(rootURI string) string {
	if rootURI == "" {
		return ""
	}
	u, err := url.Parse(rootURI)
	if err != nil || u.Scheme != "file" {
		return ""
	}
	return u.Path
}

// Merge applies override on top of base. Non-zero scalars and non-nil
// maps in override win; map keys union.
func Merge(base, override Settings) Settings {
	out := base
	if override.Schemas != nil {
		if out.Schemas == nil {
			out.Schemas = make(map[string][]string)
		}
		maps.Copy(out.Schemas, override.Schemas)
	}
	if override.Catalog != nil {
		out.Catalog = override.Catalog
	}
	if override.CatalogURL != "" {
		out.CatalogURL = override.CatalogURL
	}
	if override.Renderers != nil {
		if out.Renderers == nil {
			out.Renderers = make(map[string]json.RawMessage)
		}
		maps.Copy(out.Renderers, override.Renderers)
	}
	return out
}
