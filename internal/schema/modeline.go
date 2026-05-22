package schema

import (
	"bufio"
	"strings"
)

const modelinePrefix = "# yaml-language-server:"

// FindModelineSchema scans the leading comment block for
// `# yaml-language-server: $schema=…` and returns the URL or path.
// Stops at the first non-comment line so the directive can't be smuggled
// in via a string value.
func FindModelineSchema(text string) string {
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			return ""
		}
		if !strings.HasPrefix(line, modelinePrefix) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, modelinePrefix))
		for _, kv := range strings.Fields(rest) {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				continue
			}
			if strings.TrimSpace(k) == "$schema" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}
