package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"slices"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type policyRule struct {
	APIGroups       []string `json:"apiGroups"`
	Resources       []string `json:"resources"`
	ResourceNames   []string `json:"resourceNames"`
	NonResourceURLs []string `json:"nonResourceURLs"`
	Verbs           []string `json:"verbs"`
}

type manifest struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec  json.RawMessage `json:"spec"`
	Rules []policyRule    `json:"rules"`
}

func readManifests(path string) (map[string]manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := yaml.NewYAMLOrJSONDecoder(file, 4096)
	result := make(map[string]manifest)
	for {
		var item manifest
		if err := decoder.Decode(&item); errors.Is(err, io.EOF) {
			return result, nil
		} else if err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		if item.Kind == "" {
			continue
		}
		key := item.Kind + "/" + item.Metadata.Name
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("%s: duplicate resource %s", path, key)
		}
		result[key] = item
	}
}

func sameSpec(left, right json.RawMessage) (bool, error) {
	var a, b any
	if err := json.Unmarshal(left, &a); err != nil {
		return false, err
	}
	if err := json.Unmarshal(right, &b); err != nil {
		return false, err
	}
	return reflect.DeepEqual(a, b), nil
}

func ruleKey(rule policyRule) string {
	for _, values := range []*[]string{&rule.APIGroups, &rule.Resources, &rule.ResourceNames, &rule.NonResourceURLs, &rule.Verbs} {
		*values = slices.Clone(*values)
		slices.Sort(*values)
	}
	encoded, err := json.Marshal(rule)
	if err != nil {
		panic(err) // A struct containing only string slices cannot fail JSON encoding.
	}
	return string(encoded)
}

func verify(root string) error {
	installed, err := readManifests(filepath.Join(root, "install.yaml"))
	if err != nil {
		return err
	}
	crds, err := filepath.Glob(filepath.Join(root, "config/crd/bases/*.yaml"))
	if err != nil {
		return err
	}
	if len(crds) == 0 {
		return errors.New("no generated CRDs; run make manifests first")
	}
	for _, path := range crds {
		generated, err := readManifests(path)
		if err != nil {
			return err
		}
		if len(generated) == 0 {
			return fmt.Errorf("%s contains no generated CRD", path)
		}
		for key, want := range generated {
			if want.Kind != "CustomResourceDefinition" || want.Metadata.Name == "" {
				return fmt.Errorf("%s contains an invalid generated CRD: %s", path, key)
			}
			got, exists := installed[key]
			if !exists {
				return fmt.Errorf("install.yaml is missing generated resource %s", key)
			}
			equal, err := sameSpec(got.Spec, want.Spec)
			if err != nil {
				return fmt.Errorf("compare %s: %w", key, err)
			}
			if !equal {
				return fmt.Errorf("install.yaml has a stale spec for %s", key)
			}
		}
	}
	roles, err := readManifests(filepath.Join(root, "config/rbac/role.yaml"))
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		return errors.New("generated role.yaml contains no roles")
	}
	for key, want := range roles {
		if want.Kind != "ClusterRole" || want.Metadata.Name == "" || len(want.Rules) == 0 {
			return fmt.Errorf("invalid generated role: %s", key)
		}
		got, exists := installed[key]
		if !exists {
			return fmt.Errorf("install.yaml is missing generated role %s", key)
		}
		grants := make(map[string]bool, len(got.Rules))
		for _, rule := range got.Rules {
			grants[ruleKey(rule)] = true
		}
		for _, rule := range want.Rules {
			if !grants[ruleKey(rule)] {
				return fmt.Errorf("install.yaml is missing a generated permission in %s: %s", key, ruleKey(rule))
			}
		}
	}
	return nil
}

func main() {
	if err := verify("."); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Published installer matches generated CRD specs and controller permissions.")
}
