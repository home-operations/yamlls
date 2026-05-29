package document

import (
	"fmt"
	"sync"
	"unicode/utf16"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Document struct {
	URI        string
	LanguageID string
	Version    int32
	Text       string
}

type Store struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

func NewStore() *Store {
	return &Store{docs: make(map[string]*Document)}
}

func (s *Store) Open(uri, langID string, version int32, text string) *Document {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := &Document{URI: uri, LanguageID: langID, Version: version, Text: text}
	s.docs[uri] = d
	return d
}

func (s *Store) Close(uri string) {
	s.mu.Lock()
	delete(s.docs, uri)
	s.mu.Unlock()
}

func (s *Store) Get(uri string) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[uri]
	return d, ok
}

func (s *Store) Apply(uri string, version int32, changes []any) (*Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.docs[uri]
	if !ok {
		return nil, fmt.Errorf("document not open: %s", uri)
	}
	// Build a fresh Document rather than mutating in place: the previous
	// pointer may still be read by the async render goroutine, so an
	// in-place write to Text would be a data race.
	text := cur.Text
	for _, raw := range changes {
		switch c := raw.(type) {
		case protocol.TextDocumentContentChangeEvent:
			text = applyRangeChange(text, c)
		case protocol.TextDocumentContentChangeEventWhole:
			text = c.Text
		default:
			return nil, fmt.Errorf("unsupported change event type %T", raw)
		}
	}
	d := &Document{URI: uri, LanguageID: cur.LanguageID, Version: version, Text: text}
	s.docs[uri] = d
	return d, nil
}

func applyRangeChange(text string, c protocol.TextDocumentContentChangeEvent) string {
	start := offsetAt(text, c.Range.Start)
	end := offsetAt(text, c.Range.End)
	if start > end {
		start, end = end, start
	}
	return text[:start] + c.Text + text[end:]
}

func offsetAt(text string, pos protocol.Position) int {
	line, col := uint32(0), uint32(0)
	for i, r := range text {
		if line == pos.Line {
			if col >= pos.Character {
				return i
			}
			if r == '\n' {
				return i // column past end of line: clamp to line end
			}
		}
		if r == '\n' {
			line++
			col = 0
			continue
		}
		// LSP character offsets count UTF-16 code units; a rune outside
		// the BMP occupies two. Counting runes here would misplace edits
		// in documents containing such characters.
		n := utf16.RuneLen(r)
		if n < 0 {
			n = 1
		}
		col += uint32(n)
	}
	return len(text)
}

func (s *Store) AllURIs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.docs))
	for k := range s.docs {
		out = append(out, k)
	}
	return out
}
