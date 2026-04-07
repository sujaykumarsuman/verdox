const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost/api";

interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export class ApiRequestError extends Error {
  code: string;
  status: number;
  details?: Record<string, unknown>;

  constructor(status: number, error: ApiError) {
    super(error.message);
    this.code = error.code;
    this.status = status;
    this.details = error.details;
  }
}

let refreshPromise: Promise<void> | null = null;
let isRedirecting = false;

async function refreshAccessToken(): Promise<void> {
  const res = await fetch(`${API_BASE}/v1/auth/refresh`, {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error("refresh failed");
  }
}

function redirectToLogin() {
  if (typeof window === "undefined") return;
  // Prevent multiple redirects from racing API calls
  if (isRedirecting) return;
  isRedirecting = true;
  window.location.replace("/login");
}

export async function api<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE}${path}`;
  const isAuthEndpoint = path.startsWith("/v1/auth/");

  const res = await fetch(url, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  // Handle 401: try refresh once (skip for auth endpoints)
  if (res.status === 401 && !isAuthEndpoint) {
    try {
      if (!refreshPromise) {
        refreshPromise = refreshAccessToken();
      }
      await refreshPromise;
      refreshPromise = null;

      const retryRes = await fetch(url, {
        ...options,
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          ...options.headers,
        },
      });

      if (retryRes.status === 401) {
        redirectToLogin();
        throw new ApiRequestError(401, { code: "UNAUTHORIZED", message: "Session expired" });
      }

      return parseResponse<T>(retryRes);
    } catch {
      refreshPromise = null;
      redirectToLogin();
      throw new ApiRequestError(401, { code: "UNAUTHORIZED", message: "Session expired" });
    }
  }

  return parseResponse<T>(res);
}

async function parseResponse<T>(res: Response): Promise<T> {
  if (res.status === 204) {
    return undefined as T;
  }

  const body = await res.json();

  if (!res.ok) {
    throw new ApiRequestError(res.status, body.error || { code: "UNKNOWN", message: "An error occurred" });
  }

  return body.data as T;
}
