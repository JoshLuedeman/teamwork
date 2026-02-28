# Architecture Decision Records

## What Are ADRs?

Architecture Decision Records capture the **why** behind significant technical decisions. Each ADR documents the context, the decision made, and its consequences — creating a traceable history of project evolution.

**Why ADRs matter for agents:** Agents encounter decisions made before their session began. Without ADRs, an agent may undo or contradict a prior decision because it lacks context. ADRs give agents the rationale they need to work *with* the project's direction, not against it.

## When to Write an ADR

Write an ADR when a decision:
- Affects multiple components or workflows
- Constrains future technical choices
- Was chosen over a reasonable alternative
- Would confuse a future contributor who wasn't present for the discussion

## ADR Template

```markdown
# ADR-NNN: [Short Title]

**Status:** proposed | accepted | deprecated | superseded by ADR-NNN

**Date:** YYYY-MM-DD

## Context

What is the issue or force motivating this decision? Describe the situation
and constraints. Be specific — agents will rely on this to understand scope.

## Decision

State the decision clearly in one or two sentences, then elaborate if needed.
Use imperative tone: "We will..." or "The project uses..."

## Consequences

What becomes easier or harder as a result of this decision?

- **Positive:** [benefits]
- **Negative:** [trade-offs]
- **Neutral:** [side effects worth noting]

## Alternatives Considered

| Alternative | Why It Was Rejected |
|---|---|
| Option A | Brief reason |
| Option B | Brief reason |
```

## Example

Below is a sample ADR following the template above.

---

### ADR-001: Use Role-Based Agent Framework

**Status:** accepted

**Date:** 2025-01-01

#### Context

Projects using AI agents need structure to prevent agents from working at cross-purposes. Without clear roles, agents duplicate effort, make conflicting changes, and lack accountability. We need a framework that defines who does what, how work flows between agents, and what quality standards apply.

#### Decision

We will use a role-based agent framework where each agent operates under a defined role with explicit responsibilities, permissions, and quality bars. Roles are defined in guidance files that agents read at the start of a session. Work moves between agents through structured handoffs.

#### Consequences

- **Positive:** Agents have clear scope, reducing conflicts. Handoffs create natural review points. New agents onboard by reading their role file.
- **Negative:** Adds overhead for small projects where one agent could do everything. Role definitions require maintenance as the project evolves.
- **Neutral:** Humans interact with the same structure, making human-agent collaboration consistent.

#### Alternatives Considered

| Alternative | Why It Was Rejected |
|---|---|
| Unstructured agent access | Leads to conflicts and duplicated work without clear ownership |
| Single-agent monolith | Doesn't scale; limits specialization and parallel work |
| Task-queue only (no roles) | Lacks the context and constraints that roles provide for decision-making |
