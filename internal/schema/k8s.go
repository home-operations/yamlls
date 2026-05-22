package schema

import (
	"fmt"
	"strings"
)

const DefaultK8sSchemaBase = "https://raw.githubusercontent.com/yannh/" +
	"kubernetes-json-schema/master/master-standalone-strict"

// KubernetesSchemaURL builds the yannh/kubernetes-json-schema path for
// a GVK: `<kind>-<version>.json` for core resources, otherwise
// `<kind>-<group-first-label>-<version>.json`.
func KubernetesSchemaURL(group, version, kind string) string {
	if version == "" || kind == "" {
		return ""
	}
	kind = strings.ToLower(kind)
	if group == "" {
		return fmt.Sprintf("%s/%s-%s.json", DefaultK8sSchemaBase, kind, version)
	}
	short, _, _ := strings.Cut(group, ".")
	return fmt.Sprintf("%s/%s-%s-%s.json", DefaultK8sSchemaBase, kind, short, version)
}
