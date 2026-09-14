package mcp_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	planemcp "github.com/dmipeck/plane-mcp/internal/mcp"
)

const crudProjectID = "11111111-1111-1111-1111-111111111111"

const oneProjectDetailJSON = `{
  "id": "11111111-1111-1111-1111-111111111111",
  "name": "Demo",
  "identifier": "DEMO"
}`

func callProjectTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      name,
		Arguments: json.RawMessage(raw),
	})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return res
}

func TestProjectView_IsRegistered(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tools, err := connectMCP(t, srv).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "project_view" {
			return
		}
	}
	t.Fatal("project_view not registered")
}

func TestProjectView_RequiresWorkspace(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_view", map[string]any{
		"project_id": crudProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "workspace") {
		t.Fatalf("error %q should mention workspace", text)
	}
}

func TestProjectView_RequiresProjectID(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_view", map[string]any{
		"workspace": "acme",
	}))
	if !strings.Contains(strings.ToLower(text), "project_id") {
		t.Fatalf("error %q should mention project_id", text)
	}
}

func TestProjectView_SendsAuthPathAndMethod(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{BaseURL: "https://plane.example.com", APIKey: "pat-secret"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, oneProjectDetailJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectTool(t, connectMCP(t, srv), "project_view", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	if got == nil {
		t.Fatal("Plane HTTP request not captured")
	}
	if got.Method != http.MethodGet {
		t.Fatalf("method = %q, want GET", got.Method)
	}
	wantPath := "/api/v1/workspaces/acme/projects/" + crudProjectID + "/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %s", got.URL.Path, wantPath)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}
}

func TestProjectView_HappyPathReturnsProject(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(200, oneProjectDetailJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectTool(t, connectMCP(t, srv), "project_view", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal StructuredContent: %v", err)
	}
	if !strings.Contains(string(raw), `"identifier":"DEMO"`) && !strings.Contains(string(raw), `"identifier": "DEMO"`) {
		t.Fatalf("StructuredContent missing DEMO project: %s", raw)
	}
}

func TestProjectView_MapsUnauthorized(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "bad-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(401, `{}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_view", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}

func TestProjectView_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(404, `{}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_view", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestProjectView_MapsForbidden(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(403, `{}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_view", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "forbidden") {
		t.Fatalf("error %q should describe forbidden", text)
	}
}

func TestProjectCreate_RequiresWorkspaceNameIdentifier(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	session := connectMCP(t, srv)

	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"missing workspace", map[string]any{"name": "Demo", "identifier": "DEMO"}, "workspace"},
		{"missing name", map[string]any{"workspace": "acme", "identifier": "DEMO"}, "name"},
		{"missing identifier", map[string]any{"workspace": "acme", "name": "Demo"}, "identifier"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			text := toolErrorText(t, callProjectTool(t, session, "project_create", tc.args))
			if !strings.Contains(strings.ToLower(text), tc.want) {
				t.Fatalf("error %q should mention %s", text, tc.want)
			}
		})
	}
}

