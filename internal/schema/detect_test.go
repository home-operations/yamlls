package schema

import "testing"

func TestDetectKubernetesGVK(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{
			name: "core pod",
			in:   "apiVersion: v1\nkind: Pod\nmetadata:\n  name: x\n",
			want: DefaultK8sSchemaBase + "/pod-v1.json",
		},
		{
			name: "grouped deployment",
			in:   "apiVersion: apps/v1\nkind: Deployment\n",
			want: DefaultK8sSchemaBase + "/deployment-apps-v1.json",
		},
		{
			name: "missing kind",
			in:   "apiVersion: v1\nname: noisy\n",
			want: "",
		},
		{
			name: "non-k8s doc",
			in:   "name: Alice\nage: 30\n",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DetectKubernetesGVK(c.in); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
