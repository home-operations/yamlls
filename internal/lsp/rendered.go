package lsp

import (
	"errors"
	"fmt"

	"github.com/home-operations/yamlls/internal/diagnostics"
	"github.com/home-operations/yamlls/internal/render"
	"github.com/home-operations/yamlls/internal/schema"
	"github.com/home-operations/yamlls/internal/yamlast"
	"github.com/santhosh-tekuri/jsonschema/v5"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// renderDiagnostics emits file-level diagnostics for a rendered output:
// rendered manifests have no source line, so we embed the rendered
// location (kind/name @ jsonptr) in the message instead.
func renderDiagnostics(store *schema.Store, out *render.RenderedOutput, err error) []protocol.Diagnostic {
	if err != nil {
		return []protocol.Diagnostic{{
			Severity: ptr(protocol.DiagnosticSeverityError),
			Source:   ptr(renderSource(out)),
			Message:  "render: " + err.Error(),
			Range:    protocol.Range{},
		}}
	}
	if out == nil {
		return nil
	}
	var diags []protocol.Diagnostic
	for _, m := range out.Manifests {
		url := schema.KubernetesSchemaURL(m.GVK.Group, m.GVK.Version, m.GVK.Kind)
		if url == "" {
			continue
		}
		sch, err := store.Get(url, "")
		if err != nil {
			diags = append(diags, protocol.Diagnostic{
				Severity: ptr(protocol.DiagnosticSeverityWarning),
				Source:   ptr(renderSource(out)),
				Message:  fmt.Sprintf("[rendered %s/%s] could not load Kubernetes schema: %v", m.GVK.Kind, m.Name, err),
			})
			continue
		}
		value, err := yamlast.Decode(m.AST)
		if err != nil {
			diags = append(diags, protocol.Diagnostic{
				Severity: ptr(protocol.DiagnosticSeverityWarning),
				Source:   ptr(renderSource(out)),
				Message:  fmt.Sprintf("[rendered %s/%s] decode failed: %v", m.GVK.Kind, m.Name, err),
			})
			continue
		}
		if err := sch.Validate(value); err != nil {
			var verr *jsonschema.ValidationError
			if errors.As(err, &verr) {
				diags = append(diags, flattenRendered(out, m, verr)...)
			} else {
				diags = append(diags, protocol.Diagnostic{
					Severity: ptr(protocol.DiagnosticSeverityError),
					Source:   ptr(renderSource(out)),
					Message:  fmt.Sprintf("[rendered %s/%s] %s", m.GVK.Kind, m.Name, err.Error()),
				})
			}
		}
	}
	return diags
}

func flattenRendered(out *render.RenderedOutput, m render.RenderedManifest, verr *jsonschema.ValidationError) []protocol.Diagnostic {
	var diags []protocol.Diagnostic
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			loc := e.InstanceLocation
			if loc == "" {
				loc = "/"
			}
			diags = append(diags, protocol.Diagnostic{
				Severity: ptr(protocol.DiagnosticSeverityError),
				Source:   ptr(renderSource(out)),
				Message:  fmt.Sprintf("[rendered %s/%s @ %s] %s", m.GVK.Kind, m.Name, loc, e.Message),
			})
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(verr)
	return diags
}

func renderSource(out *render.RenderedOutput) string {
	if out == nil || out.Provider == "" {
		return diagnostics.Source + "/render"
	}
	return diagnostics.Source + "/" + out.Provider
}