func TestProjectCreate_SendsAuthPathBodyAndMethod(t *testing.T) {
	t.Parallel()

	var got *http.Request
	var body []byte
	srv, err := planemcp.New(
		planemcp.Connection{BaseURL: "https://plane.example.com", APIKey: "pat-secret"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			body, _ = io.ReadAll(r.Body)
			return jsonResponse(201, oneProjectDetailJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectTool(t, connectMCP(t, srv), "project_create", map[string]any{
		"workspace":  "acme",
		"name":       "Demo",
		"identifier": "DEMO",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	if got.Method != http.MethodPost {
		t.Fatalf("method = %q, want POST", got.Method)
	}
	if got.URL.Path != "/api/v1/workspaces/acme/projects/" {
		t.Fatalf("path = %q, want /api/v1/workspaces/acme/projects/", got.URL.Path)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}
	if !strings.Contains(string(body), `"name":"Demo"`) || !strings.Contains(string(body), `"identifier":"DEMO"`) {
		t.Fatalf("body missing name/identifier: %s", body)
	}
}

func TestProjectCreate_HappyPathReturnsProject(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(201, oneProjectDetailJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectTool(t, connectMCP(t, srv), "project_create", map[string]any{
		"workspace":  "acme",
		"name":       "Demo",
		"identifier": "DEMO",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal StructuredContent: %v", err)
	}
	if !strings.Contains(string(raw), `"identifier":"DEMO"`) && !strings.Contains(string(raw), `"identifier": "DEMO"`) {
		t.Fatalf("StructuredContent missing DEMO project: %s", raw)
	}
}

func TestProjectCreate_MapsConflict(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(409, `{}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_create", map[string]any{
		"workspace":  "acme",
		"name":       "Demo",
		"identifier": "DEMO",
	}))
	if !strings.Contains(strings.ToLower(text), "conflict") {
		t.Fatalf("error %q should describe conflict", text)
	}
}

func TestProjectCreate_MapsUnauthorized(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "bad-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(401, `{}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_create", map[string]any{
		"workspace":  "acme",
		"name":       "Demo",
		"identifier": "DEMO",
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}

func TestProjectUpdate_RequiresWorkspaceAndProjectID(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	session := connectMCP(t, srv)

	text := toolErrorText(t, callProjectTool(t, session, "project_update", map[string]any{
		"project_id": crudProjectID,
		"name":       "Renamed",
	}))
	if !strings.Contains(strings.ToLower(text), "workspace") {
		t.Fatalf("error %q should mention workspace", text)
	}

	text = toolErrorText(t, callProjectTool(t, session, "project_update", map[string]any{
		"workspace": "acme",
		"name":      "Renamed",
	}))
	if !strings.Contains(strings.ToLower(text), "project_id") {
		t.Fatalf("error %q should mention project_id", text)
	}
}

func TestProjectUpdate_SendsOnlyCallerPatchFields(t *testing.T) {
	t.Parallel()

	var got *http.Request
	var body []byte
	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			body, _ = io.ReadAll(r.Body)
			return jsonResponse(200, `{"name":"Renamed","identifier":"DEMO"}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectTool(t, connectMCP(t, srv), "project_update", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
		"name":       "Renamed",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	if got.Method != http.MethodPatch {
		t.Fatalf("method = %q, want PATCH", got.Method)
	}
	wantPath := "/api/v1/workspaces/acme/projects/" + crudProjectID + "/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %s", got.URL.Path, wantPath)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v\nbody=%s", err, body)
	}
	if payload["name"] != "Renamed" {
		t.Fatalf("name = %v, want Renamed", payload["name"])
	}
	if _, ok := payload["identifier"]; ok {
		t.Fatalf("identifier must not be sent on name-only patch: %s", body)
	}
	if _, ok := payload["description"]; ok {
		t.Fatalf("description must not be sent on name-only patch: %s", body)
	}
}

func TestProjectUpdate_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(404, `{}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_update", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
		"name":       "Renamed",
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestProjectDelete_RequiresWorkspaceAndProjectID(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	session := connectMCP(t, srv)

	text := toolErrorText(t, callProjectTool(t, session, "project_delete", map[string]any{
		"project_id": crudProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "workspace") {
		t.Fatalf("error %q should mention workspace", text)
	}

	text = toolErrorText(t, callProjectTool(t, session, "project_delete", map[string]any{
		"workspace": "acme",
	}))
	if !strings.Contains(strings.ToLower(text), "project_id") {
		t.Fatalf("error %q should mention project_id", text)
	}
}

func TestProjectDelete_SendsAuthPathAndMethod(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{BaseURL: "https://plane.example.com", APIKey: "pat-secret"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return &http.Response{
				StatusCode: 204,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectTool(t, connectMCP(t, srv), "project_delete", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	if got.Method != http.MethodDelete {
		t.Fatalf("method = %q, want DELETE", got.Method)
	}
	wantPath := "/api/v1/workspaces/acme/projects/" + crudProjectID + "/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %s", got.URL.Path, wantPath)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}
}

func TestProjectDelete_MapsNotFound(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(404, `{}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_delete", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestProjectDelete_MapsUnauthorized(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "bad-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(401, `{}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectTool(t, connectMCP(t, srv), "project_delete", map[string]any{
		"workspace":  "acme",
		"project_id": crudProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}
