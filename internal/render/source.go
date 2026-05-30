package render

import (
	"context"

	"github.com/goccy/go-yaml/ast"
)

type SourceDocument struct {
	URI      string
	Path     string
	Text     string
	AST      *ast.File
	Kind     string
	APIGroup string
	Name     string
}

type RenderedManifest struct {
	AST  *ast.DocumentNode
	GVK  GVK
	Name string
}

// GVK identifies a Kubernetes type. Empty Group denotes the core API.
type GVK struct {
	Group   string
	Version string
	Kind    string
}

type RenderedOutput struct {
	Provider  string
	Manifests []RenderedManifest
	Raw       []byte
	Stderr    []byte
}

type Renderer interface {
	Name() string
	Matches(doc *SourceDocument) bool
	Render(ctx context.Context, doc *SourceDocument) (*RenderedOutput, error)
}
