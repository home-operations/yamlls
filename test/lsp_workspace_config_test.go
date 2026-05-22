package test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceConfigFileFlowsThrough(t *testing.T) {
	bin := buildBinary(t)

	root := t.TempDir()
	schemaDir := filepath.Join(root, "schemas")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	schemaJSON := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["count"],
  "properties": { "count": { "type": "integer" } },
  "additionalProperties": false
}`
	if err := os.WriteFile(filepath.Join(schemaDir, "thing.json"), []byte(schemaJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := `schemas:
  "./schemas/thing.json":
    - "data/**/*.yaml"
catalog: false
`
	if err := os.WriteFile(filepath.Join(root, ".yamlls.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(dataDir, "thing.yaml")
	if err := os.WriteFile(docPath, []byte("count: \"nope\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stdin.Close(); _ = cmd.Wait() })

	conn := &rpcConn{w: stdin, r: bufio.NewReader(stdout)}
	rootURI := "file://" + root
	if _, err := conn.send("initialize", map[string]any{
		"processId":    nil,
		"rootUri":      rootURI,
		"capabilities": map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.readFrame(); err != nil {
		t.Fatalf("init: %v (stderr=%s)", err, stderr.String())
	}
	_ = conn.notify("initialized", map[string]any{})

	docURI := "file://" + docPath
	_ = conn.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        docURI,
			"languageId": "yaml",
			"version":    1,
			"text":       "count: \"nope\"\n",
		},
	})

	frame, err := readUntilDiagnostics(conn, 5*time.Second)
	if err != nil {
		t.Fatalf("%v (stderr=%s)", err, stderr.String())
	}
	params, _ := frame["params"].(map[string]any)
	diags, _ := params["diagnostics"].([]any)
	if len(diags) == 0 {
		t.Fatalf("expected a diagnostic from .yamlls.yaml schema mapping, got none")
	}
	combined, _ := json.Marshal(diags)
	if !strings.Contains(string(combined), "/count") {
		t.Errorf("no diagnostic mentioned /count; got: %s", combined)
	}
}
