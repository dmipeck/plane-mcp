package mcp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dmipeck/plane-mcp/internal/oas"
)

// workitemListInput is the MCP argument shape for workitem_list.
type workitemListInput struct {
	Workspace string `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID string `json:"project_id" jsonschema:"Project UUID"`
	Cursor    string `json:"cursor,omitempty" jsonschema:"Pagination cursor for the next page"`
	PerPage   int    `json:"per_page,omitempty" jsonschema:"Results per page (Plane default 20, max 100)"`
	OrderBy   string `json:"order_by,omitempty" jsonschema:"Order field; prefix with - for descending"`
}

// workitemViewInput is the MCP argument shape for workitem_view.
// Callers supply either project_id+workitem_id or a human identifier like PROJ-123.
type workitemViewInput struct {
	Workspace  string `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID  string `json:"project_id,omitempty" jsonschema:"Project UUID (with workitem_id)"`
	WorkitemID string `json:"workitem_id,omitempty" jsonschema:"Workitem UUID (with project_id)"`
	Identifier string `json:"identifier,omitempty" jsonschema:"Human identifier like PROJ-123"`
}

func (s *Server) registerWorkitemTools() {
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "workitem_list",
		Description: "List work items in a Plane project. Requires workspace and project_id; optional cursor, per_page, and order_by. No PQL/search in v0.1.",
	}, s.workitemList)
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "workitem_view",
		Description: "View a Plane work item by project_id+workitem_id or by human identifier (e.g. PROJ-123).",
	}, s.workitemView)
}

func (s *Server) workitemList(ctx context.Context, _ *mcpsdk.CallToolRequest, in workitemListInput) (*mcpsdk.CallToolResult, any, error) {
	workspace := strings.TrimSpace(in.Workspace)
	if workspace == "" {
		return nil, nil, fmt.Errorf("workspace is required")
	}
	projectID, err := parseUUID("project_id", in.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	params := oas.ListWorkItems2Params{
		Slug:      workspace,
		ProjectID: projectID,
	}
	if in.Cursor != "" {
		params.Cursor = oas.NewOptString(in.Cursor)
	}
	if in.PerPage != 0 {
		params.PerPage = oas.NewOptInt(in.PerPage)
	}
	if in.OrderBy != "" {
		params.OrderBy = oas.NewOptString(in.OrderBy)
	}

	res, err := s.plane.ListWorkItems2(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("plane: list work items: %w", err)
	}

	switch r := res.(type) {
	case *oas.PaginatedWorkItemResponse:
		return nil, r, nil
	case *oas.ListWorkItems2Unauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.ListWorkItems2Forbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.ListWorkItems2NotFound:
		return nil, nil, fmt.Errorf("plane: project %q not found in workspace %q", projectID, workspace)
	case *oas.ListWorkItems2BadRequest:
		return nil, nil, fmt.Errorf("plane: bad request listing work items")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected list work items response %T", res)
	}
}

func (s *Server) workitemView(ctx context.Context, _ *mcpsdk.CallToolRequest, in workitemViewInput) (*mcpsdk.CallToolResult, any, error) {
	workspace := strings.TrimSpace(in.Workspace)
	if workspace == "" {
		return nil, nil, fmt.Errorf("workspace is required")
	}

	identifier := strings.TrimSpace(in.Identifier)
	projectIDRaw := strings.TrimSpace(in.ProjectID)
	workitemIDRaw := strings.TrimSpace(in.WorkitemID)

	hasIdentifier := identifier != ""
	hasUUIDPair := projectIDRaw != "" || workitemIDRaw != ""

	switch {
	case hasIdentifier && hasUUIDPair:
		return nil, nil, fmt.Errorf("provide either identifier or project_id+workitem_id, not both")
	case hasIdentifier:
		return s.workitemViewByIdentifier(ctx, workspace, identifier)
	case projectIDRaw != "" && workitemIDRaw != "":
		return s.workitemViewByIDs(ctx, workspace, projectIDRaw, workitemIDRaw)
	case projectIDRaw != "" || workitemIDRaw != "":
		return nil, nil, fmt.Errorf("project_id and workitem_id are both required when not using identifier")
	default:
		return nil, nil, fmt.Errorf("provide identifier or project_id+workitem_id")
	}
}

func (s *Server) workitemViewByIDs(ctx context.Context, workspace, projectIDRaw, workitemIDRaw string) (*mcpsdk.CallToolResult, any, error) {
	projectID, err := parseUUID("project_id", projectIDRaw)
	if err != nil {
		return nil, nil, err
	}
	workitemID, err := parseUUID("workitem_id", workitemIDRaw)
	if err != nil {
		return nil, nil, err
	}

	res, err := s.plane.RetrieveWorkItem2(ctx, oas.RetrieveWorkItem2Params{
		Slug:      workspace,
		ProjectID: projectID,
		Pk:        workitemID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: view work item: %w", err)
	}

	switch r := res.(type) {
	case *oas.Issue:
		return nil, r, nil
	case *oas.RetrieveWorkItem2Unauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.RetrieveWorkItem2Forbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.RetrieveWorkItem2NotFound:
		return nil, nil, fmt.Errorf("plane: work item %q not found in project %q", workitemID, projectID)
	case *oas.RetrieveWorkItem2BadRequest:
		return nil, nil, fmt.Errorf("plane: bad request viewing work item")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected view work item response %T", res)
	}
}

func (s *Server) workitemViewByIdentifier(ctx context.Context, workspace, identifier string) (*mcpsdk.CallToolResult, any, error) {
	projectIdentifier, issueIdentifier, err := splitWorkitemIdentifier(identifier)
	if err != nil {
		return nil, nil, err
	}

	res, err := s.plane.GetWorkspaceWorkItem2(ctx, oas.GetWorkspaceWorkItem2Params{
		Slug:              workspace,
		ProjectIdentifier: projectIdentifier,
		IssueIdentifier:   issueIdentifier,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: view work item: %w", err)
	}

	switch r := res.(type) {
	case *oas.Issue:
		return nil, r, nil
	case *oas.GetWorkspaceWorkItem2NotFound:
		return nil, nil, fmt.Errorf("plane: work item %q not found in workspace %q", identifier, workspace)
	default:
		return nil, nil, fmt.Errorf("plane: unexpected view work item response %T", res)
	}
}

func splitWorkitemIdentifier(identifier string) (projectIdentifier string, issueIdentifier int, err error) {
	i := strings.LastIndex(identifier, "-")
	if i <= 0 || i == len(identifier)-1 {
		return "", 0, fmt.Errorf("identifier %q must look like PROJ-123", identifier)
	}
	projectIdentifier = identifier[:i]
	seq, err := strconv.Atoi(identifier[i+1:])
	if err != nil {
		return "", 0, fmt.Errorf("identifier %q must end with a numeric issue sequence", identifier)
	}
	return projectIdentifier, seq, nil
}

func parseUUID(field, raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, fmt.Errorf("%s is required", field)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s must be a UUID: %w", field, err)
	}
	return id, nil
}
