package schema

import "strings"

const DefaultK8sSchemaURL = "https://k8s-schemas.home-operations.com/" +
	"{groupSeg}{kindLower}_{versionLower}.json"

// BuildK8sURL renders a URL template against a GVK. Supported placeholders:
//
//	{group}         full api group, "" for core
//	{groupSeg}      "<group>/" or ""
//	{groupFirst}    first DNS label of the group
//	{groupFirstSeg} "<groupFirst>-" or ""
//	{kind}          {kindLower}
//	{version}       {versionLower}
func BuildK8sURL(template, group, version, kind string) string {
	if version == "" || kind == "" {
		return ""
	}
	if template == "" {
		template = DefaultK8sSchemaURL
	}
	groupFirst, _, _ := strings.Cut(group, ".")
	groupSeg := ""
	if group != "" {
		groupSeg = group + "/"
	}
	groupFirstSeg := ""
	if groupFirst != "" {
		groupFirstSeg = groupFirst + "-"
	}
	r := strings.NewReplacer(
		"{group}", group,
		"{groupSeg}", groupSeg,
		"{groupFirst}", groupFirst,
		"{groupFirstSeg}", groupFirstSeg,
		"{kind}", strings.ToLower(kind),
		"{kindLower}", strings.ToLower(kind),
		"{version}", strings.ToLower(version),
		"{versionLower}", strings.ToLower(version),
	)
	return r.Replace(template)
}
