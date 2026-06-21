# 08 Phase 1.5: Route-Driven Execution

**Date:** 2026-06-12
**Status:** Implemented, later simplified by the 2026-06-15 runtime package reset and tightened by the 2026-06-19 Phase 1 routing contract pass
**Tests:** route/runtime tests retained; supervisor/routing regressions extended over time

## Motivation

Phase 1 introduced `ApprovedRoute` but the runtime still converted it back to legacy `action` strings via `bridgeDecision()`. This meant:

- The policy-gate-approved route was immediately discarded for execution
- The `switch action` block remained the sole dispatcher
- `bridgeDecision` was a control function masquerading as a compatibility shim

Phase 1.5 removed this indirection. On 2026-06-15, the follow-up runtime reset finished the job by deleting `bridgeDecision()` entirely and moving approved-route execution into `internal/runtime/`. On 2026-06-19, a second tightening pass clarified which concerns live in prompt, policy, and runtime.

## Before / After

```
BEFORE (Phase 1):
  Supervisor → Policy Gate → ApprovedRoute
                               ↓
                         bridgeDecision()
                               ↓
                    (action, patch, question, ...)
                               ↓
                         switch action {
                           case "new_profile": ...
                           case "update_profile": ...
                           case "bazi_input": ...
                           case "incomplete": ...
                           case "followup": ...
                         }

AFTER (current):
  Supervisor → Policy Gate → ApprovedRoute
                               ↓
                         runtime.Execute(route, message)
                               ↓
                            preflight
                               ↓
                 short-circuit text OR agent main path
                               ↓
             Supervisor Agent + AgentTool specialists
                               ↓
                    post-run contract gate
                               ↓
                         final SSE output
```

## Key Components

### `Execute()` — runtime/executor.go

The main runtime entry. It merges extracted profile fields, writes the routing snapshot, runs deterministic preflight, executes the route-bound agent path, and only then emits the final answer after contract validation.

```go
func (e *Executor) Execute(ctx, sink, st, route, message) (turnType, assistantText, error)
```

### `preflight()` — runtime/preflight.go

The deterministic short-circuit layer. It handles:

- clarification routes
- missing-profile routes
- profile-required domain gates

### `runAgentRoute()` — runtime/executor.go

The main route-bound execution path:

- prepares session values
- performs safe reusable prefill for bazi only
- builds the supervisor agent plus allowed specialists
- runs the ADK loop
- captures tool results back into session state

### `guardFinalAnswer()` — runtime/final_guard.go

The Phase 1 contract gate added on 2026-06-19. It prevents final domain conclusions from being emitted if the corresponding required artifact was never produced.

Current checks:

- `primary_domain=qimen` requires `QimenResult`
- `primary_domain=ziwei` requires `ZiWeiResult`

### Slot Consumption

The runtime now consumes slots directly from `route.Slots`:

- profile patch from `route.Slots.Profile`
- user question from `route.Slots.QuestionText` with message fallback
- timing hints from `route.Slots.TimeScope`
- domain behavior from `route.PolicyHints`

### `Supervisor` Interface — orchestrator.go

```go
type Supervisor interface {
    Approve(ctx context.Context, msg string, st *state.SessionState) (policy.ApprovedRoute, error)
}
```

Defined locally in the orchestrator package (Go idiom: define interfaces where consumed). `*supervisor.Client` satisfies it. Tests use `mockSupervisor`.

## Reclassification Logic

Deterministic reclassification now lives in `normalizeApprovedRoute()`:

- `collect_profile -> amend_profile` when session data already exists
- `collect_profile -> fortune_followup` when chart exists and the message is not actually new birth-time info
- explicit method obey when the user clearly names `ziwei`, `qimen`, or `bazi`

## Runtime Boundary After Reset

As of 2026-06-15:

- `internal/orchestrator` owns turn lifecycle only
- `internal/runtime` owns approved-route execution
- `internal/specialists` consumes the shared `policy.ApprovedRoute` contract
- `bridgeDecision()` and the duplicated `specialists.ApprovedRoute` type are both removed

After 2026-06-19, the responsibility split is even sharper:

- **prompt** chooses the domain by capability framing
- **policy** only does deterministic correction and rollout safety
- **runtime** verifies the selected domain truly produced its required artifact

## Design Constraints Maintained

- No parallel fan-out (`ParallelAllowed` still hard-disabled in policy gate)
- No non-mingli domains enabled (allowlist: bazi, qimen, ziwei)
- No changes to `internal/tools/`
- No changes to frontend SSE protocol shape
- Specialist interfaces remain bounded by the shared runtime contract
- Session file format remains compatible; `RoutingSnapshot` now carries `QimenMode`

## Files Changed

| File | Change |
|------|--------|
| `internal/orchestrator/orchestrator.go` | lifecycle shell only |
| `internal/runtime/executor.go` | approved-route execution entry |
| `internal/runtime/preflight.go` | deterministic short-circuit gate |
| `internal/runtime/final_guard.go` | post-run contract gate |
| `internal/runtime/agent_route.go` | route-bound supervisor/specialist agent construction |
| `internal/supervisor/approved_route.go` | explicit-method deterministic normalization |
| `internal/orchestrator/orchestrator_test.go` | route/runtime regression coverage retained |

## Verification

```bash
go test ./... -v
# pass

go build ./cmd/server/
# pass

cd web && npm run build
# pass
```
