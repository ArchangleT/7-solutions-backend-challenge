# AI-DLC Workflow Active — user-management-api

## ⚠️ MANDATORY — Read Every Turn

This project uses AI-DLC skills for specification and implementation. Follow the skill workflow — do NOT generate spec artifacts outside of it.

- **Manifest**: `.aidlc/workflow/user-management-api/aidlc-manifest.yaml`
- **Specs**: `.aidlc/specs/user-management-api/`
- **Workflow**: `.aidlc/workflow/user-management-api/`

## Available Skills

| Skill | Phase | What it does |
|---|---|---|
| aidlc-context | 1 | Workspace scan, context assessment, steering files |
| aidlc-requirements | 2 | User stories with EARS acceptance criteria |
| aidlc-decomposition | 3 | Unit breakdown with DDD boundaries and dependencies |
| aidlc-design | 4 | Technology decisions and architecture |
| aidlc-tasks | 5 | Implementation task planning with execution waves |
| aidlc-implement | 6 | Code generation following design specs |
| aidlc-prototype | — | Quick throwaway prototype to validate requirements |
| aidlc-reverse-engineer | — | Deep brownfield codebase analysis (13 reports) |
| aidlc-solutions-review | — | Cross-unit design review for conflicts and alignment |
| aidlc-code-review | — | Code review against design specs and best practices |

## Recovery (after context compaction)

1. Read `.aidlc/workflow/user-management-api/aidlc-manifest.yaml` for current state
2. Read steering files at `.opencode/steering/` for project context
3. Activate the **aidlc** orchestrator skill (`.opencode/skills/aidlc/SKILL.md`) and execute `resume` — this ensures proper routing, scope checks, unit dashboard, and template re-reading
4. The orchestrator will dispatch the correct phase skill based on manifest state

## Current State

- **Feature**: user-management-api
- **Language**: th
- **Specs**: `.aidlc/specs/user-management-api/`

## Implementation Context

When implementing tasks, read design documents first:
- `design.md` — architecture overview
- `design/components.md` — component specs
- `design/data-model.md` — entities and schemas
- `design/api-spec.md` — endpoints and contracts
- `design/integration.md` — external services
- `design/implementation.md` — directory structure and conventions
- `design/testing-strategy.md` — testing architecture and coverage mapping (if exists)

Follow technology stack and patterns from design decisions. Follow testing approach from D4 decisions.