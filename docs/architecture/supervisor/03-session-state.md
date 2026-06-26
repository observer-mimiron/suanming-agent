# 03 Session State

> **Status:** Implemented
## Goal

Upgrade the session model from a single-domain reading state into a unified consulting state that can support multiple specialists while preserving the existing Go-owned persistence model.

## Target Shape

```mermaid
classDiagram
    class SessionState {
      string session_id
      ConversationState conversation
      RoutingState routing
      DomainState domains
      MemoryState memory
    }

    class ConversationState {
      string stage
      string current_topic
      string active_domain
      string last_user_question
      string last_clarification_question
    }

    class RoutingState {
      string conversation_intent
      string primary_domain
      string[] secondary_domains
      string task_intent
      bool awaiting_clarification
      bool parallel_allowed
      float confidence
    }

    class DomainState {
      BaziState bazi
      QimenState qimen
      map reserved_domains
    }

    class MemoryState {
      string running_summary
      Turn[] recent_turns
    }
```

## Domain State

Phase 1 only requires detailed state for `bazi` and `qimen`.

```mermaid
classDiagram
    class BaziState {
      map profile
      map chart_result
      bool profile_complete
      bool chart_ready
    }

    class QimenState {
      map last_question_context
      map last_chart_result
      string last_time_scope
    }
```

Future domains should not reuse `BaziResult`-shaped data blindly. Each domain owns its own result cache and reuse rules.

## State Ownership

```mermaid
flowchart TD
    G["Go runtime"] --> S["SessionState"]
    S --> C["conversation"]
    S --> R["routing"]
    S --> D["domains"]
    S --> M["memory"]
```

The LLM supervisor can suggest updates, but it does not own persistence. Go writes the canonical session.

## Migration Principle

Keep the existing session store mechanism and evolve the payload shape.

- keep file-backed session persistence
- keep recent turns and running summary
- introduce routing/domain substructures gradually
- keep backward compatibility where practical
