const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost/api";

interface ApiError {
  code: string;
  message: string;
  details?: Record<string, string>;
}

export class ApiRequestError extends Error {
  code: string;
  status: number;
  details?: Record<string, string>;

  constructor(status: number, error: ApiError) {
    super(error.message);
    this.code = error.code;
    this.status = status;
    this.details = error.details;
  }
}

let refreshPromise: Promise<void> | null = null;

async function refreshAccessToken(): Promise<void> {
  const res = await fetch(`${API_BASE}/v1/auth/refresh`, {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error("refresh failed");
  }
}

export async function api<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE}${path}`;

  const res = await fetch(url, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  // Handle 401: try refresh once (skip for auth endpoints — they return 401 for invalid credentials)
  const isAuthEndpoint = path.startsWith("/v1/auth/");
  if (res.status === 401 && !isAuthEndpoint) {
    try {
      if (!refreshPromise) {
        refreshPromise = refreshAccessToken();
      }
      await refreshPromise;
      refreshPromise = null;

      // Retry original request
      const retryRes = await fetch(url, {
        ...options,
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          ...options.headers,
        },
      });

      if (retryRes.status === 401) {
        // Second 401 — redirect to login
        if (typeof window !== "undefined") {
          window.location.href = "/login";
        }
        throw new ApiRequestError(401, { code: "UNAUTHORIZED", message: "Session expired" });
      }

      return handleResponse<T>(retryRes);
    } catch {
      refreshPromise = null;
      if (typeof window !== "undefined") {
        window.location.href = "/login";
      }
      throw new ApiRequestError(401, { code: "UNAUTHORIZED", message: "Session expired" });
    }
  }

  return handleResponse<T>(res);
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (res.status === 204) {
    return undefined as T;
  }

  const body = await res.json();

  if (!res.ok) {
    throw new ApiRequestError(res.status, body.error || { code: "UNKNOWN", message: "An error occurred" });
  }

  return body.data as T;
}
