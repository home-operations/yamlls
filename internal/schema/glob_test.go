package schema

import "testing"

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pat, in string
		want    bool
	}{
		{"**/*.yaml", "a/b/c.yaml", true},
		{"**/*.yaml", "c.yaml", true},
		{"**/*.yaml", "a/b/c.json", false},
		{"k8s/**/*.yaml", "k8s/dev/app.yaml", true},
		{"k8s/**/*.yaml", "other/k8s/app.yaml", false},
		{"*.yaml", "a.yaml", true},
		{"*.yaml", "sub/a.yaml", false},
		{"foo/*/bar.yaml", "foo/x/bar.yaml", true},
		{"foo/*/bar.yaml", "foo/x/y/bar.yaml", false},
		{"**", "anything/at/all.txt", true},
	}
	for _, c := range cases {
		if got := matchGlob(c.pat, c.in); got != c.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", c.pat, c.in, got, c.want)
		}
	}
}
