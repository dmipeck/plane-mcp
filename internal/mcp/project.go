package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dmipeck/plane-mcp/internal/oas"
)

// projectListInput is the MCP argument shape for project_list.
type projectListInput struct {
	Workspace string `json:"workspace" jsonschema:"Plane workspace slug"`
	Cursor    string `json:"cursor,omitempty" jsonschema:"Pagination cursor for the next page"`
	PerPage   int    `json:"per_page,omitempty" jsonschema:"Results per page (Plane default 20, max 100)"`
	OrderBy   string `json:"order_by,omitempty" jsonschema:"Order field; prefix with - for descending"`
}

func (s *Server) registerProjectTools() {
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "project_list",
		Description: "List projects in a Plane Workspace. Requires workspace slug; optional cursor, per_page, and order_by.",
	}, s.projectList)
}

func (s *Server) projectList(ctx context.Context, _ *mcpsdk.CallToolRequest, in projectListInput) (*mcpsdk.CallToolResult, any, error) {
	workspace := strings.TrimSpace(in.Workspace)
	if workspace == "" {
		return nil, nil, fmt.Errorf("workspace is required")
	}

	params := oas.ListProjectsParams{Slug: workspace}
	if in.Cursor != "" {
		params.Cursor = oas.NewOptString(in.Cursor)
	}
	if in.PerPage != 0 {
		params.PerPage = oas.NewOptInt(in.PerPage)
	}
	if in.OrderBy != "" {
		params.OrderBy = oas.NewOptString(in.OrderBy)
	}

	res, err := s.plane.ListProjects(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("plane: list projects: %w", err)
	}

	switch r := res.(type) {
	case *oas.PaginatedProjectResponse:
		return nil, r, nil
	case *oas.ListProjectsUnauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.ListProjectsForbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.ListProjectsNotFound:
		return nil, nil, fmt.Errorf("plane: workspace %q not found", workspace)
	default:
		return nil, nil, fmt.Errorf("plane: unexpected list projects response %T", res)
	}
}
