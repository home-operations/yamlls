package flate_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/home-operations/yamlls/internal/render"
	"github.com/home-operations/yamlls/internal/render/flate"
)

func writeStub(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub script uses /bin/sh")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "flate")
	script := "#!/bin/sh\ncat <<'YAML_EOF'\n" + body + "\nYAML_EOF\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFlate_RenderHelmRelease(t *testing.T) {
	rendered := `apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
    - name: c
      image: nginx:latest
---
apiVersion: v1
kind: Service
metadata:
  name: bar
spec:
  ports:
    - port: 80`
	stub := writeStub(t, rendered)

	r := &flate.Renderer{Binary: stub}
	src := &render.SourceDocument{
		URI:      "file:///tmp/hr.yaml",
		Path:     "/tmp/hr.yaml",
		Kind:     "HelmRelease",
		APIGroup: "helm.toolkit.fluxcd.io/v2beta2",
	}

	out, err := r.Render(context.Background(), src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(out.Manifests) != 2 {
		t.Fatalf("expected 2 manifests, got %d", len(out.Manifests))
	}
	gotKinds := []string{out.Manifests[0].GVK.Kind, out.Manifests[1].GVK.Kind}
	wantKinds := []string{"Pod", "Service"}
	for i := range wantKinds {
		if gotKinds[i] != wantKinds[i] {
			t.Errorf("manifests[%d].GVK.Kind = %s, want %s", i, gotKinds[i], wantKinds[i])
		}
	}
	if !strings.Contains(string(out.Raw), "Service") {
		t.Errorf("Raw missing Service doc: %s", out.Raw)
	}
}

func TestFlate_MissingBinaryGivesActionableError(t *testing.T) {
	r := &flate.Renderer{Binary: "/no/such/path/flate-does-not-exist"}
	src := &render.SourceDocument{
		Path:     "/tmp/hr.yaml",
		Kind:     "HelmRelease",
		APIGroup: "helm.toolkit.fluxcd.io/v2",
	}
	_, err := r.Render(context.Background(), src)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "install with `go install") {
		t.Errorf("error message not actionable: %v", err)
	}
}

func TestFlate_MatchesKindsExactly(t *testing.T) {
	r := flate.New()
	cases := []struct {
		name string
		doc  *render.SourceDocument
		want bool
	}{
		{"helm release v2", &render.SourceDocument{Kind: "HelmRelease", APIGroup: "helm.toolkit.fluxcd.io/v2"}, true},
		{"kustomization", &render.SourceDocument{Kind: "Kustomization", APIGroup: "kustomize.toolkit.fluxcd.io/v1"}, true},
		{"vanilla pod", &render.SourceDocument{Kind: "Pod", APIGroup: "v1"}, false},
		{"unrelated CR", &render.SourceDocument{Kind: "Other", APIGroup: "example.com/v1"}, false},
	}
	for _, c := range cases {
		if got := r.Matches(c.doc); got != c.want {
			t.Errorf("%s: Matches = %v, want %v", c.name, got, c.want)
		}
	}
}
