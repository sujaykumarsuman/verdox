import type { PaginationMeta } from "@/types/repository";

export type ExecutionMode = "fork_gha";
export type TestRunStatus = "queued" | "running" | "passed" | "failed" | "cancelled";
export type TestResultStatus = "pass" | "fail" | "skip" | "error";

export interface TestSuite {
  id: string;
  repository_id: string;
  name: string;
  type: string;
  execution_mode: ExecutionMode;
  docker_image: string | null;
  test_command: string | null;
  gha_workflow_id: string | null;
  env_vars: Record<string, string>;
  config_path: string | null;
  timeout_seconds: number;
  workflow_config: WorkflowConfig;
  workflow_yaml: string | null;
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
  gha_run_id?: number;
  gha_run_url?: string;
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
  execution_mode: ExecutionMode;
  repository_id: string;
  repository_name: string;
  log_output?: string;
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

export interface WorkflowService {
  name: string;
  image: string;
  ports?: string[];
  env?: Record<string, string>;
}

export interface WorkflowStep {
  name: string;
  run?: string;
  uses?: string;
  with?: Record<string, string>;
}

export interface WorkflowMatrix {
  dimensions: Record<string, string[]>;
  fail_fast: boolean;
}

export interface WorkflowConcurrency {
  group: string;
  cancel_in_progress: boolean;
}

export interface WorkflowConfig {
  runner_os: string;
  custom_runner?: string;
  env_vars?: Record<string, string>;
  services?: WorkflowService[];
  setup_steps?: WorkflowStep[];
  matrix?: WorkflowMatrix | null;
  concurrency?: WorkflowConcurrency | null;
}

// --- Import Suite Types ---

export interface WorkflowFile {
  name: string;
  path: string;
}

export interface ListWorkflowFilesResponse {
  repository_id: string;
  files: WorkflowFile[];
}

export interface ImportSuiteResponse {
  name: string;
  type: string;
  timeout_seconds: number;
  workflow_yaml: string;
}

// --- Hierarchy Types (Phase 2) ---

export interface TestGroup {
  id: string;
  group_id: string;
  name: string;
  package: string | null;
  status: TestResultStatus;
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  duration_ms: number | null;
  pass_rate: number | null;
  created_at: string;
}

export interface TestCaseItem {
  id: string;
  case_id: string;
  name: string;
  status: TestResultStatus;
  duration_ms: number | null;
  error_message: string | null;
  stack_trace: string | null;
  retry_count: number;
  logs_url: string | null;
  created_at: string;
}

export interface RunSummaryV2 {
  total_jobs: number;
  total_cases: number;
  passed: number;
  failed: number;
  skipped: number;
  duration_ms: number;
  pass_rate: number;
}

export interface TestRunDetailV2 extends TestRunDetail {
  summary_v2?: RunSummaryV2 | null;
  groups?: TestGroup[];
  report_id?: string | null;
}

export interface GroupCasesResponse {
  cases: TestCaseItem[];
  meta: PaginationMeta;
}

export interface ReportResponse {
  report_id: string;
  runs: TestRun[];
}
