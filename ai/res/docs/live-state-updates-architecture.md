# CareerDock Live State Update Architecture

## Overview

CareerDock uses a **4-layer real-time update strategy** that combines push and pull mechanisms to create a stateful UI experience. The layers work together with graceful degradation — if SSE fails, polling kicks in; if polling is stale, optimistic updates provide instant feedback.

---

## Layer 1: Server-Sent Events (SSE) — Push from Backend

### Backend: Go + Redis Pub/Sub

**`backend/internal/handler/events.go`**

The SSE handler opens a persistent HTTP connection per authenticated user and forwards events from Redis pub/sub:

```
GET /api/events (authenticated, persistent connection)
```

Key mechanics:
- **User-specific Redis channels**: `sse:user:{userID}` — no cross-user leakage
- **`http.ResponseController`**: Extends write deadline (60s) before each write to bypass the server's 30s `WriteTimeout`
- **`http.Flusher`**: Unwraps middleware wrappers (Chi) to get access to chunked flush
- **Heartbeat**: 25-second ticker keeps the connection alive and detects dead clients
- **SSE headers**: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no` (disables Nginx buffering)

**Publishing from workers** (`backend/internal/worker/task_*.go`):

Workers call `PublishSSEEvent(redisClient, userID, eventType, data)` after completing async operations. This publishes a JSON payload to the user's Redis channel which the SSE handler forwards to the client.

Event types published:
| Event | Published By | When |
|-------|-------------|------|
| `resume_ready` | Resume parse worker | Resume parsing + general ATS scoring complete |
| `ats_company_complete` | ATS company worker | Company-specific ATS check done |
| `ats_job_complete` | ATS job worker | Job description ATS check done |
| `ats_resume_complete` | ATS resume worker | Resume-only ATS check done |
| `curated_list_complete` | Curate list worker | AI company ranking done |
| `credits_updated` | Admin service | Admin grants credits to user |

### Frontend: `useSSE()` Hook

**`frontend/src/hooks/use-sse.ts`**

A single `useSSE()` hook (initialized once in the auth provider) maintains the EventSource connection:

```typescript
const es = new EventSource(`${API_BASE}/api/events`, { withCredentials: true });
```

On each event, it **invalidates specific TanStack Query cache keys** — it doesn't handle the data directly, just tells React Query to refetch:

```typescript
es.addEventListener('resume_ready', () => {
  qc.invalidateQueries({ queryKey: queryKeys.resumes.all(userId) });
  qc.invalidateQueries({ queryKey: queryKeys.credits.balance(userId) });
});
```

Resilience features:
- **Exponential backoff reconnect**: 1s -> 2s -> 4s -> ... -> 30s max
- **Auth-aware**: Closes on logout, only reconnects if still authenticated
- **Cleanup on unmount**: Closes connection and clears retry timers

---

## Layer 2: Smart Polling — Pull with Dynamic Intervals

**`frontend/src/hooks/use-ats.ts`, `use-curated-lists.ts`, `use-payments.ts`, `use-notifications.ts`**

For resources with pending/processing states, hooks use React Query's `refetchInterval` that dynamically stops once the task completes:

```typescript
// ATS check: poll every 5s while pending, stop when done
refetchInterval: (query) => {
  const data = query.state.data;
  if (!data || !isATSComplete(data.result)) return 5_000;
  return false; // stop polling
}
```

Polling strategy by resource:
| Resource | Poll Interval | Stops When | SSE Backup? |
|----------|--------------|------------|-------------|
| ATS check detail | 5s | `result.score` exists | Yes |
| Curated list detail | 8s | Result is complete | Yes |
| Credit balance | 60s | Never (always polls) | Yes |
| Unread notification count | 30s | Never | No |

---

## Layer 3: Optimistic Updates — Instant UI Feedback on Mutations

**`frontend/src/hooks/use-applications.ts`, `use-lists.ts`, `use-admin.ts`**

For user-initiated mutations (status changes, entry updates), the UI patches the cache immediately then invalidates for eventual consistency:

```typescript
// use-applications.ts — useUpdateApplication()
onSuccess: (updatedApp) => {
  // STEP 1: Immediately patch every cached list (instant UI update)
  qc.setQueriesData<Application[]>(
    { queryKey: queryKeys.applications.all(userId) },
    (old) => old?.map((a) => (a.id === updatedApp.id ? updatedApp : a)),
  );
  // STEP 2: Background refetch for eventual consistency
  qc.invalidateQueries({ queryKey: queryKeys.applications.all(userId) });
};
```

This two-step pattern (`setQueriesData` + `invalidateQueries`) means:
1. UI updates **instantly** from the mutation response
2. Background refetch ensures cache stays consistent with server

---

## Layer 4: Hierarchical Cache Keys + Stale Time Strategy

**`frontend/src/lib/query-keys.ts`**

Cache keys are hierarchical, enabling targeted invalidation at any granularity:

```typescript
queryKeys.ats.all(userId)              // Invalidate ALL ats queries for user
queryKeys.ats.list(userId)             // Just the list
queryKeys.ats.detail(userId, checkId)  // Just one check
```

Stale time varies by data mutability:

```typescript
staleTimes = {
  atsResults: Infinity,     // Immutable once complete — never refetch
  curatedLists: Infinity,   // Immutable once complete
  resumes: 60_000,          // 60s — rarely changes
  userLists: 30_000,        // 30s — moderate change frequency
  credits: 0,               // Always refetch — can change via admin grant
  notifications: 0,         // Always fresh
}
```

---

## End-to-End Flow Example: Resume Upload

```
1. User uploads PDF
   └─ useUploadResume() mutation → POST /api/resumes
   └─ onSuccess: invalidateQueries(resumes.all, credits.balance)
   └─ UI shows new resume card with "Processing" spinner

