package schema

import (
	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// DetectKubernetesGVK reads the first YAML document's apiVersion+kind and
// returns the matching yannh/kubernetes-json-schema URL, or "" when the
// document is not a Kubernetes manifest. Parse failures yield "".
func DetectKubernetesGVK(text string) string {
	f, err := parser.ParseBytes([]byte(text), 0)
	if err != nil || f == nil || len(f.Docs) == 0 {
		return ""
	}
	return DetectKubernetesGVKFromNode(f.Docs[0].Body)
}

// DetectKubernetesGVKFromNode is the per-document variant used when a
// multi-doc file may mix kinds (e.g., a Namespace alongside a Deployment).
func DetectKubernetesGVKFromNode(body ast.Node) string {
	if body == nil {
		return ""
	}
	var head struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}
	if err := yaml.NodeToValue(body, &head); err != nil {
		return ""
	}
	if head.APIVersion == "" || head.Kind == "" {
		return ""
	}
	group, version := splitOnce(head.APIVersion, '/')
	if version == "" {
		version = group
		group = ""
	}
	return KubernetesSchemaURL(group, version, head.Kind)
}

func splitOnce(s string, sep byte) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
