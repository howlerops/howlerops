# AI Memory Architecture (Draft)

> Goal: give the SQL Assistant enough memory to handle long-form conversations, multi-step query refinement, and cross-session recall without overloading the model context window.

## Memory Tiers

| Tier | Purpose | Lifetime | Backing store |
|------|---------|----------|---------------|
| **Short-term** | Recent turns in the active chat (used for immediate context injection). | Current AI tab. | `useAIMemoryStore` (persisted in `localStorage`). |
| **Mid-term** | Summaries of exhausted chat windows plus high-signal user preferences. | Current desktop session. | Planned: shared SQLite / IndexedDB. |
| **Long-term** | Cross-session knowledge (team best practices, env metadata). | Until revoked. | Planned: backend `memory` service (PostgreSQL + embeddings). |

## Frontend Store (`useAIMemoryStore`)

### Features implemented

- Multi-session support with session metadata (title, timestamps).
- Message-level metadata (mode, provider, explanation, etc.).
- Token-aware context builder that trims older turns while preserving ordering.
- Session lifecycle helpers (`startNewSession`, `ensureActiveSession`, `resetActiveSession`).
- Persistence via `localStorage` (keeps the last N sessions per user).

### Planned enhancements

- Automatic summarisation when the message stack exceeds `maxContextTokens`.
- Session tagging (e.g. "fix", "generate", "analysis") to drive context selection.
- Embedding-based memory recall (vector store) for long-term repository search.

## Backend Responsibilities (next iterations)

1. **Memory Service**
   - Provide a gRPC/HTTP API for storing, retrieving, and pruning memories.
   - Support multiple providers (short/long-term) with configurable retention policies.
2. **Summarisation Pipelines**
   - Async summariser jobs that condense older batches into a short narrative.
   - Conflict resolution when a summary drops key fields (use structured metadata).
3. **Vector Index**
   - Embed high-value messages and allow similarity search during context assembly.
   - Candidate providers: pgvector, Qdrant, Milvus, or a serverless vector DB.

## Context Injection Strategy

When generating or fixing SQL, we build the `context` payload as:

1. Schema / multi-DB metadata (existing behaviour).
2. `<--- new --->`  
   `Conversation Memory` block generated from `useAIMemoryStore.buildContext(...)`  
   (includes rolling history + summary indicators).
3. System prompt adjustments (multi-DB rules, etc.).

By keeping the conversational memory separate we can:

- Swap in a summarised variant when the session is long.
- Switch between *fix* vs *generate* templates without losing thread history.
- Provide the backend with extra metadata (`provider`, `model`) for analytics.

## Next Up

1. **Summaries**: hook into the AI service to summarise conversation chunks once they leave the context window.
2. **Persistence** *(optional today via settings toggle)*: replicate `useAIMemoryStore` sessions into a lightweight backend store so multi-device users keep their history.
3. **Recall API** *(implemented)*: embed stored memories and surface the top-k matches for new prompts via `RecallAIMemorySessions`.
4. **UI**: surface a session switcher/history timeline inside the AI sheet so users can jump across conversations.

This scaffold gives us a working short-term memory today while leaving clear extension points for larger, shared memory systems tomorrow.
