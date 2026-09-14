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

const (
	testStateID  = "770e8400-e29b-41d4-a716-446655440002"
	testParentID = "880e8400-e29b-41d4-a716-446655440003"
	testAssignee = "990e8400-e29b-41d4-a716-446655440004"
	testLabelID  = "aa0e8400-e29b-41d4-a716-446655440005"
)

const createdWorkitemJSON = `{"name": "File from agent", "sequence_id": 42}`

func callWorkitemWrite(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	return callTool(t, session, name, args)
}

func TestWorkitemCreate_IsRegistered(t *testing.T) {
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
		if tool.Name == "workitem_create" {
			return
		}
	}
	t.Fatal("workitem_create not registered")
}

func TestWorkitemCreate_RequiresWorkspaceProjectIDName(t *testing.T) {
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
		{"missing workspace", map[string]any{"project_id": testProjectID, "name": "Task"}, "workspace"},
		{"missing project_id", map[string]any{"workspace": "acme", "name": "Task"}, "project_id"},
		{"missing name", map[string]any{"workspace": "acme", "project_id": testProjectID}, "name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			text := toolErrorText(t, callWorkitemWrite(t, session, "workitem_create", tc.args))
			if !strings.Contains(strings.ToLower(text), tc.want) {
				t.Fatalf("error %q should mention %s", text, tc.want)
			}
		})
	}
}

func TestWorkitemCreate_SendsAuthPathBodyAndMethod(t *testing.T) {
	t.Parallel()

	var got *http.Request
	var body []byte
	srv, err := planemcp.New(
		planemcp.Connection{BaseURL: "https://plane.example.com", APIKey: "pat-secret"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			body, _ = io.ReadAll(r.Body)
			return jsonResponse(201, createdWorkitemJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemWrite(t, connectMCP(t, srv), "workitem_create", map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"name":        "File from agent",
		"description": "<p>Details</p>",
		"priority":    "high",
		"state":       testStateID,
		"parent":      testParentID,
		"assignees":   []string{testAssignee},
		"labels":      []string{testLabelID},
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	if got.Method != http.MethodPost {
		t.Fatalf("method = %q, want POST", got.Method)
	}
	wantPath := "/api/v1/workspaces/acme/projects/" + testProjectID + "/work-items/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %q", got.URL.Path, wantPath)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v\nbody=%s", err, body)
	}
	if payload["name"] != "File from agent" {
		t.Fatalf("name = %v, want File from agent", payload["name"])
	}
	if payload["description_html"] != "<p>Details</p>" {
		t.Fatalf("description_html = %v, want <p>Details</p>", payload["description_html"])
	}
	if payload["priority"] != "high" {
		t.Fatalf("priority = %v, want high", payload["priority"])
	}
	if payload["state"] != testStateID {
		t.Fatalf("state = %v, want %s", payload["state"], testStateID)
	}
	if payload["parent"] != testParentID {
		t.Fatalf("parent = %v, want %s", payload["parent"], testParentID)
	}
	assignees, ok := payload["assignees"].([]any)
	if !ok || len(assignees) != 1 || assignees[0] != testAssignee {
		t.Fatalf("assignees = %v, want [%s]", payload["assignees"], testAssignee)
	}
	labels, ok := payload["labels"].([]any)
	if !ok || len(labels) != 1 || labels[0] != testLabelID {
		t.Fatalf("labels = %v, want [%s]", payload["labels"], testLabelID)
	}
}

func TestWorkitemCreate_HappyPathReturnsWorkitem(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(201, createdWorkitemJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemWrite(t, connectMCP(t, srv), "workitem_create", map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
		"name":       "File from agent",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal StructuredContent: %v", err)
	}
	if !strings.Contains(string(raw), "File from agent") {
		t.Fatalf("StructuredContent missing work item name: %s", raw)
	}
}

func TestWorkitemCreate_OmitsUnsetOptionalFields(t *testing.T) {
	t.Parallel()

	var body []byte
	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, _ = io.ReadAll(r.Body)
			return jsonResponse(201, createdWorkitemJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemWrite(t, connectMCP(t, srv), "workitem_create", map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
		"name":       "Minimal",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v\nbody=%s", err, body)
	}
	for _, key := range []string{"description_html", "priority", "state", "parent", "assignees", "labels"} {
		if _, ok := payload[key]; ok {
			t.Fatalf("%s must be omitted when unset: %s", key, body)
		}
	}
}

func TestWorkitemCreate_MapsUnauthorized(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemWrite(t, connectMCP(t, srv), "workitem_create", map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
		"name":       "Task",
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}

func TestWorkitemCreate_MapsNotFound(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemWrite(t, connectMCP(t, srv), "workitem_create", map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
		"name":       "Task",
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestWorkitemCreate_MapsConflict(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemWrite(t, connectMCP(t, srv), "workitem_create", map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
		"name":       "Task",
	}))
	if !strings.Contains(strings.ToLower(text), "conflict") {
		t.Fatalf("error %q should describe conflict", text)
	}
}

func TestWorkitemCreate_MapsForbidden(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemWrite(t, connectMCP(t, srv), "workitem_create", map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
		"name":       "Task",
	}))
	if !strings.Contains(strings.ToLower(text), "forbidden") {
		t.Fatalf("error %q should describe forbidden", text)
	}
}

func TestWorkitemUpdate_RequiresWorkspaceProjectIDWorkitemID(t *testing.T) {
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
		{"missing workspace", map[string]any{"project_id": testProjectID, "workitem_id": testWorkitemID, "name": "Renamed"}, "workspace"},
		{"missing project_id", map[string]any{"workspace": "acme", "workitem_id": testWorkitemID, "name": "Renamed"}, "project_id"},
		{"missing workitem_id", map[string]any{"workspace": "acme", "project_id": testProjectID, "name": "Renamed"}, "workitem_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			text := toolErrorText(t, callWorkitemWrite(t, session, "workitem_update", tc.args))
			if !strings.Contains(strings.ToLower(text), tc.want) {
				t.Fatalf("error %q should mention %s", text, tc.want)
			}
		})
	}
}

func TestWorkitemUpdate_SendsOnlyCallerPatchFields(t *testing.T) {
	t.Parallel()

	var got *http.Request
	var body []byte
	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			body, _ = io.ReadAll(r.Body)
			return jsonResponse(200, `{"name":"Renamed"}`), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemWrite(t, connectMCP(t, srv), "workitem_update", map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
		"name":        "Renamed",
		"priority":    "low",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	if got.Method != http.MethodPatch {
		t.Fatalf("method = %q, want PATCH", got.Method)
	}
	wantPath := "/api/v1/workspaces/acme/projects/" + testProjectID + "/work-items/" + testWorkitemID + "/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %q", got.URL.Path, wantPath)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v\nbody=%s", err, body)
	}
	if payload["name"] != "Renamed" {
		t.Fatalf("name = %v, want Renamed", payload["name"])
	}
	if payload["priority"] != "low" {
		t.Fatalf("priority = %v, want low", payload["priority"])
	}
	for _, key := range []string{"description_html", "state", "parent", "assignees", "labels"} {
		if _, ok := payload[key]; ok {
			t.Fatalf("%s must not be sent on partial patch: %s", key, body)
		}
	}
}

