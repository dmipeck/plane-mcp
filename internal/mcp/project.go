package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

// projectViewInput is the MCP argument shape for project_view.
type projectViewInput struct {
	Workspace string `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID string `json:"project_id" jsonschema:"Project UUID"`
}

// projectCreateInput is the MCP argument shape for project_create.
type projectCreateInput struct {
	Workspace   string `json:"workspace" jsonschema:"Plane workspace slug"`
	Name        string `json:"name" jsonschema:"Project name"`
	Identifier  string `json:"identifier" jsonschema:"Project identifier (unique within workspace)"`
	Description string `json:"description,omitempty" jsonschema:"Optional project description"`
}

// projectUpdateInput is the MCP argument shape for project_update.
// Optional fields are pointers so unset args are omitted from the Plane PATCH body.
type projectUpdateInput struct {
	Workspace   string  `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID   string  `json:"project_id" jsonschema:"Project UUID"`
	Name        *string `json:"name,omitempty" jsonschema:"New project name"`
	Identifier  *string `json:"identifier,omitempty" jsonschema:"New project identifier"`
	Description *string `json:"description,omitempty" jsonschema:"New project description"`
}

// projectDeleteInput is the MCP argument shape for project_delete.
type projectDeleteInput struct {
	Workspace string `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID string `json:"project_id" jsonschema:"Project UUID"`
}

func (s *Server) registerProjectTools() {
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "project_list",
		Description: "List projects in a Plane Workspace. Requires workspace slug; optional cursor, per_page, and order_by.",
	}, s.projectList)
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "project_view",
		Description: "Retrieve a single Plane project by project_id. Requires workspace slug and project_id.",
	}, s.projectView)
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "project_create",
		Description: "Create a Plane project. Requires workspace slug, name, and identifier.",
	}, s.projectCreate)
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "project_update",
		Description: "Patch a Plane project with caller-supplied fields only (not a full replace). Requires workspace and project_id; optional name, identifier, description.",
	}, s.projectUpdate)
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "project_delete",
		Description: "Delete a Plane project. Requires workspace slug and project_id.",
	}, s.projectDelete)
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

func (s *Server) projectView(ctx context.Context, _ *mcpsdk.CallToolRequest, in projectViewInput) (*mcpsdk.CallToolResult, any, error) {
	workspace, projectID, err := requireWorkspaceProjectID(in.Workspace, in.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	res, err := s.plane.RetrieveProject(ctx, oas.RetrieveProjectParams{
		Slug: workspace,
		Pk:   projectID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: retrieve project: %w", err)
	}

	switch r := res.(type) {
	case *oas.Project:
		return nil, r, nil
	case *oas.RetrieveProjectUnauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.RetrieveProjectForbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.RetrieveProjectNotFound:
		return nil, nil, fmt.Errorf("plane: project not found")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected retrieve project response %T", res)
	}
}

func (s *Server) projectCreate(ctx context.Context, _ *mcpsdk.CallToolRequest, in projectCreateInput) (*mcpsdk.CallToolResult, any, error) {
	workspace := strings.TrimSpace(in.Workspace)
	if workspace == "" {
		return nil, nil, fmt.Errorf("workspace is required")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}
	identifier := strings.TrimSpace(in.Identifier)
	if identifier == "" {
		return nil, nil, fmt.Errorf("identifier is required")
	}

	req := &oas.CreateProjectApplicationJSON{
		Name:       name,
		Identifier: identifier,
	}
	if desc := strings.TrimSpace(in.Description); desc != "" {
		req.Description = oas.NewOptString(desc)
	}

	res, err := s.plane.CreateProject(ctx, req, oas.CreateProjectParams{Slug: workspace})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: create project: %w", err)
	}

	switch r := res.(type) {
	case *oas.Project:
		return nil, r, nil
	case *oas.CreateProjectUnauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.CreateProjectForbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.CreateProjectNotFound:
		return nil, nil, fmt.Errorf("plane: workspace %q not found", workspace)
	case *oas.CreateProjectConflict:
		return nil, nil, fmt.Errorf("plane: conflict creating project")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected create project response %T", res)
	}
}

func (s *Server) projectUpdate(ctx context.Context, _ *mcpsdk.CallToolRequest, in projectUpdateInput) (*mcpsdk.CallToolResult, any, error) {
	workspace, projectID, err := requireWorkspaceProjectID(in.Workspace, in.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	patch := &oas.UpdateProjectApplicationJSON{}
	if in.Name != nil {
		patch.Name = oas.NewOptString(*in.Name)
	}
	if in.Identifier != nil {
		patch.Identifier = oas.NewOptString(*in.Identifier)
	}
	if in.Description != nil {
		patch.Description = oas.NewOptString(*in.Description)
	}

	res, err := s.plane.UpdateProject(ctx, patch, oas.UpdateProjectParams{
		Slug: workspace,
		Pk:   projectID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: update project: %w", err)
	}

	switch r := res.(type) {
	case *oas.Project:
		return nil, r, nil
	case *oas.UpdateProjectUnauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.UpdateProjectForbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.UpdateProjectNotFound:
		return nil, nil, fmt.Errorf("plane: project not found")
	case *oas.UpdateProjectConflict:
		return nil, nil, fmt.Errorf("plane: conflict updating project")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected update project response %T", res)
	}
}

func (s *Server) projectDelete(ctx context.Context, _ *mcpsdk.CallToolRequest, in projectDeleteInput) (*mcpsdk.CallToolResult, any, error) {
	workspace, projectID, err := requireWorkspaceProjectID(in.Workspace, in.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	res, err := s.plane.DeleteProject(ctx, oas.DeleteProjectParams{
		Slug: workspace,
		Pk:   projectID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: delete project: %w", err)
	}

	switch r := res.(type) {
	case *oas.DeleteProjectNoContent:
		return nil, r, nil
	case *oas.DeleteProjectUnauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.DeleteProjectForbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.DeleteProjectNotFound:
		return nil, nil, fmt.Errorf("plane: project not found")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected delete project response %T", res)
	}
}

func requireWorkspaceProjectID(workspaceRaw, projectIDRaw string) (string, uuid.UUID, error) {
	workspace := strings.TrimSpace(workspaceRaw)
	if workspace == "" {
		return "", uuid.UUID{}, fmt.Errorf("workspace is required")
	}
	projectIDStr := strings.TrimSpace(projectIDRaw)
	if projectIDStr == "" {
		return "", uuid.UUID{}, fmt.Errorf("project_id is required")
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return "", uuid.UUID{}, fmt.Errorf("project_id must be a UUID: %w", err)
	}
	return workspace, projectID, nil
}
