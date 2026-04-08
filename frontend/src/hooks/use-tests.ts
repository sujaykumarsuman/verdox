"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { api } from "@/lib/api";
import type {
  TestSuite,
  TestRun,
  TestRunDetail,
  TestRunListResponse,
  RunLogsResponse,
  ListWorkflowFilesResponse,
  GenerateSuiteResponse,
} from "@/types/test";
import type { PaginationMeta } from "@/types/repository";

// --- Suites ---

export function useTestSuites(repoId: string) {
  const [suites, setSuites] = useState<TestSuite[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSuites = useCallback(async () => {
    if (!repoId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<TestSuite[]>(
        `/v1/repositories/${repoId}/suites`
      );
      setSuites(data || []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load test suites"
      );
    } finally {
      setIsLoading(false);
    }
  }, [repoId]);

  useEffect(() => {
    fetchSuites();
  }, [fetchSuites]);

  return { suites, isLoading, error, refetch: fetchSuites };
}

export async function createTestSuite(
  repoId: string,
  data: {
    name: string;
    type: string;
    execution_mode?: string;
    docker_image?: string;
    test_command?: string;
    gha_workflow_id?: string;
    env_vars?: Record<string, string>;
    config_path?: string;
    timeout_seconds?: number;
  }
): Promise<TestSuite> {
  return api<TestSuite>(`/v1/repositories/${repoId}/suites`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateTestSuite(
  suiteId: string,
  data: Partial<{
    name: string;
    type: string;
    config_path: string;
    timeout_seconds: number;
    workflow_yaml: string;
  }>
): Promise<TestSuite> {
  return api<TestSuite>(`/v1/suites/${suiteId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteTestSuite(suiteId: string): Promise<void> {
  return api<void>(`/v1/suites/${suiteId}`, { method: "DELETE" });
}

// --- Latest Runs (per-suite, with polling) ---

export function useLatestRuns(suiteIds: string[], branch?: string) {
  const [latestRuns, setLatestRuns] = useState<Record<string, TestRun | null>>({});

  const fetchAll = useCallback(async () => {
    if (suiteIds.length === 0) return;
    const results: Record<string, TestRun | null> = {};
    const branchParam = branch ? `&branch=${encodeURIComponent(branch)}` : "";
    await Promise.all(
      suiteIds.map(async (sid) => {
        try {
          const data = await api<TestRunListResponse>(
            `/v1/suites/${sid}/runs?page=1&per_page=1${branchParam}`
          );
          results[sid] = data.runs && data.runs.length > 0 ? data.runs[0] : null;
        } catch {
          results[sid] = null;
        }
      })
    );
    setLatestRuns(results);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [suiteIds.join(","), branch]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  // Poll every 5s while any run is non-terminal (queued/running)
  useEffect(() => {
    const hasActive = Object.values(latestRuns).some(
      (r) => r && (r.status === "queued" || r.status === "running")
    );
    if (!hasActive) return;

    const interval = setInterval(fetchAll, 5000);
    return () => clearInterval(interval);
  }, [latestRuns, fetchAll]);

  return { latestRuns, refetch: fetchAll };
}

// --- Runs ---

export function useTestRuns(suiteId: string, page: number = 1) {
  const [runs, setRuns] = useState<TestRun[]>([]);
  const [meta, setMeta] = useState<PaginationMeta>({
    page: 1,
    per_page: 20,
    total: 0,
    total_pages: 0,
  });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRuns = useCallback(async () => {
    if (!suiteId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<TestRunListResponse>(
        `/v1/suites/${suiteId}/runs?page=${page}&per_page=20`
      );
      setRuns(data.runs || []);
      setMeta(data.meta);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load test runs"
      );
    } finally {
      setIsLoading(false);
    }
  }, [suiteId, page]);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  return { runs, meta, isLoading, error, refetch: fetchRuns };
}

export function useTestRunDetail(runId: string) {
  const [run, setRun] = useState<TestRunDetail | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchRun = useCallback(async () => {
    if (!runId) return;
    try {
      const data = await api<TestRunDetail>(`/v1/runs/${runId}`);
      setRun(data);
      setError(null);

      // Stop polling when terminal
      if (
        data.status === "passed" ||
        data.status === "failed" ||
        data.status === "cancelled"
      ) {
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load test run"
      );
    } finally {
      setIsLoading(false);
    }
  }, [runId]);

  useEffect(() => {
    fetchRun();
  }, [fetchRun]);

  // Poll every 3s while non-terminal
  useEffect(() => {
    if (
      !run ||
      run.status === "passed" ||
      run.status === "failed" ||
      run.status === "cancelled"
    ) {
      return;
    }

    intervalRef.current = setInterval(fetchRun, 3000);
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [run?.status, fetchRun]);

  return { run, isLoading, error, refetch: fetchRun };
}

export function useRunLogs(runId: string) {
  const [logs, setLogs] = useState<RunLogsResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchLogs = useCallback(async () => {
    if (!runId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<RunLogsResponse>(`/v1/runs/${runId}/logs`);
      setLogs(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load logs"
      );
    } finally {
      setIsLoading(false);
    }
  }, [runId]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  return { logs, isLoading, error, refetch: fetchLogs };
}

// --- Actions ---

export async function triggerRun(
  suiteId: string,
  branch: string,
  commitHash: string
): Promise<TestRun> {
  return api<TestRun>(`/v1/suites/${suiteId}/run`, {
    method: "POST",
    body: JSON.stringify({ branch, commit_hash: commitHash }),
  });
}

export async function cancelRun(
  runId: string
): Promise<{ id: string; status: string; message: string }> {
  return api(`/v1/runs/${runId}/cancel`, { method: "POST" });
}

export async function rerunRun(
  runId: string
): Promise<TestRun> {
  return api<TestRun>(`/v1/runs/${runId}/rerun`, { method: "POST" });
}

export async function runAllSuites(
  repoId: string,
  branch: string,
  commitHash: string
): Promise<{ message: string; runs: TestRun[] }> {
  return api(`/v1/repositories/${repoId}/run-all`, {
    method: "POST",
    body: JSON.stringify({ branch, commit_hash: commitHash }),
  });
}

// --- Generate Suite ---

export async function listWorkflowFiles(
  repoId: string
): Promise<ListWorkflowFilesResponse> {
  return api<ListWorkflowFilesResponse>(
    `/v1/repositories/${repoId}/workflows`
  );
}

export async function generateSuite(
  repoId: string,
  data: {
    workflow_file?: string;
    workflow_yaml?: string;
    model?: string;
    timeout_seconds?: number;
  },
  signal?: AbortSignal
): Promise<GenerateSuiteResponse> {
  return api<GenerateSuiteResponse>(
    `/v1/repositories/${repoId}/generate-suite`,
    {
      method: "POST",
      body: JSON.stringify(data),
      signal,
    }
  );
}
