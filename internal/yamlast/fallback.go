package yamlast

import (
	"strconv"
	"strings"
)

// inferPathByIndent computes the JSON Pointer of the cursor's enclosing
// mapping using a text-only indent scan. Used when the AST can't locate
// the cursor (parse failures, trailing whitespace). Does not understand
// block scalars or flow-style mappings.
func inferPathByIndent(text string, cursorLine int) string {
	lines := strings.Split(text, "\n")
	if cursorLine >= len(lines) {
		cursorLine = len(lines) - 1
	}
	if cursorLine < 0 {
		return ""
	}

	type frame struct {
		indent int
		key    string
		seqIdx int
		isSeq  bool
	}
	var stack []frame

	pushKey := func(indent int, key string) {
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, frame{indent: indent, key: key})
	}
	pushSeqItem := func(indent int) {
		for len(stack) > 0 && stack[len(stack)-1].indent > indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 && stack[len(stack)-1].indent == indent && stack[len(stack)-1].isSeq {
			stack[len(stack)-1].seqIdx++
			return
		}
		stack = append(stack, frame{indent: indent, isSeq: true})
	}

	indentOf := func(s string) int {
		for i, r := range s {
			if r != ' ' && r != '\t' {
				return i
			}
		}
		return len(s)
	}

	for i := 0; i < cursorLine; i++ {
		ln := lines[i]
		trimmed := strings.TrimSpace(ln)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := indentOf(ln)
		switch {
		case strings.HasPrefix(trimmed, "- "):
			pushSeqItem(indent)
			rest := strings.TrimPrefix(trimmed, "- ")
			if k, _, ok := strings.Cut(rest, ":"); ok && !strings.ContainsAny(k, " \t") {
				pushKey(indent+2, strings.TrimSpace(k))
			}
		case trimmed == "-":
			pushSeqItem(indent)
		default:
			if k, _, ok := strings.Cut(trimmed, ":"); ok && !strings.ContainsAny(k, " \t") {
				pushKey(indent, strings.TrimSpace(k))
			}
		}
	}

	cursorIndent := indentOf(lines[cursorLine])
	for len(stack) > 0 && stack[len(stack)-1].indent >= cursorIndent {
		stack = stack[:len(stack)-1]
	}

	if len(stack) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range stack {
		b.WriteByte('/')
		if f.isSeq {
			b.WriteString(strconv.Itoa(f.seqIdx))
		} else {
			b.WriteString(escapePointerSegment(f.key))
		}
	}
	return b.String()
}

func escapePointerSegment(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	return strings.ReplaceAll(s, "/", "~1")
}
