package schema

import (
	"testing"

	"github.com/home-operations/yamlls/internal/config"
)

func TestResolver_ModelineBeatsSettings(t *testing.T) {
	r := NewResolver()
	r.SetSettings(config.Settings{
		Schemas: map[string][]string{"./glob.json": {"**/*.yaml"}},
		Catalog: ptrBool(false),
	})

	got := r.Resolve("# yaml-language-server: $schema=./modeline.json\nx: 1\n", "/tmp/a.yaml")
	if got != "./modeline.json" {
		t.Errorf("modeline lost to settings: %q", got)
	}
}

func TestResolver_SettingsGlobMatch(t *testing.T) {
	r := NewResolver()
	r.SetSettings(config.Settings{
		Schemas: map[string][]string{"./k8s.json": {"k8s/**/*.yaml"}},
		Catalog: ptrBool(false),
	})

	got := r.Resolve("apiVersion: v1\n", "/repo/k8s/dev/app.yaml")
	if got != "./k8s.json" {
		t.Errorf("expected ./k8s.json, got %q", got)
	}

	if r.Resolve("x: 1\n", "/repo/other/file.yaml") != "" {
		t.Errorf("expected empty for non-matching path")
	}
}

func ptrBool(b bool) *bool { return &b }
