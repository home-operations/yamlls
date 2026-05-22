package schema

import "testing"

func TestBuildK8sURL_DefaultsToYannh(t *testing.T) {
	got := BuildK8sURL("", "apps", "v1", "Deployment")
	want := DefaultK8sSchemaBase + "/deployment-apps-v1.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildK8sURL_TemplatePlaceholders(t *testing.T) {
	tmpl := "https://schemas.example.com/{groupSeg}{kindLower}_{versionLower}.json"
	cases := []struct {
		group, version, kind, want string
	}{
		{"helm.toolkit.fluxcd.io", "v2", "HelmRelease",
			"https://schemas.example.com/helm.toolkit.fluxcd.io/helmrelease_v2.json"},
		{"", "v1", "Pod",
			"https://schemas.example.com/pod_v1.json"},
	}
	for _, c := range cases {
		got := BuildK8sURL(tmpl, c.group, c.version, c.kind)
		if got != c.want {
			t.Errorf("group=%q kind=%q: got %q, want %q", c.group, c.kind, got, c.want)
		}
	}
}

func TestBuildK8sURL_GroupFirstLabel(t *testing.T) {
	tmpl := "https://example.com/{groupFirst}/{kindLower}-{version}.json"
	got := BuildK8sURL(tmpl, "cert-manager.io", "v1", "Certificate")
	want := "https://example.com/cert-manager/certificate-v1.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
