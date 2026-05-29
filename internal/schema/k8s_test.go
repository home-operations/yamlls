package schema

import (
	"strings"
	"testing"
)

func TestBuildK8sURL_DefaultPointsAtHomeOperations(t *testing.T) {
	got := BuildK8sURL("", "apps", "v1", "Deployment")
	want := "https://k8s-schemas.home-operations.com/apps/deployment_v1.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildK8sURL_DefaultCoreResource(t *testing.T) {
	got := BuildK8sURL("", "", "v1", "Pod")
	want := "https://k8s-schemas.home-operations.com/pod_v1.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildK8sURL_TemplatePlaceholders(t *testing.T) {
	tmpl := "https://schemas.example.com/{groupSeg}{kindLower}_{versionLower}.json"
	got := BuildK8sURL(tmpl, "helm.toolkit.fluxcd.io", "v2", "HelmRelease")
	want := "https://schemas.example.com/helm.toolkit.fluxcd.io/helmrelease_v2.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildK8sURL_YannhLayoutViaTemplate(t *testing.T) {
	tmpl := "https://yannh.example/{kindLower}-{groupFirstSeg}{version}.json"
	if got := BuildK8sURL(tmpl, "", "v1", "Pod"); got != "https://yannh.example/pod-v1.json" {
		t.Errorf("core: got %q", got)
	}
	if got := BuildK8sURL(tmpl, "apps", "v1", "Deployment"); got != "https://yannh.example/deployment-apps-v1.json" {
		t.Errorf("grouped: got %q", got)
	}
}

func TestBuildK8sURL_EmptyWhenMissingFields(t *testing.T) {
	if got := BuildK8sURL("", "apps", "", "Deployment"); got != "" {
		t.Errorf("missing version should yield empty, got %q", got)
	}
	if got := BuildK8sURL("", "apps", "v1", ""); got != "" {
		t.Errorf("missing kind should yield empty, got %q", got)
	}
}

func TestBuildK8sURL_KindAndVersionLowercased(t *testing.T) {
	// The documented contract is {kind} == {kindLower} and {version} ==
	// {versionLower}; both must be lowercased.
	got := BuildK8sURL("{kind}_{version}", "apps", "V1Beta1", "Deployment")
	if got != "deployment_v1beta1" {
		t.Errorf("got %q, want deployment_v1beta1", got)
	}
}

func TestBuildK8sURL_AllPlaceholdersResolve(t *testing.T) {
	tmpl := "{group}|{groupSeg}|{groupFirst}|{groupFirstSeg}|{kind}|{kindLower}|{version}|{versionLower}"
	got := BuildK8sURL(tmpl, "apps.k8s.io", "v1beta1", "Deployment")
	if strings.Contains(got, "{") || strings.Contains(got, "}") {
		t.Errorf("unsubstituted placeholder leaked: %q", got)
	}
}
