"use client";

import { Badge } from "@/components/ui/badge";
import type { TestRunStatus, TestResultStatus } from "@/types/test";

const runStatusMap: Record<
  TestRunStatus,
  { variant: "success" | "danger" | "warning" | "info" | "neutral"; label: string }
> = {
  passed: { variant: "success", label: "Passed" },
  failed: { variant: "danger", label: "Failed" },
  queued: { variant: "neutral", label: "Queued" },
  running: { variant: "info", label: "Running" },
  cancelled: { variant: "neutral", label: "Cancelled" },
};

const resultStatusMap: Record<
  TestResultStatus,
  { variant: "success" | "danger" | "warning" | "info" | "neutral"; label: string }
> = {
  pass: { variant: "success", label: "Pass" },
  fail: { variant: "danger", label: "Fail" },
  skip: { variant: "warning", label: "Skip" },
  error: { variant: "danger", label: "Error" },
};

export function RunStatusBadge({ status }: { status: TestRunStatus }) {
  const config = runStatusMap[status] || { variant: "neutral" as const, label: status };
  return <Badge variant={config.variant}>{config.label}</Badge>;
}

export function ResultStatusBadge({ status }: { status: TestResultStatus }) {
  const config = resultStatusMap[status] || { variant: "neutral" as const, label: status };
  return <Badge variant={config.variant}>{config.label}</Badge>;
}
