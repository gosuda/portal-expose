package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/component-helpers/auth/rbac/validation"
)

type manifest struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec  json.RawMessage     `json:"spec"`
	Rules []rbacv1.PolicyRule `json:"rules"`
}

func readManifests(path string) (map[string]manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)
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
		covered, missing := validation.Covers(got.Rules, want.Rules)
		if !covered {
			return fmt.Errorf("install.yaml is missing a generated permission in %s: %+v", key, missing)
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
