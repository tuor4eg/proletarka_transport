# AGENTS.md

## Standing Authorization

This file is a standing user instruction for this repository.

The user explicitly authorizes and requests Codex to use local specialized agents whenever the routing rules below apply.

When in doubt, use an agent.

## Local Agent Source

Specialized agents are defined by files in the repository-local `.codex/agents/` directory.

Codex must treat these files as the source of truth for agent behavior:

- `.codex/agents/architect.md`
- `.codex/agents/implementer.md`
- `.codex/agents/reviewer.md`
- `.codex/agents/doc_writer.md`

When a routing rule requires a role, Codex must first read the corresponding file from `.codex/agents/` and follow those instructions for that role.

Do not create generic ad-hoc agents when a matching `.codex/agents/` role file exists. Reuse the local agent definition from `.codex/agents/` instead.

If a required `.codex/agents/` file is missing, Codex must say that the local agent definition is missing and then proceed with the role description from this `AGENTS.md` as a fallback.

## Mandatory Agent Routing

For every non-trivial task, Codex must route work through the specialized local agents defined in `.codex/agents/`.

Codex may work directly without a specialized agent only for:

- answering a simple question without code changes;
- making a tiny mechanical edit of 1-3 lines;
- running a simple command requested by the user;
- fixing an immediate syntax, type, or runtime error caused by Codex's own last change.

If Codex skips agent routing, the task must clearly fit one of these exceptions.

## Roles

The role names below map to local agent definition files in `.codex/agents/`.

- `architect` — `.codex/agents/architect.md` — structure, naming, data model, boundaries, feature design, route design, database shape.
- `implementer` — `.codex/agents/implementer.md` — code implementation, refactors, migrations, UI changes, query changes.
- `reviewer` — `.codex/agents/reviewer.md` — code review, business logic, edge cases, regressions, maintainability, validation of existing changes.
- `doc_writer` — `.codex/agents/doc_writer.md` — documentation, admin guides, UI wording, non-technical explanations.

Do not overload one agent with work that belongs to another.

## Required Routing Rules

### Feature Work

If the task adds or changes user-visible behavior, Codex must:

1. consult `architect` first, unless the change is tiny and purely mechanical;
2. use `implementer` for code changes;
3. review the result locally before responding.

### Data Model, Database, and Query Logic

If the task touches database schema, migrations, topic logic, list membership, filtering, counts, or relations, Codex must:

1. consult `architect`;
2. use `implementer`;
3. run relevant verification, such as lint, typecheck, and a targeted data/query check when possible.

### Bug Fixes

If the bug is non-trivial or affects business logic, Codex must:

1. inspect or reproduce the issue locally;
2. use `implementer` for the fix;
3. use `reviewer` or perform an explicit review pass before the final response.

Tiny bugs caused by Codex's immediately preceding edit may be fixed directly, but Codex must still explain the cause.

### Reviews and Validation

If the user asks to review, validate, check, audit, or assess existing code, Codex must use `reviewer`.

### Documentation and Wording

If the task is primarily documentation, admin instructions, explanatory text, or UI wording, Codex must use `doc_writer`.

## Delegation Expectations

When using agents:

- Use the matching local definition from `.codex/agents/` for the required role.
- Reuse an already-loaded role definition during the same task instead of inventing a new generic agent.
- Give each agent a narrow, concrete task.
- Do not ask one agent to do another role's work.
- Do not duplicate work between Codex and an agent.
- Codex remains responsible for final integration, verification, and the final answer.

## Scope

Keep changes focused.
Do not rewrite unrelated code or documentation unless the task requires it.

## Uncertainty

Do not invent behavior that is not confirmed by code, UI, data, or task context.
If something is unclear, state it directly and proceed with the most useful grounded result.

## Final Response

Codex must mention which agents were used and for what, unless the task was tiny enough to skip agent routing under the explicit exceptions above.