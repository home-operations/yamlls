package yamlast

import (
	"strings"

	"github.com/goccy/go-yaml/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type CursorContext struct {
	Pointer string
	IsKey   bool
	Doc     *ast.DocumentNode
}

// LocateCursor returns the enclosing structural context at pos. The AST
// is consulted first; when it carries no information for the cursor we
// fall back to an indent-based text walker so completion still works
// while the user is typing.
func LocateCursor(parsed *Parsed, text string, pos protocol.Position) CursorContext {
	if parsed == nil || parsed.File == nil || len(parsed.File.Docs) == 0 {
		return CursorContext{
			Pointer: inferPathByIndent(text, int(pos.Line)),
			IsKey:   isKeyLine(text, pos),
		}
	}
	offset := offsetOf(text, pos)
	doc := parsed.File.Docs[0]
	for _, d := range parsed.File.Docs {
		if d.Body == nil {
			continue
		}
		if tok := d.Body.GetToken(); tok != nil && tok.Position != nil && tok.Position.Offset <= offset {
			doc = d
		}
	}
	if doc == nil || doc.Body == nil {
		return CursorContext{Doc: doc}
	}
	ctx := CursorContext{Doc: doc}

	v := &cursorVisitor{offset: offset, best: doc.Body}
	ast.Walk(v, doc.Body)

	ctx.Pointer = pathToPointer(v.best.GetPath())
	if ctx.Pointer == "" {
		ctx.Pointer = inferPathByIndent(text, int(pos.Line))
	}
	if isKeyLine(text, pos) {
		ctx.IsKey = true
	}
	return ctx
}

type cursorVisitor struct {
	offset int
	best   ast.Node
}

func (v *cursorVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	tok := n.GetToken()
	if tok == nil || tok.Position == nil {
		return v
	}
	if tok.Position.Offset <= v.offset {
		v.best = n
	}
	return v
}

func pathToPointer(yp string) string {
	if yp == "" || yp == "$" {
		return ""
	}
	rest := strings.TrimPrefix(yp, "$")
	var b strings.Builder
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '.':
			b.WriteByte('/')
		case '[':
			b.WriteByte('/')
			for i++; i < len(rest) && rest[i] != ']'; i++ {
				b.WriteByte(rest[i])
			}
		default:
			b.WriteByte(rest[i])
		}
	}
	return b.String()
}

func offsetOf(text string, pos protocol.Position) int {
	line, col := uint32(0), uint32(0)
	for i, r := range text {
		if line == pos.Line && col == pos.Character {
			return i
		}
		if r == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return len(text)
}

func isKeyLine(text string, pos protocol.Position) bool {
	lines := strings.Split(text, "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}
	line := lines[pos.Line]
	if int(pos.Character) > len(line) {
		return !strings.Contains(line, ":")
	}
	return !strings.Contains(line[:pos.Character], ":")
}
