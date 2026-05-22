package yamlast

import (
	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

func Decode(doc *ast.DocumentNode) (any, error) {
	if doc == nil || doc.Body == nil {
		return nil, nil
	}
	var v any
	if err := yaml.NodeToValue(doc.Body, &v); err != nil {
		return nil, err
	}
	return v, nil
}
