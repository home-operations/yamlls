package yamlast

import "testing"

func TestPathToPointer(t *testing.T) {
	cases := map[string]string{
		"$":            "",
		"$.a.b":        "/a/b",
		"$.items[0].x": "/items/0/x",
		"$.'a.b'.c":    "/a.b/c", // quoted key keeps its dot as one segment
		"$.a/b":        "/a~1b",  // slash in a key is JSON-Pointer escaped
		"$.'x/y'":      "/x~1y",  // quoted key containing a slash
	}
	for in, want := range cases {
		if got := pathToPointer(in); got != want {
			t.Errorf("pathToPointer(%q) = %q, want %q", in, got, want)
		}
	}
}
