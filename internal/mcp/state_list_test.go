package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	planemcp "github.com/dmipeck/plane-mcp/internal/mcp"
)

const emptyStatesJSON = `{
  "grouped_by": null,
  "sub_grouped_by": null,
  "total_count": 0,
  "next_cursor": "",
  "prev_cursor": "",
  "next_page_results": false,
  "prev_page_results": false,
  "count": 0,
  "total_pages": 0,
  "total_results": 0,
  "extra_stats": null,
  "results": []
}`

const oneStateJSON = `{
  "grouped_by": null,
  "sub_grouped_by": null,
  "total_count": 1,
  "next_cursor": "abc",
  "prev_cursor": "",
  "next_page_results": false,
  "prev_page_results": false,
  "count": 1,
  "total_pages": 1,
  "total_results": 1,
  "extra_stats": null,
  "results": [{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "In Progress",
    "color": "#ffa500",
    "group": "started",
    "sequence": 2
  }]
}`

const testStateProjectID = "550e8400-e29b-41d4-a716-446655440001"

func callStateList(t *testing.T, session *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "state_list",
		Arguments: json.RawMessage(raw),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func TestStateList_IsRegisteredAndNoWriteTools(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	session := connectMCP(t, srv)
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	foundStateList := false
	for _, tool := range tools.Tools {
		switch tool.Name {
		case "state_list":
			foundStateList = true
		case "create_state", "state_create", "state_view", "state_update", "state_delete":
			t.Fatalf("state write/view Tool %q must not be registered", tool.Name)
		}
	}
	if !foundStateList {
		t.Fatal("state_list not registered")
	}
}

func TestStateList_RequiresWorkspace(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callStateList(t, connectMCP(t, srv), map[string]any{
		"project_id": testStateProjectID,
	})
	text := toolErrorText(t, res)
	if !strings.Contains(strings.ToLower(text), "workspace") {
		t.Fatalf("error %q should mention workspace", text)
	}
}

func TestStateList_RequiresProjectID(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callStateList(t, connectMCP(t, srv), map[string]any{
		"workspace": "acme",
	})
	text := toolErrorText(t, res)
	if !strings.Contains(strings.ToLower(text), "project") {
		t.Fatalf("error %q should mention project_id", text)
	}
}

func TestStateList_SendsAuthPathAndMethod(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{
			BaseURL: "https://plane.example.com",
			APIKey:  "pat-secret",
		},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, emptyStatesJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callStateList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testStateProjectID,
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
	wantPath := "/api/v1/workspaces/acme/projects/" + testStateProjectID + "/states/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %s", got.URL.Path, wantPath)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}
}

func TestStateList_ForwardsOptionalParams(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, emptyStatesJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callStateList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testStateProjectID,
		"cursor":     "cur-1",
		"per_page":   50,
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	q := got.URL.Query()
	if q.Get("cursor") != "cur-1" {
		t.Fatalf("cursor = %q, want cur-1", q.Get("cursor"))
	}
	if q.Get("per_page") != "50" {
		t.Fatalf("per_page = %q, want 50", q.Get("per_page"))
	}
}

func TestStateList_HappyPathReturnsStates(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(200, oneStateJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callStateList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testStateProjectID,
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal StructuredContent: %v", err)
	}
	if !strings.Contains(string(raw), `"name":"In Progress"`) && !strings.Contains(string(raw), `"name": "In Progress"`) {
		t.Fatalf("StructuredContent missing In Progress state: %s", raw)
	}
	if !strings.Contains(string(raw), "550e8400-e29b-41d4-a716-446655440000") {
		t.Fatalf("StructuredContent missing state id: %s", raw)
	}
}

func TestStateList_MapsUnauthorized(t *testing.T) {
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

	text := toolErrorText(t, callStateList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testStateProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}

func TestStateList_MapsNotFound(t *testing.T) {
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

	text := toolErrorText(t, callStateList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "missing",
		"project_id": testStateProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestStateList_MapsForbidden(t *testing.T) {
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

	text := toolErrorText(t, callStateList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testStateProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "forbidden") {
		t.Fatalf("error %q should describe forbidden", text)
	}
}

func TestStateList_RejectsInvalidProjectID(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callStateList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": "not-a-uuid",
	}))
	if !strings.Contains(strings.ToLower(text), "project") {
		t.Fatalf("error %q should mention project_id", text)
	}
}
