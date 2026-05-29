package diagnostics_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/home-operations/yamlls/internal/diagnostics"
	"github.com/home-operations/yamlls/internal/schema"
	"github.com/home-operations/yamlls/internal/yamlast"
)

func TestValidate_TypeMismatchProducesDiagnostic(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repo := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	docPath := filepath.Join(repo, "test", "fixtures", "person-invalid.yaml")
	body := `# yaml-language-server: $schema=./schemas/person.json
name: Alice
age: "thirty"
`
	parsed := yamlast.Parse([]byte(body))
	if parsed.Err != nil {
		t.Fatalf("parse: %v", parsed.Err)
	}

	store := schema.NewStore()
	sch, err := store.Get("./schemas/person.json", docPath)
	if err != nil {
		t.Fatalf("schema compile: %v", err)
	}

	diags := diagnostics.Validate(parsed, sch)
	if len(diags) == 0 {
		t.Fatalf("expected at least one diagnostic, got none")
	}
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "/age") {
			found = true
			if d.Range.Start.Line == 0 {
				t.Errorf("expected /age diagnostic past line 0, got %+v", d.Range)
			}
		}
	}
	if !found {
		t.Errorf("no diagnostic mentioned /age; got: %+v", diags)
	}
}

func TestValidate_ParseErrorAnchoredAtPosition(t *testing.T) {
	// `- b` under a mapping key is a syntax error goccy reports at [2:3].
	body := "a:\n- b\n  c: d\n"
	parsed := yamlast.Parse([]byte(body))
	if parsed.Err == nil {
		t.Fatalf("expected a parse error")
	}
	diags := diagnostics.Validate(parsed, nil)
	if len(diags) != 1 {
		t.Fatalf("expected one parse diagnostic, got %d: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Range.Start.Line != 1 || d.Range.Start.Character != 2 {
		t.Errorf("parse error anchored at %+v, want line 1 char 2", d.Range.Start)
	}
	if strings.Contains(d.Message, "[2:3]") || strings.Contains(d.Message, "\n") {
		t.Errorf("message should be the clean text without the position prefix or snippet, got %q", d.Message)
	}
}

func TestValidate_ValidDocProducesNoDiagnostic(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repo := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	docPath := filepath.Join(repo, "test", "fixtures", "person-valid.yaml")
	body := `# yaml-language-server: $schema=./schemas/person.json
name: Alice
age: 30
`
	parsed := yamlast.Parse([]byte(body))
	store := schema.NewStore()
	sch, err := store.Get("./schemas/person.json", docPath)
	if err != nil {
		t.Fatalf("schema compile: %v", err)
	}
	diags := diagnostics.Validate(parsed, sch)
	if len(diags) != 0 {
		t.Errorf("expected zero diagnostics, got: %+v", diags)
	}
}
