"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { api } from "@/lib/api";
import type {
  TestSuite,
  TestRun,
  TestRunDetail,
  TestRunListResponse,
  RunLogsResponse,
  DiscoveryResponse,
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

// --- Discovery ---

export async function discoverTests(
  repoId: string
): Promise<DiscoveryResponse> {
  return api<DiscoveryResponse>(`/v1/repositories/${repoId}/discover`, {
    method: "POST",
  });
}

export function useDiscovery(repoId: string) {
  const [discovery, setDiscovery] = useState<DiscoveryResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const scan = useCallback(async () => {
    if (!repoId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await discoverTests(repoId);
      setDiscovery(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Discovery failed"
      );
    } finally {
      setIsLoading(false);
    }
  }, [repoId]);

  return { discovery, isLoading, error, scan };
}
