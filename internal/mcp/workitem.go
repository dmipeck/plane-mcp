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

// workitemCreateInput is the MCP argument shape for workitem_create.
type workitemCreateInput struct {
	Workspace   string   `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID   string   `json:"project_id" jsonschema:"Project UUID"`
	Name        string   `json:"name" jsonschema:"Work item name"`
	State       string   `json:"state,omitempty" jsonschema:"State UUID"`
	Description string   `json:"description,omitempty" jsonschema:"Description HTML sent as Plane description_html"`
	Priority    string   `json:"priority,omitempty" jsonschema:"Priority: urgent|high|medium|low|none"`
	Assignees   []string `json:"assignees,omitempty" jsonschema:"Assignee UUIDs"`
	Labels      []string `json:"labels,omitempty" jsonschema:"Label UUIDs"`
	Parent      string   `json:"parent,omitempty" jsonschema:"Parent work item UUID"`
}

// workitemUpdateInput is the MCP argument shape for workitem_update.
// Optional fields are pointers/slices so unset args are omitted from the Plane PATCH body.
type workitemUpdateInput struct {
	Workspace   string   `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID   string   `json:"project_id" jsonschema:"Project UUID"`
	WorkitemID  string   `json:"workitem_id" jsonschema:"Work item UUID"`
	Name        *string  `json:"name,omitempty" jsonschema:"New work item name"`
	State       *string  `json:"state,omitempty" jsonschema:"State UUID"`
	Description *string  `json:"description,omitempty" jsonschema:"Description HTML sent as Plane description_html"`
	Priority    *string  `json:"priority,omitempty" jsonschema:"Priority: urgent|high|medium|low|none"`
	Assignees   []string `json:"assignees,omitempty" jsonschema:"Assignee UUIDs"`
	Labels      []string `json:"labels,omitempty" jsonschema:"Label UUIDs"`
	Parent      *string  `json:"parent,omitempty" jsonschema:"Parent work item UUID"`
}

// workitemDeleteInput is the MCP argument shape for workitem_delete.
type workitemDeleteInput struct {
	Workspace  string `json:"workspace" jsonschema:"Plane workspace slug"`
	ProjectID  string `json:"project_id" jsonschema:"Project UUID"`
	WorkitemID string `json:"workitem_id" jsonschema:"Work item UUID"`
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
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "workitem_create",
		Description: "Create a Plane work item. Requires workspace, project_id, and name; optional state, description, priority, assignees, labels, parent.",
	}, s.workitemCreate)
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "workitem_update",
		Description: "Patch a Plane work item with caller-supplied fields only. Requires workspace, project_id, and workitem_id; optional name, state, description, priority, assignees, labels, parent.",
	}, s.workitemUpdate)
	mcpsdk.AddTool(s.MCP, &mcpsdk.Tool{
		Name:        "workitem_delete",
		Description: "Delete a Plane work item. Requires workspace, project_id, and workitem_id.",
	}, s.workitemDelete)
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

