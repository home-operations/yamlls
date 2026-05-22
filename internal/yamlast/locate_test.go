package yamlast

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestLocateRange_SpansMultilineMapping(t *testing.T) {
	text := `spec:
  containers:
    - name: web
      image: nginx
`
	parsed := Parse([]byte(text))
	if parsed.Err != nil {
		t.Fatalf("parse: %v", parsed.Err)
	}
	docs := parsed.Docs()
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	// /spec/containers resolves to the sequence node; its first token
	// is the `-` on line 2.
	r := LocateRange(docs[0], "/spec/containers")
	if r.Start.Line != 2 {
		t.Errorf("start line = %d, want 2 (the `-` line)", r.Start.Line)
	}
	if r.End.Line < 3 {
		t.Errorf("end line = %d, want >= 3", r.End.Line)
	}
}

func TestLocateRange_ScalarHasNonZeroWidth(t *testing.T) {
	text := "age: thirty\n"
	parsed := Parse([]byte(text))
	r := LocateRange(parsed.Docs()[0], "/age")
	want := protocol.Position{Line: 0, Character: 5 + 6}
	if r.End != want {
		t.Errorf("end = %+v, want %+v", r.End, want)
	}
}
