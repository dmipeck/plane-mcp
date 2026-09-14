package plane_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"gopkg.in/yaml.v2"
)

// Six path templates required by the v0.1 Tool inventory (spectacular naming).
var v01Paths = []string{
	"/api/v1/workspaces/{slug}/projects/",
	"/api/v1/workspaces/{slug}/projects/{pk}/",
	"/api/v1/workspaces/{slug}/projects/{project_id}/work-items/",
	"/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{pk}/",
	"/api/v1/workspaces/{slug}/work-items/{project_identifier}-{issue_identifier}/",
	"/api/v1/workspaces/{slug}/projects/{project_id}/states/",
}

// Matches .ogen.yml path_regex — kept here so drift between filter and SoT fails tests.
var ogenPathRegex = regexp.MustCompile(
	`^/api/v1/workspaces/\{[^/]+\}/(?:projects/?$|projects/\{[^/]+\}/?$|projects/\{[^/]+\}/work-items/?$|projects/\{[^/]+\}/work-items/\{[^/]+\}/?$|projects/\{[^/]+\}/states/?$|work-items/\{[^/]+\}/?$)`,
)

func openAPIPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "openapi.yaml")
}

func TestOpenAPI_LivesBesideOgenCleanTarget(t *testing.T) {
	t.Parallel()

	yamlPath := openAPIPath(t)
	if _, err := os.Stat(yamlPath); err != nil {
		t.Fatalf("vendored OpenAPI missing at %s: %v", yamlPath, err)
	}

	oasSibling := filepath.Join(filepath.Dir(yamlPath), "..", "oas", "openapi.yaml")
	if _, err := os.Stat(oasSibling); !os.IsNotExist(err) {
		t.Fatalf("OpenAPI must not live inside ogen --clean target; found %s (err=%v)", oasSibling, err)
	}
}

func TestOpenAPI_ContainsExactlySixV01PathsUnderFilter(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(openAPIPath(t))
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}

	var doc struct {
		Paths map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}

	for _, p := range v01Paths {
		if _, ok := doc.Paths[p]; !ok {
			t.Errorf("missing v0.1 path %s", p)
		}
		if !ogenPathRegex.MatchString(p) {
			t.Errorf("path %s does not match .ogen.yml path_regex", p)
		}
	}

	var matched []string
	for p := range doc.Paths {
		if ogenPathRegex.MatchString(p) {
			matched = append(matched, p)
		}
	}
	if len(matched) != len(v01Paths) {
		t.Fatalf("path_regex matches %d paths, want %d: %v", len(matched), len(v01Paths), matched)
	}

	// Deprecated /issues/ twins and nested resources must not match.
	banned := []string{
		"/api/v1/workspaces/{slug}/projects/{project_id}/issues/",
		"/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/comments/",
		"/api/v1/workspaces/{slug}/projects/{project_id}/states/{state_id}/",
	}
	for _, p := range banned {
		if ogenPathRegex.MatchString(p) {
			t.Errorf("path_regex incorrectly includes %s", p)
		}
	}
}
