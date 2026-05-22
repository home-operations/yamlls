package yamlast

import "testing"

func TestInferPathByIndent_NestedMapping(t *testing.T) {
	text := `spec:
  template:
    metadata:
      labels:
        `
	got := inferPathByIndent(text, 4)
	want := "/spec/template/metadata/labels"
	if got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

func TestInferPathByIndent_SequenceItemMapping(t *testing.T) {
	text := `containers:
  - name: web
    `
	got := inferPathByIndent(text, 2)
	want := "/containers/0"
	if got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

func TestParseForCursor_RecoversFromBrokenLine(t *testing.T) {
	text := "name: Alice\nage: \"thirty"
	p := ParseForCursor(text, 1)
	if p.File == nil || len(p.File.Docs) == 0 {
		t.Fatalf("expected recovered AST, got nil docs")
	}
}
