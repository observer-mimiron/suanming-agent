# 08 Phase 1.5: Route-Driven Execution

**Date:** 2026-06-12
**Status:** Implemented, later simplified by the 2026-06-15 runtime package reset
**Tests:** 47 (5 new for phase 1.5)

## Motivation

Phase 1 introduced `ApprovedRoute` but the runtime still converted it back to legacy `action` strings via `bridgeDecision()`. This meant:

- The policy-gate-approved route was immediately discarded for execution
- The `switch action` block remained the sole dispatcher
- `bridgeDecision` was a control function masquerading as a compatibility shim

Phase 1.5 removed this indirection. On 2026-06-15, the follow-up runtime reset finished the job by deleting `bridgeDecision()` entirely and moving approved-route execution into `internal/runtime/`.

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
                         executeRoute(route, message)
                               ↓
                    switch route.TaskIntent {
                      case "collect_profile": executeCollectProfileRoute
                      case "amend_profile":   executeAmendProfileRoute
                      case "direct_bazi":     executeDirectBaziRoute
                      case "timing_followup": executeFollowupRoute(qimen=true)
                      default:                executeFollowupRoute
                    }
                    + route.NeedsClarification → executeClarificationRoute
```

## Key Components

### `executeRoute()` — runtime/candidate.go

The main dispatcher. Takes `policy.ApprovedRoute` plus the original user message. Routes by `NeedsClarification` first, then by `TaskIntent`.

```go
func (e *Executor) ExecuteRoute(ctx, sink, st, route, message) (turnType, assistantText, error)
```

### 5 Route Handlers

Each handler encapsulates: clone state → setup candidate → call handler → conditionally persist.

| Handler | TaskIntent | Behavior |
|---------|-----------|----------|
| `executeClarificationRoute` | `NeedsClarification=true` | Merge patch, ask or followup if chart exists |
| `executeCollectProfileRoute` | `collect_profile` | Reset profile/chart, ask or full reading |
| `executeAmendProfileRoute` | `amend_profile` | Merge patch (preserving chart), ask/full/followup |
| `executeDirectBaziRoute` | `direct_bazi` | Merge gender, direct bazi input |
| `executeFollowupRoute` | `interpret_chart`, `fortune_followup`, `timing_followup`, `cross_domain_consult`, default | Reuse chart or fall through to ask/full |

### Slot Consumption

The runtime now consumes slots directly from `route.Slots`:

- `profilePatch` from `route.Slots.Profile`
- `userQuestion` from `route.Slots.QuestionText` with message fallback
- `needsQimen` from `route.PolicyHints.NeedsQimen` plus task intent
- `rawBazi` from direct extraction inside the `direct_bazi` route path

### `Supervisor` Interface — orchestrator.go

```go
type Supervisor interface {
    Approve(ctx context.Context, msg string, st *state.SessionState) (policy.ApprovedRoute, error)
}
```

Defined locally in the orchestrator package (Go idiom: define interfaces where consumed). `*supervisor.Client` satisfies it. Tests use `mockSupervisor`.

## Reclassification Logic

A single reclassification check remains in `executeRoute()`: if `TaskIntent == "collect_profile"` but the session already has data and the message doesn't contain birth time, reclassify as `amend_profile`. This prevents wiping existing profile/chart on spurious collection intents.

## Runtime Boundary After Reset

As of 2026-06-15:

- `internal/orchestrator` owns turn lifecycle only
- `internal/runtime` owns approved-route execution
- `internal/specialists` consumes the shared `policy.ApprovedRoute` contract
- `bridgeDecision()` and the duplicated `specialists.ApprovedRoute` type are both removed

## Design Constraints Maintained

- No parallel fan-out (`ParallelAllowed` still hard-disabled in policy gate)
- No non-mingli domains enabled (allowlist: bazi, qimen)
- No changes to `internal/tools/`
- No changes to frontend SSE protocol
- No changes to specialist interfaces
- Session file format unchanged

## Files Changed

| File | Change |
|------|--------|
| `internal/orchestrator/orchestrator.go` | lifecycle shell only |
| `internal/runtime/executor.go` | approved-route execution entry |
| `internal/runtime/candidate.go` | route handlers |
| `internal/runtime/answer.go` | answer pipeline |
| `internal/runtime/bazi.go` / `qimen.go` / `ziwei.go` | domain execution lanes |
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
