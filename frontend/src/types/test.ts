import type { PaginationMeta } from "@/types/repository";

export type TestType = "unit" | "integration";
export type TestRunStatus = "queued" | "running" | "passed" | "failed" | "cancelled";
export type TestResultStatus = "pass" | "fail" | "skip" | "error";

export interface TestSuite {
  id: string;
  repository_id: string;
  name: string;
  type: TestType;
  config_path: string | null;
  timeout_seconds: number;
  created_at: string;
  updated_at: string;
}

export interface TestRun {
  id: string;
  test_suite_id: string;
  triggered_by: string | null;
  triggered_by_username?: string;
  run_number: number;
  branch: string;
  commit_hash: string;
  status: TestRunStatus;
  started_at: string | null;
  finished_at: string | null;
  created_at: string;
}

export interface TestResult {
  id: string;
  test_name: string;
  status: TestResultStatus;
  duration_ms: number | null;
  error_message: string | null;
  created_at: string;
}

export interface RunSummary {
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  errors: number;
  duration_ms: number;
}

export interface TestRunDetail extends TestRun {
  suite_name: string;
  suite_type: string;
  repository_id: string;
  repository_name: string;
  summary: RunSummary | null;
  results: TestResult[];
}

export interface TestRunListResponse {
  runs: TestRun[];
  meta: PaginationMeta;
}

export interface RunLogEntry {
  test_name: string;
  status: string;
  duration_ms: number | null;
  log_output: string | null;
}

export interface RunLogsResponse {
  run_id: string;
  logs: RunLogEntry[];
}
