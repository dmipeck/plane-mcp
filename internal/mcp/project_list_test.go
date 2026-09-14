package mcp_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	planemcp "github.com/dmipeck/plane-mcp/internal/mcp"
)

const emptyProjectsJSON = `{
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

const oneProjectJSON = `{
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
  "results": [{"name": "Demo", "identifier": "DEMO"}]
}`

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func connectMCP(t *testing.T, srv *planemcp.Server) *mcp.ClientSession {
	t.Helper()

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := srv.MCP.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func callProjectList(t *testing.T, session *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "project_list",
		Arguments: json.RawMessage(raw),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func toolErrorText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if !res.IsError {
		t.Fatal("want IsError=true")
	}
	if len(res.Content) == 0 {
		t.Fatal("want error content")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type %T, want TextContent", res.Content[0])
	}
	return text.Text
}

func TestProjectList_IsRegistered(t *testing.T) {
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
		if tool.Name == "project_list" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("project_list not registered")
	}
}

func TestProjectList_RequiresWorkspace(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"}, planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP call")
		return nil, nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectList(t, connectMCP(t, srv), map[string]any{})
	text := toolErrorText(t, res)
	if !strings.Contains(strings.ToLower(text), "workspace") {
		t.Fatalf("error %q should mention workspace", text)
	}
}

func TestProjectList_SendsAuthPathAndMethod(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{
			BaseURL: "https://plane.example.com",
			APIKey:  "pat-secret",
		},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, emptyProjectsJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectList(t, connectMCP(t, srv), map[string]any{
		"workspace": "acme",
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
	if got.URL.Path != "/api/v1/workspaces/acme/projects/" {
		t.Fatalf("path = %q, want /api/v1/workspaces/acme/projects/", got.URL.Path)
	}
	if got.Header.Get("X-API-Key") != "pat-secret" {
		t.Fatalf("X-API-Key = %q, want pat-secret", got.Header.Get("X-API-Key"))
	}
}

func TestProjectList_ForwardsOptionalParams(t *testing.T) {
	t.Parallel()

	var got *http.Request
	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r.Clone(r.Context())
			return jsonResponse(200, emptyProjectsJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectList(t, connectMCP(t, srv), map[string]any{
		"workspace": "acme",
		"cursor":    "cur-1",
		"per_page":  50,
		"order_by":  "-created_at",
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

func TestProjectList_HappyPathReturnsProjects(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(200, oneProjectJSON), nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res := callProjectList(t, connectMCP(t, srv), map[string]any{"workspace": "acme"})
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

func TestProjectList_MapsUnauthorized(t *testing.T) {
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

	text := toolErrorText(t, callProjectList(t, connectMCP(t, srv), map[string]any{"workspace": "acme"}))
	if !strings.Contains(strings.ToLower(text), "unauthorized") && !strings.Contains(strings.ToLower(text), "auth") {
		t.Fatalf("error %q should describe auth failure", text)
	}
}

func TestProjectList_MapsNotFound(t *testing.T) {
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

	text := toolErrorText(t, callProjectList(t, connectMCP(t, srv), map[string]any{"workspace": "missing"}))
	if !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("error %q should describe not-found", text)
	}
}

func TestProjectList_MapsForbidden(t *testing.T) {
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

	text := toolErrorText(t, callProjectList(t, connectMCP(t, srv), map[string]any{"workspace": "acme"}))
	if !strings.Contains(strings.ToLower(text), "forbidden") {
		t.Fatalf("error %q should describe forbidden", text)
	}
}

func TestProjectList_MapsValidationError(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(
		planemcp.Connection{APIKey: "test-pat"},
		planemcp.WithRoundTripper(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 400,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"detail":"invalid cursor"}`)),
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	text := toolErrorText(t, callProjectList(t, connectMCP(t, srv), map[string]any{
		"workspace": "acme",
		"cursor":    "bad",
	}))
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "400") && !strings.Contains(lower, "validation") && !strings.Contains(lower, "unexpected") {
		t.Fatalf("error %q should surface validation/unexpected Plane status", text)
	}
}
