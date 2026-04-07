// Hierarchical cache key factory for TanStack Query / SWR-style invalidation.
// SSE event handlers invalidate at the appropriate granularity.

export const queryKeys = {
  notifications: {
    all: (userId: string) => ["notifications", userId] as const,
    unread: (userId: string) => ["notifications", userId, "unread"] as const,
    detail: (userId: string, id: string) => ["notifications", userId, id] as const,
  },
  admin: {
    banReviews: () => ["admin", "ban-reviews"] as const,
    stats: () => ["admin", "stats"] as const,
    users: (search: string, role: string, status: string, page: number) =>
      ["admin", "users", { search, role, status, page }] as const,
  },
  tests: {
    runs: (suiteId: string) => ["test-runs", suiteId] as const,
    runDetail: (runId: string) => ["test-run", runId] as const,
  },
} as const;
