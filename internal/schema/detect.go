package schema

import (
	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// GVK identifies a Kubernetes type extracted from a YAML document.
type GVK struct {
	Group   string
	Version string
	Kind    string
}

// DetectGVK extracts apiVersion+kind from a YAML document body. Returns
// !ok when the document isn't a recognizable Kubernetes manifest.
func DetectGVK(body ast.Node) (GVK, bool) {
	if body == nil {
		return GVK{}, false
	}
	var head struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}
	if err := yaml.NodeToValue(body, &head); err != nil {
		return GVK{}, false
	}
	if head.APIVersion == "" || head.Kind == "" {
		return GVK{}, false
	}
	group, version := splitOnce(head.APIVersion, '/')
	if version == "" {
		version = group
		group = ""
	}
	return GVK{Group: group, Version: version, Kind: head.Kind}, true
}

// DetectKubernetesGVK reads the first YAML document's apiVersion+kind
// and returns the matching yannh/kubernetes-json-schema URL.
func DetectKubernetesGVK(text string) string {
	f, err := parser.ParseBytes([]byte(text), 0)
	if err != nil || f == nil || len(f.Docs) == 0 {
		return ""
	}
	return DetectKubernetesGVKFromNode(f.Docs[0].Body)
}

// DetectKubernetesGVKFromNode is the per-document variant.
func DetectKubernetesGVKFromNode(body ast.Node) string {
	gvk, ok := DetectGVK(body)
	if !ok {
		return ""
	}
	return KubernetesSchemaURL(gvk.Group, gvk.Version, gvk.Kind)
}

func splitOnce(s string, sep byte) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
