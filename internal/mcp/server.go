package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultBaseURL is used when Connection.BaseURL is empty.
const DefaultBaseURL = "https://api.plane.so"

// Connection is a Plane MCP session configured with base URL and PAT only.
// It is not locked to one Workspace.
type Connection struct {
	BaseURL string
	APIKey  string
}

// Server is the Plane MCP server built from a Connection.
type Server struct {
	Connection Connection
	MCP        *mcpsdk.Server
}

// New constructs an MCP server from Connection settings.
// Empty BaseURL becomes DefaultBaseURL. Missing APIKey fails fast.
// No Tools are registered yet.
func New(conn Connection) (*Server, error) {
	if strings.TrimSpace(conn.APIKey) == "" {
		return nil, fmt.Errorf("PLANE_API_KEY is required")
	}
	if strings.TrimSpace(conn.BaseURL) == "" {
		conn.BaseURL = DefaultBaseURL
	}

	s := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "plane-mcp",
		Version: "0.1.0",
	}, nil)

	return &Server{
		Connection: conn,
		MCP:        s,
	}, nil
}

// Run serves MCP over stdio until the client disconnects.
func (s *Server) Run(ctx context.Context) error {
	return s.MCP.Run(ctx, &mcpsdk.StdioTransport{})
}
