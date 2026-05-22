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

// BuildK8sURL renders a user-supplied URL template against a GVK. When
// template is empty, falls back to KubernetesSchemaURL (yannh layout).
//
// Placeholders are documented on config.KubernetesSettings.
func BuildK8sURL(template, group, version, kind string) string {
	if version == "" || kind == "" {
		return ""
	}
	if template == "" {
		return KubernetesSchemaURL(group, version, kind)
	}
	groupFirst, _, _ := strings.Cut(group, ".")
	groupSeg := ""
	if group != "" {
		groupSeg = group + "/"
	}
	r := strings.NewReplacer(
		"{group}", group,
		"{groupSeg}", groupSeg,
		"{groupFirst}", groupFirst,
		"{kind}", kind,
		"{kindLower}", strings.ToLower(kind),
		"{version}", version,
		"{versionLower}", strings.ToLower(version),
	)
	return r.Replace(template)
}
