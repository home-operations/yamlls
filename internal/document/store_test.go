package document

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// "😀" is a single astral-plane rune that occupies two UTF-16 code units.
// Clients address columns after it in UTF-16 units, so an offset computed
// by counting runes would land one byte short and corrupt the edit.
func TestApplyRangeChange_AstralColumnsAreUTF16(t *testing.T) {
	text := "a: 😀x\n"
	c := protocol.TextDocumentContentChangeEvent{
		Range: &protocol.Range{
			Start: protocol.Position{Line: 0, Character: 5},
			End:   protocol.Position{Line: 0, Character: 6},
		},
		Text: "y",
	}
	got := applyRangeChange(text, c)
	if want := "a: 😀y\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
