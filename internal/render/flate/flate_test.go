package flate_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/home-operations/yayamlls/internal/render"
	"github.com/home-operations/yayamlls/internal/render/flate"
)

// Run with -race: Render (pipeline goroutine) and Configure (LSP request
// goroutine) touch the same resolution state and must not race.
func TestRenderer_ConfigureDuringRenderNoRace(t *testing.T) {
	r := flate.New()
	doc := &render.SourceDocument{
		URI:      "file:///tmp/x.yaml",
		Path:     "/tmp/x.yaml",
		Kind:     "HelmRelease",
		APIGroup: "helm.toolkit.fluxcd.io/v2",
	}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = r.Render(context.Background(), doc)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				_ = r.Configure(json.RawMessage(fmt.Sprintf(`{"binary":"flate-missing-%d"}`, i%2)))
			}
		}
	}()
	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()
}

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

// writeArgStub is writeStub plus a recording of the argv to argsPath.
func writeArgStub(t *testing.T, body string) (binPath, argsPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub script uses /bin/sh")
	}
	dir := t.TempDir()
	binPath = filepath.Join(dir, "flate")
	argsPath = filepath.Join(dir, "args")
	script := "#!/bin/sh\necho \"$@\" > " + argsPath + "\ncat <<'YAML_EOF'\n" + body + "\nYAML_EOF\n"
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return binPath, argsPath
}

func TestFlate_BuildPath_ScopesByName(t *testing.T) {
	bin, argsPath := writeArgStub(t, "apiVersion: v1\nkind: Pod\nmetadata:\n  name: foo")
	r := &flate.Renderer{Binary: bin}
	if err := r.Configure(json.RawMessage(`{"path":"/repo/kubernetes"}`)); err != nil {
		t.Fatalf("configure: %v", err)
	}
	src := &render.SourceDocument{
		Path:     "/repo/kubernetes/apps/home-infra/frigate/app/hr.yaml",
		Kind:     "HelmRelease",
		APIGroup: "helm.toolkit.fluxcd.io/v2",
		Name:     "frigate",
	}
	if _, err := r.Render(context.Background(), src); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := readArgs(t, argsPath)
	want := "build hr frigate --path /repo/kubernetes -o yaml"
	if got != want {
		t.Errorf("argv = %q, want %q", got, want)
	}
}

func TestFlate_BuildPath_RelativeAnchoredAtWorkspaceRoot(t *testing.T) {
	bin, argsPath := writeArgStub(t, "apiVersion: v1\nkind: Pod\nmetadata:\n  name: foo")
	r := &flate.Renderer{Binary: bin}
	if err := r.Configure(json.RawMessage(`{"path":"kubernetes"}`)); err != nil {
		t.Fatalf("configure: %v", err)
	}
	r.SetWorkspaceRoot("/repo")
	src := &render.SourceDocument{
		Path:     "/repo/kubernetes/apps/home-infra/frigate/app/hr.yaml",
		Kind:     "HelmRelease",
		APIGroup: "helm.toolkit.fluxcd.io/v2",
		Name:     "frigate",
	}
	if _, err := r.Render(context.Background(), src); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := readArgs(t, argsPath)
	want := "build hr frigate --path " + filepath.Join("/repo", "kubernetes") + " -o yaml"
	if got != want {
		t.Errorf("argv = %q, want %q", got, want)
	}
}

func TestFlate_BuildPath_SkipsWhenNameUnknown(t *testing.T) {
	bin, argsPath := writeArgStub(t, "apiVersion: v1\nkind: Pod\nmetadata:\n  name: foo")
	r := &flate.Renderer{Binary: bin}
	if err := r.Configure(json.RawMessage(`{"path":"/repo/kubernetes"}`)); err != nil {
		t.Fatalf("configure: %v", err)
	}
	src := &render.SourceDocument{
		Path:     "/repo/kubernetes/apps/home-infra/frigate/app/hr.yaml",
		Kind:     "HelmRelease",
		APIGroup: "helm.toolkit.fluxcd.io/v2",
	}
	out, err := r.Render(context.Background(), src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(out.Manifests) != 0 {
		t.Errorf("expected no manifests when name is unknown, got %d", len(out.Manifests))
	}
	if _, err := os.Stat(argsPath); !os.IsNotExist(err) {
		t.Errorf("flate should not have been invoked; stat err = %v", err)
	}
}

func readArgs(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	return strings.TrimSpace(string(b))
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
