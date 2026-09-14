package main

// Codegen tools are pinned in go.mod (tool directive) and invoked via go tool.
// OpenAPI SoT lives at internal/plane/openapi.yaml (beside, not inside, --clean).

//go:generate go tool ogen --target internal/oas --package oas --clean internal/plane/openapi.yaml
