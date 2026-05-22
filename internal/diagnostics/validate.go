package diagnostics

import (
	"errors"
	"fmt"

	"github.com/goccy/go-yaml/ast"
	"github.com/home-operations/yamlls/internal/yamlast"
	"github.com/santhosh-tekuri/jsonschema/v5"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const Source = "yamlls"

func Validate(parsed *yamlast.Parsed, sch *jsonschema.Schema) []protocol.Diagnostic {
	if parsed == nil {
		return nil
	}
	var out []protocol.Diagnostic
	if parsed.Err != nil {
		out = append(out, parseErrorDiag(parsed.Err))
	}
	if sch == nil {
		return out
	}
	for _, doc := range parsed.Docs() {
		out = append(out, validateDoc(doc, sch)...)
	}
	return out
}

// ValidateDoc runs schema validation against a single YAML document.
// Returns nil when sch is nil or the doc validates cleanly.
func ValidateDoc(doc *ast.DocumentNode, sch *jsonschema.Schema) []protocol.Diagnostic {
	if sch == nil {
		return nil
	}
	return validateDoc(doc, sch)
}

// ParseErrorDiagnostic produces the file-level diagnostic for a YAML
// parse failure. Exposed so the LSP layer can surface it once per file
// rather than once per doc.
func ParseErrorDiagnostic(err error) protocol.Diagnostic {
	return parseErrorDiag(err)
}

func validateDoc(doc *ast.DocumentNode, sch *jsonschema.Schema) []protocol.Diagnostic {
	value, err := yamlast.Decode(doc)
	if err != nil {
		return []protocol.Diagnostic{{
			Severity: ptr(protocol.DiagnosticSeverityError),
			Source:   ptr(Source),
			Message:  fmt.Sprintf("decode: %v", err),
			Range:    yamlast.LocateRange(doc, ""),
		}}
	}
	if err := sch.Validate(value); err != nil {
		var verr *jsonschema.ValidationError
		if errors.As(err, &verr) {
			return flattenValidationError(doc, verr)
		}
		return []protocol.Diagnostic{{
			Severity: ptr(protocol.DiagnosticSeverityError),
			Source:   ptr(Source),
			Message:  err.Error(),
			Range:    yamlast.LocateRange(doc, ""),
		}}
	}
	return nil
}

// flattenValidationError emits one diagnostic per leaf cause — leaves
// carry the actionable message; root nodes are generic wrappers.
func flattenValidationError(doc *ast.DocumentNode, verr *jsonschema.ValidationError) []protocol.Diagnostic {
	var out []protocol.Diagnostic
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			out = append(out, protocol.Diagnostic{
				Severity: ptr(protocol.DiagnosticSeverityError),
				Source:   ptr(Source),
				Message:  fmt.Sprintf("%s (at %s)", e.Message, displayPointer(e.InstanceLocation)),
				Range:    yamlast.LocateRange(doc, e.InstanceLocation),
			})
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(verr)
	return out
}

func displayPointer(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func parseErrorDiag(err error) protocol.Diagnostic {
	return protocol.Diagnostic{
		Severity: ptr(protocol.DiagnosticSeverityError),
		Source:   ptr(Source),
		Message:  err.Error(),
		Range:    protocol.Range{},
	}
}

func ptr[T any](v T) *T { return &v }
