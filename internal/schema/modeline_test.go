package schema

import "testing"

func TestFindModelineSchema(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "leading modeline",
			in:   "# yaml-language-server: $schema=./foo.json\nname: x\n",
			want: "./foo.json",
		},
		{
			name: "modeline after blank lines",
			in:   "\n\n# yaml-language-server: $schema=https://example.com/s.json\nfoo: bar\n",
			want: "https://example.com/s.json",
		},
		{
			name: "no modeline",
			in:   "name: x\nage: 30\n",
			want: "",
		},
		{
			name: "modeline inside body is ignored",
			in:   "name: x\n# yaml-language-server: $schema=./late.json\nage: 30\n",
			want: "",
		},
		{
			name: "extra trailing kvs tolerated",
			in:   "# yaml-language-server: $schema=./foo.json someOther=value\n",
			want: "./foo.json",
		},
		{
			name: "modeline after doc separator",
			in:   "---\n# yaml-language-server: $schema=./sep.json\nname: x\n",
			want: "./sep.json",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FindModelineSchema(c.in)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