func TestWorkitemUpdate_MapsNotFound(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemWrite(t, connectMCP(t, srv), "workitem_update", map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
		"name":        "Renamed",
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestWorkitemUpdate_MapsUnauthorized(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemWrite(t, connectMCP(t, srv), "workitem_update", map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
		"name":        "Renamed",
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}

func TestWorkitemDelete_RequiresWorkspaceProjectIDWorkitemID(t *testing.T) {
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
		{"missing workspace", map[string]any{"project_id": testProjectID, "workitem_id": testWorkitemID}, "workspace"},
		{"missing project_id", map[string]any{"workspace": "acme", "workitem_id": testWorkitemID}, "project_id"},
		{"missing workitem_id", map[string]any{"workspace": "acme", "project_id": testProjectID}, "workitem_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			text := toolErrorText(t, callWorkitemWrite(t, session, "workitem_delete", tc.args))
			if !strings.Contains(strings.ToLower(text), tc.want) {
				t.Fatalf("error %q should mention %s", text, tc.want)
			}
		})
	}
}

func TestWorkitemDelete_SendsAuthPathAndMethod(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{BaseURL: "https://plane.example.com", APIKey: "pat-secret"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return &http.Response{
				StatusCode: 204,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemWrite(t, connectMCP(t, srv), "workitem_delete", map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	if got.Method != http.MethodDelete {
		t.Fatalf("method = %q, want DELETE", got.Method)
	}
	wantPath := "/api/v1/workspaces/acme/projects/" + testProjectID + "/work-items/" + testWorkitemID + "/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %q", got.URL.Path, wantPath)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}
}

func TestWorkitemDelete_MapsNotFound(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemWrite(t, connectMCP(t, srv), "workitem_delete", map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestWorkitemDelete_MapsUnauthorized(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemWrite(t, connectMCP(t, srv), "workitem_delete", map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}
