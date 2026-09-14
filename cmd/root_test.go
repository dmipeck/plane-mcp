package cmd

import (
	"testing"

	planemcp "github.com/dmipeck/plane-mcp/internal/mcp"
)

func TestConnectionFromCmd_FlagOverridesEnv(t *testing.T) {
	t.Setenv("PLANE_BASE_URL", "https://env.example")
	t.Setenv("PLANE_API_KEY", "pat-from-env")

	root := newRoot()
	if err := root.ParseFlags([]string{"--plane-base-url", "https://flag.example"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	conn, err := connectionFromCmd(root)
	if err != nil {
		t.Fatalf("connectionFromCmd: %v", err)
	}
	if got, want := conn.BaseURL, "https://flag.example"; got != want {
		t.Fatalf("BaseURL = %q, want %q", got, want)
	}
	if got, want := conn.APIKey, "pat-from-env"; got != want {
		t.Fatalf("APIKey = %q, want %q", got, want)
	}
}

func TestConnectionFromCmd_EnvOverridesDefault(t *testing.T) {
	t.Setenv("PLANE_BASE_URL", "https://env.example")
	t.Setenv("PLANE_API_KEY", "pat-from-env")

	root := newRoot()
	conn, err := connectionFromCmd(root)
	if err != nil {
		t.Fatalf("connectionFromCmd: %v", err)
	}
	if got, want := conn.BaseURL, "https://env.example"; got != want {
		t.Fatalf("BaseURL = %q, want %q", got, want)
	}
}

func TestConnectionFromCmd_DefaultBaseURL(t *testing.T) {
	t.Setenv("PLANE_API_KEY", "pat-from-env")
	t.Setenv("PLANE_BASE_URL", "")

	root := newRoot()
	conn, err := connectionFromCmd(root)
	if err != nil {
		t.Fatalf("connectionFromCmd: %v", err)
	}
	if got, want := conn.BaseURL, planemcp.DefaultBaseURL; got != want {
		t.Fatalf("BaseURL = %q, want %q", got, want)
	}
}