func (s *Server) workitemCreate(ctx context.Context, _ *mcpsdk.CallToolRequest, in workitemCreateInput) (*mcpsdk.CallToolResult, any, error) {
	workspace := strings.TrimSpace(in.Workspace)
	if workspace == "" {
		return nil, nil, fmt.Errorf("workspace is required")
	}
	projectID, err := parseUUID("project_id", in.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}

	req := &oas.CreateWorkItem2ApplicationJSON{Name: name}
	if desc := strings.TrimSpace(in.Description); desc != "" {
		req.DescriptionHTML = oas.NewOptString(desc)
	}
	if in.Priority != "" {
		priority, err := parsePriority(in.Priority)
		if err != nil {
			return nil, nil, err
		}
		req.Priority = oas.NewOptPriorityEnum(priority)
	}
	if in.State != "" {
		stateID, err := parseUUID("state", in.State)
		if err != nil {
			return nil, nil, err
		}
		req.State = oas.NewOptNilUUID(stateID)
	}
	if in.Parent != "" {
		parentID, err := parseUUID("parent", in.Parent)
		if err != nil {
			return nil, nil, err
		}
		req.Parent = oas.NewOptNilUUID(parentID)
	}
	if in.Assignees != nil {
		assignees, err := parseUUIDList("assignees", in.Assignees)
		if err != nil {
			return nil, nil, err
		}
		req.Assignees = assignees
	}
	if in.Labels != nil {
		labels, err := parseUUIDList("labels", in.Labels)
		if err != nil {
			return nil, nil, err
		}
		req.Labels = labels
	}

	res, err := s.plane.CreateWorkItem2(ctx, req, oas.CreateWorkItem2Params{
		Slug:      workspace,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: create work item: %w", err)
	}

	switch r := res.(type) {
	case *oas.Issue:
		return nil, r, nil
	case *oas.CreateWorkItem2Unauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.CreateWorkItem2Forbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.CreateWorkItem2NotFound:
		return nil, nil, fmt.Errorf("plane: project %q not found in workspace %q", projectID, workspace)
	case *oas.CreateWorkItem2BadRequest:
		return nil, nil, fmt.Errorf("plane: bad request creating work item")
	case *oas.CreateWorkItem2Conflict:
		return nil, nil, fmt.Errorf("plane: conflict creating work item")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected create work item response %T", res)
	}
}

func (s *Server) workitemUpdate(ctx context.Context, _ *mcpsdk.CallToolRequest, in workitemUpdateInput) (*mcpsdk.CallToolResult, any, error) {
	workspace, projectID, workitemID, err := requireWorkspaceProjectWorkitem(in.Workspace, in.ProjectID, in.WorkitemID)
	if err != nil {
		return nil, nil, err
	}

	patch := &oas.UpdateWorkItem2ApplicationJSON{}
	if in.Name != nil {
		patch.Name = oas.NewOptString(*in.Name)
	}
	if in.Description != nil {
		patch.DescriptionHTML = oas.NewOptString(*in.Description)
	}
	if in.Priority != nil {
		priority, err := parsePriority(*in.Priority)
		if err != nil {
			return nil, nil, err
		}
		patch.Priority = oas.NewOptPriorityEnum(priority)
	}
	if in.State != nil {
		stateID, err := parseUUID("state", *in.State)
		if err != nil {
			return nil, nil, err
		}
		patch.State = oas.NewOptNilUUID(stateID)
	}
	if in.Parent != nil {
		parentID, err := parseUUID("parent", *in.Parent)
		if err != nil {
			return nil, nil, err
		}
		patch.Parent = oas.NewOptNilUUID(parentID)
	}
	if in.Assignees != nil {
		assignees, err := parseUUIDList("assignees", in.Assignees)
		if err != nil {
			return nil, nil, err
		}
		patch.Assignees = assignees
	}
	if in.Labels != nil {
		labels, err := parseUUIDList("labels", in.Labels)
		if err != nil {
			return nil, nil, err
		}
		patch.Labels = labels
	}

	res, err := s.plane.UpdateWorkItem2(ctx, patch, oas.UpdateWorkItem2Params{
		Slug:      workspace,
		ProjectID: projectID,
		Pk:        workitemID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: update work item: %w", err)
	}

	switch r := res.(type) {
	case *oas.Issue:
		return nil, r, nil
	case *oas.UpdateWorkItem2Unauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.UpdateWorkItem2Forbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.UpdateWorkItem2NotFound:
		return nil, nil, fmt.Errorf("plane: work item %q not found in project %q", workitemID, projectID)
	case *oas.UpdateWorkItem2BadRequest:
		return nil, nil, fmt.Errorf("plane: bad request updating work item")
	case *oas.UpdateWorkItem2Conflict:
		return nil, nil, fmt.Errorf("plane: conflict updating work item")
	default:
		return nil, nil, fmt.Errorf("plane: unexpected update work item response %T", res)
	}
}

func (s *Server) workitemDelete(ctx context.Context, _ *mcpsdk.CallToolRequest, in workitemDeleteInput) (*mcpsdk.CallToolResult, any, error) {
	workspace, projectID, workitemID, err := requireWorkspaceProjectWorkitem(in.Workspace, in.ProjectID, in.WorkitemID)
	if err != nil {
		return nil, nil, err
	}

	res, err := s.plane.DeleteWorkItem2(ctx, oas.DeleteWorkItem2Params{
		Slug:      workspace,
		ProjectID: projectID,
		Pk:        workitemID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("plane: delete work item: %w", err)
	}

	switch r := res.(type) {
	case *oas.DeleteWorkItem2NoContent:
		return nil, r, nil
	case *oas.DeleteWorkItem2Unauthorized:
		return nil, nil, fmt.Errorf("plane: unauthorized (check PLANE_API_KEY)")
	case *oas.DeleteWorkItem2Forbidden:
		return nil, nil, fmt.Errorf("plane: forbidden")
	case *oas.DeleteWorkItem2NotFound:
		return nil, nil, fmt.Errorf("plane: work item %q not found in project %q", workitemID, projectID)
	default:
		return nil, nil, fmt.Errorf("plane: unexpected delete work item response %T", res)
	}
}

func requireWorkspaceProjectWorkitem(workspaceRaw, projectIDRaw, workitemIDRaw string) (string, uuid.UUID, uuid.UUID, error) {
	workspace := strings.TrimSpace(workspaceRaw)
	if workspace == "" {
		return "", uuid.Nil, uuid.Nil, fmt.Errorf("workspace is required")
	}
	projectID, err := parseUUID("project_id", projectIDRaw)
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	workitemID, err := parseUUID("workitem_id", workitemIDRaw)
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	return workspace, projectID, workitemID, nil
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

func parseUUIDList(field string, raw []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(raw))
	for i, item := range raw {
		id, err := parseUUID(fmt.Sprintf("%s[%d]", field, i), item)
		if err != nil {
			return nil, fmt.Errorf("%s contains an invalid UUID: %w", field, err)
		}
		out = append(out, id)
	}
	return out, nil
}

func parsePriority(raw string) (oas.PriorityEnum, error) {
	p := oas.PriorityEnum(strings.TrimSpace(strings.ToLower(raw)))
	switch p {
	case oas.PriorityEnumUrgent, oas.PriorityEnumHigh, oas.PriorityEnumMedium, oas.PriorityEnumLow, oas.PriorityEnumNone:
		return p, nil
	default:
		return "", fmt.Errorf("priority must be one of urgent|high|medium|low|none")
	}
}
