# plane-mcp

Multi-workspace Plane MCP server (Go) over **stdio**. A Connection is configured
with a Plane API base URL and PAT only — it is not locked to one Workspace.
Every Tool takes an explicit `workspace` slug.

## Connection config

| Setting | Flag | Environment | Default |
| --- | --- | --- | --- |
| Plane API base URL | `--plane-base-url` | `PLANE_BASE_URL` | `https://api.plane.so` |
| Plane PAT | _(none — env only)_ | `PLANE_API_KEY` | _(required)_ |

Precedence for the base URL: **flag > env > default**.

`PLANE_API_KEY` must be set in the environment. The process exits at start if it
is missing (no CLI flag, so the PAT is less likely to leak via process listings
or shell history).

Example (Cloud):

```bash
export PLANE_API_KEY=plane_api_...
plane-mcp
```

Example (self-hosted):

```bash
export PLANE_API_KEY=plane_api_...
plane-mcp --plane-base-url https://plane.example.com/api
```

## Build and run

```bash
go build -o plane-mcp .
export PLANE_API_KEY=...
./plane-mcp
```

With Nix:

```bash
nix build
nix run
```

## Development

```bash
nix develop          # Go toolchain + pre-commit hooks
go test ./...
go generate ./...    # regenerate internal/oas from internal/plane/openapi.yaml
```

Plane HTTP client code is generated with ogen from the vendored spectacular
OpenAPI SoT (pin `makeplane/plane` `5f7d927…` / v1.4.2). See
[`internal/plane/README.md`](./internal/plane/README.md).

## Tool inventory

| Tool | Required args | Optional args |
| --- | --- | --- |
| `project_list` | `workspace` | `cursor`, `per_page`, `order_by` |
| `workitem_list` | `workspace`, `project_id` | `cursor`, `per_page`, `order_by` |
| `workitem_view` | `workspace`, and either (`project_id` + `workitem_id`) or `identifier` (e.g. `PROJ-123`) | — |

`workitem_view` accepts dual id forms: UUID pair (`project_id` + `workitem_id`) or
a single human `identifier` like `PROJ-123` (split into project_identifier +
issue_identifier before the Plane call). No PQL/search on `workitem_list` in v0.1.

v0.1 will also register `project_{view,create,update,delete}`, `workitem_{create,update,delete}`,
and `state_list` (later issues).

Domain glossary: [`CONTEXT.md`](./CONTEXT.md).
