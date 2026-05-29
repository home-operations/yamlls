// Package lint validates YAML documents against their resolved schemas,
// independent of the LSP transport. Both the language server and the
// `yamlls validate` command share Document so they report identically.
package lint

import (
	"github.com/home-operations/yamlls/internal/diagnostics"
	"github.com/home-operations/yamlls/internal/schema"
	"github.com/home-operations/yamlls/internal/yamlast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Document validates a single file's text. path is the on-disk path used
// for relative schema resolution. It returns the parse error (if any),
// per-document schema violations, and one schema-load failure per
// user-intended ref. yamlls-disable suppressions are NOT applied here —
// callers filter, so the LSP server can suppress rendered diagnostics in
// the same pass.
func Document(text, path string, resolver *schema.Resolver, store *schema.Store) []protocol.Diagnostic {
	parsed := yamlast.Parse([]byte(text))
	fileRef := resolver.Resolve(text, path)

	diags := []protocol.Diagnostic{}
	if parsed.Err != nil {
		diags = append(diags, diagnostics.ParseErrorDiagnostic(parsed.Err))
	}

	// Surface load failures only for file-level refs; per-doc auto-detect
	// would warn on every CRD missing from the configured mirror.
	loadFailures := make(map[string]bool)
	for _, doc := range parsed.Docs() {
		ref := schema.FindModelineSchemaForDoc(doc)
		if ref == "" {
			ref = fileRef
		}
		userIntent := ref != ""
		if ref == "" {
			ref = resolver.K8sURLForNode(doc.Body)
		}
		if ref == "" {
			continue
		}
		sch, err := store.Get(ref, path)
		if err != nil {
			if userIntent && !loadFailures[ref] {
				loadFailures[ref] = true
				diags = append(diags, SchemaLoadDiagnostic(err))
			}
			continue
		}
		diags = append(diags, diagnostics.ValidateDoc(doc, sch, text)...)
	}
	return diags
}

// SchemaLoadDiagnostic is the file-level warning emitted when a
// user-intended schema ref fails to load.
func SchemaLoadDiagnostic(err error) protocol.Diagnostic {
	sev := protocol.DiagnosticSeverityWarning
	src := diagnostics.Source
	return protocol.Diagnostic{
		Severity: &sev,
		Source:   &src,
		Message:  "schema load failed: " + err.Error(),
		Range:    protocol.Range{},
	}
}
