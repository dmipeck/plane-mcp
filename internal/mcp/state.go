package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dmipeck/plane-mcp/internal/oas"
)

// stateListInput is the MCP argument shape for state_list.
type stateListInput struct {
	Workspace string `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID string `json:"project_id" jsonschema:"Project UUID"`
	Cursor    string `json:"cursor,omitempty" jsonschema:"Pagination cursor for the next page"`
	PerPage   int    `json:"per_page,omitempty" jsonschema:"Results per page (Plane default 20, max 100)"`
}

func (s *Server) registerStateTools() {
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "state_list",
		Description: "List workflow States for a Plane project. Supporting Resource only in v0.1 — resolve State UUIDs before workitem writes. Requires workspace and project_id; optional cursor and per_page.",
	}, s.stateList)
}

func (s *Server) stateList(ctx context.Context, _ *mcpsdk.CallToolRequest, in stateListInput) (*mcpsdk.CallToolResult, any, error) {
	workspace := strings.TrimSpace(in.Workspace)
	if workspace == "" {
		return nil, nil, fmt.Errorf("workspace is required")
	}
	projectIDRaw := strings.TrimSpace(in.ProjectID)
	if projectIDRaw == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}
	projectID, err := uuid.Parse(projectIDRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("project_id must be a UUID: %w", err)
	}

	params := oas.ListStatesParams{
		Slug:      workspace,
		ProjectID: projectID,
	}
	if in.Cursor != "" {
		params.Cursor = oas.NewOptString(in.Cursor)
	}
	if in.PerPage != 0 {
		params.PerPage = oas.NewOptInt(in.PerPage)
	}

	res, err := s.plane.ListStates(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("plane: list states: %w", err)
	}

	switch r := res.(type) {
	case *oas.PaginatedStateResponse:
		return nil, r, nil
	case *oas.ListStatesUnauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.ListStatesForbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.ListStatesNotFound:
		return nil, nil, fmt.Errorf("plane: workspace or project not found")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected list states response %T", res)
	}
}
