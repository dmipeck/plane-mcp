package oas_test

import (
	"context"
	"testing"

	"github.com/dmipeck/plane-mcp/internal/oas"
)

// staticAPIKey satisfies oas.SecuritySource for construction tests.
type staticAPIKey struct {
	key string
}

func (s staticAPIKey) ApiKeyAuthentication(_ context.Context, _ oas.OperationName) (oas.ApiKeyAuthentication, error) {
	return oas.ApiKeyAuthentication{APIKey: s.key}, nil
}

func TestNewClient_Constructs(t *testing.T) {
	t.Parallel()

	client, err := oas.NewClient("https://api.plane.so", staticAPIKey{key: "test-pat"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient: got nil client")
	}

	var _ oas.Invoker = client
}

func TestInvoker_ExposesV01Surface(t *testing.T) {
	t.Parallel()

	// Compile-time surface for the eleven Tools (+ collateral CreateState on the
	// states collection path). Names come from spectacular operationIds at the
	// OpenAPI pin; _2 suffixes are work-items vs deprecated /issues/ twins.
	want := []oas.OperationName{
		oas.ListProjectsOperation,
		oas.CreateProjectOperation,
		oas.RetrieveProjectOperation,
		oas.UpdateProjectOperation,
		oas.DeleteProjectOperation,
		oas.ListWorkItems2Operation,
		oas.CreateWorkItem2Operation,
		oas.RetrieveWorkItem2Operation,
		oas.UpdateWorkItem2Operation,
		oas.DeleteWorkItem2Operation,
		oas.GetWorkspaceWorkItem2Operation,
		oas.ListStatesOperation,
		oas.CreateStateOperation, // collateral; not an MCP Tool in v0.1
	}
	if got := len(want); got != 13 {
		t.Fatalf("operation count = %d, want 13", got)
	}
}
