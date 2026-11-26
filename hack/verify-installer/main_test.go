package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerDrift(t *testing.T) {
	const crd = `kind: CustomResourceDefinition
metadata:
  name: widgets.example.test
spec:
  scope: Namespaced
`
	const roleHead = `kind: ClusterRole
metadata:
  name: manager-role
rules:
`
	const role = roleHead + `- apiGroups: [example.test]
  resources: [widgets, gadgets]
  verbs: [get, list]
`
	const splitResources = roleHead + `- apiGroups: [example.test]
  resources: [widgets]
  verbs: [get, list]
- apiGroups: [example.test]
  resources: [gadgets]
  verbs: [get, list]
`
	const splitVerbs = roleHead + `- apiGroups: [example.test]
  resources: [widgets, gadgets]
  verbs: [get]
- apiGroups: [example.test]
  resources: [widgets, gadgets]
  verbs: [list]
`
	for _, test := range []struct {
		name      string
		installed string
		failure   string
	}{
		{"current", crd + "---\n" + role, ""},
		{"split resources", crd + "---\n" + splitResources, ""},
		{"split verbs", crd + "---\n" + splitVerbs, ""},
		{"stale CRD", strings.Replace(crd, "Namespaced", "Cluster", 1) + "---\n" + role, "stale spec"},
		{
			"missing permission",
			crd + "---\n" + strings.Replace(role, "[get, list]", "[get]", 1),
			"missing a generated permission",
		},
		{
			"missing resource permission",
			crd + "---\n" + strings.Replace(role, "[widgets, gadgets]", "[widgets]", 1),
			"missing a generated permission",
		},
		{"missing resource", role, "missing generated resource"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for path, content := range map[string]string{
				"install.yaml":                 test.installed,
				"config/crd/bases/widget.yaml": crd,
				"config/rbac/role.yaml":        role,
			} {
				path = filepath.Join(root, path)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err := verify(root)
			if test.failure == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.failure) {
				t.Fatalf("want %q, got %v", test.failure, err)
			}
		})
	}
}
