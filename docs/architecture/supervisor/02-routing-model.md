# 02 Routing Model

## Routing Layers

The supervisor should output layered decisions, not a single `action`.

```mermaid
flowchart TD
    L0["L0 conversation intent"] --> L1["L1 primary/secondary domain"]
    L1 --> L2["L2 domain task intent"]
    L2 --> L3["L3 slots and execution flags"]
```

## Layer Definitions

### L0 Conversation Intent

- `consult`
- `clarify`
- `smalltalk`
- `meta_help`
- `switch_topic`

### L1 Domain

- `bazi`
- `qimen`
- `emotion`
- `career`
- `general`

### L2 Task Intent

Initial `bazi` tasks:

- `collect_profile`
- `amend_profile`
- `direct_bazi`
- `interpret_chart`
- `fortune_followup`
- `timing_followup`
- `cross_domain_consult`

### L3 Slots And Flags

- profile slots
- question text
- time scope
- target subject
- language
- `needs_clarification`
- `parallelizable`
- `needs_knowledge`
- `needs_qimen`
- `can_reuse_session_profile`

## Supervisor Decision Contract

```mermaid
classDiagram
    class SupervisorDecision {
      string conversation_intent
      string primary_domain
      string[] secondary_domains
      string task_intent
      bool needs_clarification
      string clarification_question
      bool parallelizable
      float confidence
      Slots slots
      PolicyHints policy_hints
    }

    class Slots {
      map profile
      string question_text
      string time_scope
      string target_subject
      string language
    }

    class PolicyHints {
      bool needs_knowledge
      bool needs_qimen
      bool can_reuse_session_profile
      bool can_reuse_cached_result
    }
```

## Execution Modes

```mermaid
flowchart TD
    S["SupervisorDecision"] --> M{"mode"}
    M -->|single-domain| A["primary specialist only"]
    M -->|primary + secondary| B["run primary then optional secondary"]
    M -->|parallel| C["fan out to multiple specialists"]
    A --> O["aggregate and answer"]
    B --> O
    C --> O
```

## Phase 1.5: Route-Driven Dispatch

Since phase 1.5, the `ApprovedRoute` (post-policy-gate) directly drives runtime execution via `executeRoute()`. The intermediate `action` string conversion has been removed from the supervisor path.

```mermaid
flowchart TD
    R["ApprovedRoute"] --> EX["executeRoute(route)"]
    EX -->|NeedsClarification| CL["executeClarificationRoute"]
    EX -->|collect_profile| CP["executeCollectProfileRoute"]
    EX -->|amend_profile| AP["executeAmendProfileRoute"]
    EX -->|direct_bazi| DB["executeDirectBaziRoute"]
    EX -->|timing / followup / default| FU["executeFollowupRoute"]
    CL --> H1["handleAsk | handleFollowupReading"]
    CP --> H2["handleAsk | handleFullReading"]
    AP --> H3["handleAsk | handleFullReading | handleFollowupReading"]
    DB --> H4["handleBaziInput"]
    FU --> H5["handleAsk | handleFullReading | handleFollowupReading (+qimen)"]
```

**Key principle:** `ApprovedRoute` fields (`TaskIntent`, `NeedsClarification`, `PolicyHints`) are now the primary control inputs. The `bridgeDecision()` function only extracts slot data (`profilePatch`, `questionText`, `needsQimen`, `rawBazi`) — it no longer decides execution flow.

## Mode Selection Rules

- Default to `single-domain`
- Use `primary + secondary` when the primary domain can determine whether secondary help is necessary
- Use `parallel` only when:
  - the user explicitly asks for a combined view, or
  - the domains are independent, and
  - required profile/context is already available, and
  - answer aggregation can remain coherent

## Router Strategy

The first implementation should use a layered router design, even if runtime calls are later merged for token efficiency.

```mermaid
flowchart LR
    C["conversation_router"] --> D["domain_router"]
    D --> T["task_router"]
```

This keeps the design debuggable even if an optimized implementation later compresses these into one structured call.
