package lsp

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/home-operations/yayamlls/internal/render"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const testLangID = "yaml"

func schemaModeline(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	repo := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	schema := filepath.Join(repo, "test", "fixtures", "schemas", "person.json")
	return "# yaml-language-server: $schema=" + schema + "\n"
}

type recorder struct {
	mu   sync.Mutex
	last map[string]protocol.PublishDiagnosticsParams
}

func (r *recorder) ctx() *glsp.Context {
	r.last = map[string]protocol.PublishDiagnosticsParams{}
	return &glsp.Context{Notify: func(method string, params any) {
		if method != protocol.ServerTextDocumentPublishDiagnostics {
			return
		}
		p := params.(protocol.PublishDiagnosticsParams)
		r.mu.Lock()
		r.last[p.URI] = p
		r.mu.Unlock()
	}}
}

func (r *recorder) diags(uri string) []protocol.Diagnostic {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last[uri].Diagnostics
}

// waitVersion blocks until diagnostics for uri at version v (or newer) have
// been published, since publishing is now asynchronous. Local-file schemas
// resolve in well under the timeout.
func (r *recorder) waitVersion(t *testing.T, uri string, v uint32) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		p, ok := r.last[uri]
		r.mu.Unlock()
		if ok && p.Version != nil && *p.Version >= v {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for diagnostics version %d on %s", v, uri)
}

func TestSettings_OverridesSurviveWorkspaceFolderChange(t *testing.T) {
	rec := &recorder{}
	s := New("test", render.NewRegistry())

	// A client pushes kubernetes.schemaUrl via didChangeConfiguration.
	if err := s.didChangeConfig(rec.ctx(), &protocol.DidChangeConfigurationParams{
		Settings: map[string]any{"kubernetes": map[string]any{"schemaUrl": "https://mirror/{kindLower}.json"}},
	}); err != nil {
		t.Fatalf("didChangeConfig: %v", err)
	}
	if s.settings.Kubernetes == nil || s.settings.Kubernetes.SchemaURL == "" {
		t.Fatalf("override not applied: %+v", s.settings.Kubernetes)
	}

	// Adding a workspace folder must not discard that override.
	if err := s.didChangeWorkspaceFolders(rec.ctx(), &protocol.DidChangeWorkspaceFoldersParams{
		Event: protocol.WorkspaceFoldersChangeEvent{
			Added: []protocol.WorkspaceFolder{{URI: "file:///tmp/yayamlls-test-ws"}},
		},
	}); err != nil {
		t.Fatalf("didChangeWorkspaceFolders: %v", err)
	}
	if s.settings.Kubernetes == nil || s.settings.Kubernetes.SchemaURL != "https://mirror/{kindLower}.json" {
		t.Errorf("override dropped after workspace-folder change: %+v", s.settings.Kubernetes)
	}
}

func TestDiagnostics_ReloadOnWholeChange(t *testing.T) {
	ml := schemaModeline(t)
	rec := &recorder{}
	ctx := rec.ctx()
	s := New("test", render.NewRegistry())
	uri := "file:///tmp/person.yaml"

	if err := s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, LanguageID: testLangID, Version: 1, Text: ml + "name: Alice\nage: \"thirty\"\n"},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}
	rec.waitVersion(t, uri, 1)
	if len(rec.diags(uri)) == 0 {
		t.Fatalf("expected a diagnostic for the bad age, got none")
	}

	if err := s.didChange(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument:   protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri}, Version: 2},
		ContentChanges: []any{protocol.TextDocumentContentChangeEventWhole{Text: ml + "name: Alice\nage: 30\n"}},
	}); err != nil {
		t.Fatalf("didChange: %v", err)
	}
	rec.waitVersion(t, uri, 2)
	if got := rec.diags(uri); len(got) != 0 {
		t.Fatalf("expected diagnostics cleared after fix, got %+v", got)
	}
}

func TestDiagnostics_ReloadOnIncrementalChange(t *testing.T) {
	ml := schemaModeline(t)
	rec := &recorder{}
	ctx := rec.ctx()
	s := New("test", render.NewRegistry())
	uri := "file:///tmp/person.yaml"

	// age is valid on open.
	open := ml + "name: Alice\nage: 30\n"
	if err := s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, LanguageID: testLangID, Version: 1, Text: open},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}
	rec.waitVersion(t, uri, 1)
	if got := rec.diags(uri); len(got) != 0 {
		t.Fatalf("expected no diagnostics on open, got %+v", got)
	}

	// Incrementally replace `30` with `"x"` on the age line (line index 2).
	ageLine := uint32(2)
	if err := s.didChange(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri}, Version: 2},
		ContentChanges: []any{protocol.TextDocumentContentChangeEvent{
			Range: &protocol.Range{
				Start: protocol.Position{Line: ageLine, Character: 5},
				End:   protocol.Position{Line: ageLine, Character: 7},
			},
			Text: "\"x\"",
		}},
	}); err != nil {
		t.Fatalf("didChange: %v", err)
	}
	rec.waitVersion(t, uri, 2)
	if got := rec.diags(uri); len(got) == 0 {
		t.Fatalf("expected a diagnostic after making age a string, got none")
	}
}
