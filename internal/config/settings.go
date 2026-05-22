package config

import "encoding/json"

type Settings struct {
	Schemas    map[string][]string        `json:"schemas,omitempty"`
	Catalog    *bool                      `json:"catalog,omitempty"`
	CatalogURL string                     `json:"catalogUrl,omitempty"`
	Kubernetes *KubernetesSettings        `json:"kubernetes,omitempty"`
	Renderers  map[string]json.RawMessage `json:"renderers,omitempty"`
}

// KubernetesSettings tunes the per-document apiVersion+kind auto-detect.
type KubernetesSettings struct {
	// SchemaURL is the URL template applied to detected GVKs. Supported
	// placeholders:
	//
	//   {group}         — full api group, "" for core (e.g. "apps", "")
	//   {groupSeg}      — "<group>/" when non-empty, "" otherwise
	//   {groupFirst}    — first DNS label of the group ("apps" from "apps.k8s.io")
	//   {kind}          — kind in original case (e.g. "HelmRelease")
	//   {kindLower}     — kind lowercased (e.g. "helmrelease")
	//   {version}       — version (e.g. "v2", "v1beta1")
	//   {versionLower}  — version lowercased
	//
	// When unset, the built-in yannh/kubernetes-json-schema layout is used.
	SchemaURL string `json:"schemaUrl,omitempty"`
}

// CatalogEnabled treats nil (unset) as enabled.
func (s *Settings) CatalogEnabled() bool {
	if s == nil || s.Catalog == nil {
		return true
	}
	return *s.Catalog
}

func Parse(raw json.RawMessage) (Settings, error) {
	var s Settings
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return Settings{}, err
	}
	return s, nil
}
