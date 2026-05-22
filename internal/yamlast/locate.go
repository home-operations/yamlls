package yamlast

import (
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// JSONPointerToYAMLPath converts an RFC 6901 JSON Pointer into goccy's
// YAMLPath form (`/a/b/0` → `$.a.b[0]`).
func JSONPointerToYAMLPath(ptr string) string {
	if ptr == "" || ptr == "/" {
		return "$"
	}
	var b strings.Builder
	b.WriteByte('$')
	for _, raw := range strings.Split(strings.TrimPrefix(ptr, "/"), "/") {
		seg := unescapePointerSegment(raw)
		if isAllDigits(seg) {
			b.WriteByte('[')
			b.WriteString(seg)
			b.WriteByte(']')
			continue
		}
		b.WriteByte('.')
		b.WriteString(seg)
	}
	return b.String()
}

func unescapePointerSegment(s string) string {
	s = strings.ReplaceAll(s, "~1", "/")
	return strings.ReplaceAll(s, "~0", "~")
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// LocateRange returns the LSP range that covers the node at ptr.
// Falls back to the document body's range when ptr can't be resolved —
// common for `required` violations where the field is simply missing.
func LocateRange(doc *ast.DocumentNode, ptr string) protocol.Range {
	node, ok := lookup(doc, ptr)
	if !ok {
		node = doc.Body
	}
	return nodeRange(node)
}

func lookup(doc *ast.DocumentNode, ptr string) (ast.Node, bool) {
	if doc == nil || doc.Body == nil {
		return nil, false
	}
	if ptr == "" || ptr == "/" {
		return doc.Body, true
	}
	yp, err := yaml.PathString(JSONPointerToYAMLPath(ptr))
	if err != nil {
		return nil, false
	}
	n, err := yp.FilterNode(doc.Body)
	if err != nil || n == nil {
		return nil, false
	}
	return n, true
}

func nodeRange(n ast.Node) protocol.Range {
	if n == nil {
		return protocol.Range{}
	}
	v := &extentVisitor{}
	ast.Walk(v, n)
	if v.start == nil {
		if tok := n.GetToken(); tok != nil && tok.Position != nil {
			v.start, v.end = tok, tok
		} else {
			return protocol.Range{}
		}
	}

	startLine, startCol := tokenStart(v.start)
	endLine, endCol := tokenEnd(v.end)
	r := protocol.Range{
		Start: protocol.Position{Line: startLine, Character: startCol},
		End:   protocol.Position{Line: endLine, Character: endCol},
	}
	if r.End.Line < r.Start.Line || (r.End.Line == r.Start.Line && r.End.Character <= r.Start.Character) {
		r.End = protocol.Position{Line: startLine, Character: startCol + 1}
	}
	return r
}

func tokenStart(t *token.Token) (line, col uint32) {
	if t == nil || t.Position == nil {
		return 0, 0
	}
	return uint32(t.Position.Line - 1), uint32(t.Position.Column - 1)
}

// tokenEnd advances past the token's meaningful content. goccy's Origin
// includes the whitespace preceding the token (e.g. the space after `:`)
// but Position is anchored to the first meaningful rune, so for
// single-line tokens we measure Value; for multi-line content we walk
// the trimmed Origin to count newlines.
func tokenEnd(t *token.Token) (line, col uint32) {
	line, col = tokenStart(t)
	body := t.Value
	if body == "" || !strings.ContainsRune(body, '\n') && !strings.ContainsRune(t.Origin, '\n') {
		col += uint32(len(body))
		return line, col
	}
	body = strings.TrimLeft(t.Origin, " \t")
	body = strings.TrimRight(body, " \t\n")
	for _, r := range body {
		if r == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return line, col
}

type extentVisitor struct{ start, end *token.Token }

func (v *extentVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	if t := n.GetToken(); t != nil && t.Position != nil {
		if v.start == nil || t.Position.Offset < v.start.Position.Offset {
			v.start = t
		}
		if v.end == nil || t.Position.Offset > v.end.Position.Offset {
			v.end = t
		}
	}
	return v
}
