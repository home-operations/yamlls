package schema

import (
	"path"
	"strings"
)

// matchGlob supports `*` (no separators) and `**` (any chars). Paths must
// already be forward-slash normalized.
func matchGlob(pattern, p string) bool {
	pattern = path.Clean(strings.ReplaceAll(pattern, "\\", "/"))
	p = path.Clean(strings.ReplaceAll(p, "\\", "/"))
	return globMatch(pattern, p)
}

func globMatch(pat, s string) bool {
	for {
		if pat == "" {
			return s == ""
		}
		if strings.HasPrefix(pat, "**") {
			rest := strings.TrimPrefix(pat, "**")
			rest = strings.TrimPrefix(rest, "/")
			if rest == "" {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if globMatch(rest, s[i:]) {
					return true
				}
			}
			return false
		}
		if pat[0] == '*' {
			rest := pat[1:]
			for i := 0; i <= len(s); i++ {
				if i > 0 && s[i-1] == '/' {
					break
				}
				if globMatch(rest, s[i:]) {
					return true
				}
			}
			return false
		}
		if s == "" || pat[0] != s[0] {
			return false
		}
		pat = pat[1:]
		s = s[1:]
	}
}