2. Backend handler saves file to S3, creates DB record (status: parsing)
   └─ Enqueues Asynq task: resume:parse_and_score

3. Asynq worker picks up task
   └─ Parses resume with AI, runs general ATS scoring
   └─ Updates DB record (status: ready, parsed_data, ats_general)
   └─ Publishes SSE: redis.Publish("sse:user:{id}", resume_ready event)

4. SSE handler forwards event to client's EventSource connection

5. useSSE() receives resume_ready event
   └─ qc.invalidateQueries(resumes.all) — React Query refetches
   └─ UI re-renders: spinner replaced with score badge
```

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│  FRONTEND (Next.js)                                     │
│                                                         │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ useSSE() │  │ useQuery()   │  │ useMutation()    │  │
│  │ (push)   │  │ (poll/fetch) │  │ (optimistic)     │  │
│  └────┬─────┘  └──────┬───────┘  └────────┬─────────┘  │
│       │               │                    │            │
│       └───────┬───────┘                    │            │
│               ▼                            ▼            │
│  ┌──────────────────────────────────────────────────┐   │
│  │  TanStack Query Cache (queryKeys + staleTimes)   │   │
│  │  invalidateQueries / setQueriesData              │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────┬───────────────────────────┬───────────────┘
              │ EventSource               │ REST API
              │ GET /api/events           │ GET/POST/PUT
              ▼                           ▼
┌─────────────────────────────────────────────────────────┐
│  BACKEND (Go + Chi)                                     │
│                                                         │
│  ┌──────────────┐         ┌─────────────────────────┐   │
│  │ SSEHandler   │◄────────│ Redis Pub/Sub           │   │
│  │ (events.go)  │ subscribe│ channel: sse:user:{id} │   │
│  └──────────────┘         └────────────┬────────────┘   │
│                                        │ publish        │
│  ┌─────────────────────────────────────┴────────────┐   │
│  │  Asynq Workers (Redis-backed job queue)          │   │
│  │  ├─ task_resume_parse.go → resume_ready          │   │
│  │  ├─ task_ats_company.go  → ats_company_complete  │   │
│  │  ├─ task_ats_job.go      → ats_job_complete      │   │
│  │  ├─ task_ats_resume.go   → ats_resume_complete   │   │
│  │  └─ task_curate_list.go  → curated_list_complete │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

---

## Key Files Reference

| Component | Path |
|-----------|------|
| SSE backend handler | `backend/internal/handler/events.go` |
| SSE frontend hook | `frontend/src/hooks/use-sse.ts` |
| Query key factory | `frontend/src/lib/query-keys.ts` |
| API client (401 refresh) | `frontend/src/lib/api.ts` |
| Auth store (Zustand) | `frontend/src/store/auth-store.ts` |
| Providers (QueryClient) | `frontend/src/components/providers.tsx` |
| Application mutations | `frontend/src/hooks/use-applications.ts` |
| ATS polling + mutations | `frontend/src/hooks/use-ats.ts` |
| Resume worker (publishes SSE) | `backend/internal/worker/task_resume_parse.go` |
| Route registration | `backend/internal/handler/routes.go` |

---

## Prompt-Ready Summary for Another Project

To replicate this pattern, you need:

1. **SSE endpoint** (backend): Persistent HTTP connection per user, Redis pub/sub for event distribution, heartbeat to keep alive, ResponseController to bypass write timeouts
2. **Event publisher** (backend workers/services): After any async operation completes, publish typed event to user's Redis channel
3. **`useSSE()` hook** (frontend): Single EventSource connection with exponential backoff reconnect, maps event types to React Query cache invalidations
4. **Smart polling** (frontend): `refetchInterval` that dynamically stops when resource reaches terminal state — serves as fallback if SSE disconnects
5. **Optimistic mutations** (frontend): `setQueriesData` for instant UI patch + `invalidateQueries` for background consistency
6. **Hierarchical cache keys**: Enable invalidating at any granularity (all user data, all of one resource type, or single item)
7. **Stale time strategy**: Immutable results = Infinity, frequently changing = 0, moderate = 30-60s
