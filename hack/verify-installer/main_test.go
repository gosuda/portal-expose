package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerDrift(t *testing.T) {
	const crd = "kind: CustomResourceDefinition\nmetadata:\n  name: widgets.example.test\nspec:\n  scope: Namespaced\n"
	const role = "kind: ClusterRole\nmetadata:\n  name: manager-role\nrules:\n- apiGroups: [example.test]\n  resources: [widgets]\n  verbs: [get, list]\n"
	for _, test := range []struct {
		name      string
		installed string
		failure   string
	}{
		{"current", crd + "---\n" + role, ""},
		{"stale CRD", strings.Replace(crd, "Namespaced", "Cluster", 1) + "---\n" + role, "stale spec"},
		{"missing permission", crd + "---\n" + strings.Replace(role, "[get, list]", "[get]", 1), "missing a generated permission"},
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
