# Plane MCP

Domain language for the multi-workspace Plane MCP server (Go). Glossary only — not a spec.

## Language

**Connection**:
A running MCP server session configured with a Plane API base URL and PAT. It is not locked to one Workspace.
_Avoid_: session workspace, current workspace

**Workspace**:
A Plane workspace identified by its slug. Every Tool takes an explicit `workspace` argument (the slug); the agent learns it from the user, not from Connection config or discovery.
_Avoid_: workspace_slug (as the Tool arg name), org, team

**Resource**:
A Plane entity family exposed through Tools (project, workitem, state).
_Avoid_: model, entity (when meaning a Tool family)

**Tool**:
One MCP tool named `{resource}_{action}` for a single Resource/action pair (e.g. `workitem_list`). Actions are `list`, `view`, `create`, `update`, `delete`.
_Avoid_: retrieve (use view), issue (use workitem), one fat resource tool with an `action` enum

**Workitem**:
A Plane work item (issue/task/epic). Tool prefix `workitem_`.
_Avoid_: issue, work_item

**State**:
A project workflow status. Supporting Resource in v0.1: enough to resolve UUIDs for workitem writes, not a full CRUD Surface.
_Avoid_: status (when meaning Plane state), first-class state CRUD (v0.1)
