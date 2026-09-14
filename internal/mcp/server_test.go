package mcp_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	planemcp "github.com/dmipeck/plane-mcp/internal/mcp"
)

func TestNew_AppliesDefaultBaseURL(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{
		APIKey: "test-pat",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got, want := srv.Connection.BaseURL, planemcp.DefaultBaseURL; got != want {
		t.Fatalf("BaseURL = %q, want %q", got, want)
	}
	if got, want := srv.Connection.APIKey, "test-pat"; got != want {
		t.Fatalf("APIKey = %q, want %q", got, want)
	}
}

func TestNew_KeepsExplicitBaseURL(t *testing.T) {
	t.Parallel()

	const url = "https://plane.example.com"
	srv, err := planemcp.New(planemcp.Connection{
		BaseURL: url,
		APIKey:  "test-pat",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := srv.Connection.BaseURL; got != url {
		t.Fatalf("BaseURL = %q, want %q", got, url)
	}
}

func TestNew_FailsWhenAPIKeyMissing(t *testing.T) {
	t.Parallel()

	_, err := planemcp.New(planemcp.Connection{
		BaseURL: "https://plane.example.com",
	})
	if err == nil {
		t.Fatal("New: want error for missing API key, got nil")
	}
}

func TestNew_RegistersZeroTools(t *testing.T) {
	t.Parallel()

	srv, err := planemcp.New(planemcp.Connection{APIKey: "test-pat"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

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

	tools, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) != 0 {
		t.Fatalf("registered tools = %d, want 0", len(tools.Tools))
	}
}
