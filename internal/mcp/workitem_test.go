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

const (
	testProjectID  = "550e8400-e29b-41d4-a716-446655440000"
	testWorkitemID = "660e8400-e29b-41d4-a716-446655440001"
)

const emptyWorkitemsJSON = `{
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

const oneWorkitemListJSON = `{
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
  "results": [{"name": "Ship dual-id view", "sequence_id": 123}]
}`

const oneWorkitemJSON = `{"name": "Ship dual-id view", "sequence_id": 123}`

func callWorkitemList(t *testing.T, session *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	return callTool(t, session, "workitem_list", args)
}

func callWorkitemView(t *testing.T, session *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	return callTool(t, session, "workitem_view", args)
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: json.RawMessage(raw),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func TestWorkitemList_IsRegistered(t *testing.T) {
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

	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "workitem_list" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("workitem_list not registered")
	}
}

func TestWorkitemList_RequiresWorkspaceAndProjectID(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	session := connectMCP(t, srv)

	text := toolErrorText(t, callWorkitemList(t, session, map[string]any{
		"project_id": testProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "workspace") {
		t.Fatalf("error %q should mention workspace", text)
	}

	text = toolErrorText(t, callWorkitemList(t, session, map[string]any{
		"workspace": "acme",
	}))
	if !strings.Contains(strings.ToLower(text), "project_id") {
		t.Fatalf("error %q should mention project_id", text)
	}
}

func TestWorkitemList_SendsAuthPathAndMethod(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{
			BaseURL: "https://plane.example.com",
			APIKey:  "pat-secret",
		},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, emptyWorkitemsJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
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
	wantPath := "/api/v1/workspaces/acme/projects/" + testProjectID + "/work-items/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %q", got.URL.Path, wantPath)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}
}

func TestWorkitemList_ForwardsOptionalParams(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, emptyWorkitemsJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
		"cursor":     "cur-1",
		"per_page":   50,
		"order_by":   "-created_at",
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
	if q.Get("order_by") != "-created_at" {
		t.Fatalf("order_by = %q, want -created_at", q.Get("order_by"))
	}
}

func TestWorkitemList_HappyPathReturnsWorkitems(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(200, oneWorkitemListJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal StructuredContent: %v", err)
	}
	if !strings.Contains(string(raw), "Ship dual-id view") {
		t.Fatalf("StructuredContent missing work item name: %s", raw)
	}
}

func TestWorkitemList_MapsUnauthorized(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}

func TestWorkitemList_MapsNotFound(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "missing",
		"project_id": testProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestWorkitemList_MapsForbidden(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemList(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"project_id": testProjectID,
	}))
	if !strings.Contains(strings.ToLower(text), "forbidden") {
		t.Fatalf("error %q should describe forbidden", text)
	}
}

func TestWorkitemView_ByProjectAndWorkitemID(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{
			BaseURL: "https://plane.example.com",
			APIKey:  "pat-secret",
		},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, oneWorkitemJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemView(t, connectMCP(t, srv), map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	wantPath := "/api/v1/workspaces/acme/projects/" + testProjectID + "/work-items/" + testWorkitemID + "/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %q", got.URL.Path, wantPath)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal StructuredContent: %v", err)
	}
	if !strings.Contains(string(raw), "Ship dual-id view") {
		t.Fatalf("StructuredContent missing work item name: %s", raw)
	}
}

func TestWorkitemView_ByIdentifierSplitsBeforeClientCall(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, oneWorkitemJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemView(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"identifier": "PROJ-123",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	wantPath := "/api/v1/workspaces/acme/work-items/PROJ-123/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %q (identifier must split into project_identifier + issue_identifier)", got.URL.Path, wantPath)
	}
}

func TestWorkitemView_IdentifierWithHyphenatedProject(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, oneWorkitemJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callWorkitemView(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"identifier": "MY-PROJ-42",
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", toolErrorText(t, res))
	}
	wantPath := "/api/v1/workspaces/acme/work-items/MY-PROJ-42/"
	if got.URL.Path != wantPath {
		t.Fatalf("path = %q, want %q", got.URL.Path, wantPath)
	}
}

func TestWorkitemView_RejectsMixedIDModes(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callWorkitemView(t, connectMCP(t, srv), map[string]any{
		"workspace":   "acme",
		"identifier":  "PROJ-123",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
	}))
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "identifier") || !strings.Contains(lower, "project_id") {
		t.Fatalf("error %q should reject mixed id modes", text)
	}
}

func TestWorkitemView_RejectsInvalidIdentifier(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callWorkitemView(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"identifier": "PROJ",
	}))
	if !strings.Contains(strings.ToLower(text), "identifier") {
		t.Fatalf("error %q should mention identifier shape", text)
	}
}

func TestWorkitemView_MapsNotFoundForUUIDPath(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemView(t, connectMCP(t, srv), map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestWorkitemView_MapsNotFoundForIdentifierPath(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemView(t, connectMCP(t, srv), map[string]any{
		"workspace":  "acme",
		"identifier": "PROJ-999",
	}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestWorkitemView_MapsUnauthorizedForUUIDPath(t *testing.T) {
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

	text := toolErrorText(t, callWorkitemView(t, connectMCP(t, srv), map[string]any{
		"workspace":   "acme",
		"project_id":  testProjectID,
		"workitem_id": testWorkitemID,
	}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}
