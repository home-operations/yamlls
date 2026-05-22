package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".yamlls.yaml")
	body := `schemas:
  "./schemas/local.json":
    - "k8s/**/*.yaml"
catalog: false
renderers:
  flate:
    enabled: false
    binary: /opt/bin/flate
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if want := []string{"k8s/**/*.yaml"}; len(got.Schemas) != 1 || got.Schemas["./schemas/local.json"][0] != want[0] {
		t.Errorf("schemas: %+v", got.Schemas)
	}
	if got.Catalog == nil || *got.Catalog != false {
		t.Errorf("catalog: %+v", got.Catalog)
	}
	if _, ok := got.Renderers["flate"]; !ok {
		t.Errorf("renderers missing flate: %+v", got.Renderers)
	}
}

func TestLoadFile_AbsentReturnsZero(t *testing.T) {
	got, err := LoadFile(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Errorf("absent file should not error: %v", err)
	}
	if len(got.Schemas) != 0 || len(got.Renderers) != 0 {
		t.Errorf("expected zero-value, got %+v", got)
	}
}

func TestMerge_OverrideWins(t *testing.T) {
	base := Settings{
		Schemas:   map[string][]string{"a.json": {"a"}},
		Catalog:   ptrBool(true),
		Renderers: map[string]json.RawMessage{"flate": json.RawMessage(`{"binary":"x"}`)},
	}
	overrideTrue := false
	override := Settings{
		Schemas:   map[string][]string{"b.json": {"b"}},
		Catalog:   &overrideTrue,
		Renderers: map[string]json.RawMessage{"flate": json.RawMessage(`{"binary":"y"}`)},
	}
	got := Merge(base, override)
	if got.Schemas["a.json"][0] != "a" || got.Schemas["b.json"][0] != "b" {
		t.Errorf("schemas merge wrong: %+v", got.Schemas)
	}
	if *got.Catalog != false {
		t.Errorf("override catalog should win: %v", *got.Catalog)
	}
	if string(got.Renderers["flate"]) != `{"binary":"y"}` {
		t.Errorf("override renderer should win: %s", got.Renderers["flate"])
	}
}

func ptrBool(b bool) *bool { return &b }
