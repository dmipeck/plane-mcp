package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dmipeck/plane-mcp/internal/oas"
)

// DefaultBaseURL is used when Connection.BaseURL is empty.
const DefaultBaseURL = "https://api.plane.so"

// Connection is a Plane MCP session configured with base URL and PAT only.
// It is not locked to one Workspace.
type Connection struct {
	BaseURL string
	APIKey  string
}

// Option configures Server construction.
type Option func(*options)

type options struct {
	roundTripper http.RoundTripper
}

// WithRoundTripper injects the Plane HTTP transport (test seam).
func WithRoundTripper(rt http.RoundTripper) Option {
	return func(o *options) {
		o.roundTripper = rt
	}
}

// Server is the Plane MCP server built from a Connection.
type Server struct {
	Connection Connection
	MCP        *mcpsdk.Server
	plane      *oas.Client
}

type apiKeySource struct {
	key string
}

func (s apiKeySource) ApiKeyAuthentication(_ context.Context, _ oas.OperationName) (oas.ApiKeyAuthentication, error) {
	return oas.ApiKeyAuthentication{APIKey: s.key}, nil
}

// New constructs an MCP server from Connection settings.
// Empty BaseURL becomes DefaultBaseURL. Missing APIKey fails fast.
func New(conn Connection, opts ...Option) (*Server, error) {
	if strings.TrimSpace(conn.APIKey) == "" {
		return nil, fmt.Errorf("PLANE_API_KEY is required")
	}
	if strings.TrimSpace(conn.BaseURL) == "" {
		conn.BaseURL = DefaultBaseURL
	}

	var cfg options
	for _, opt := range opts {
		opt(&cfg)
	}

	httpClient := http.DefaultClient
	if cfg.roundTripper != nil {
		httpClient = &http.Client{Transport: cfg.roundTripper}
	}

	plane, err := oas.NewClient(conn.BaseURL, apiKeySource{key: conn.APIKey}, oas.WithClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("plane client: %w", err)
	}

	s := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "plane-mcp",
		Version: "0.1.0",
	}, nil)

	srv := &Server{
		Connection: conn,
		MCP:        s,
		plane:      plane,
	}
	srv.registerProjectTools()
	srv.registerStateTools()
	return srv, nil
}

// Run serves MCP over stdio until the client disconnects.
func (s *Server) Run(ctx context.Context) error {
	return s.MCP.Run(ctx, &mcpsdk.StdioTransport{})
}
