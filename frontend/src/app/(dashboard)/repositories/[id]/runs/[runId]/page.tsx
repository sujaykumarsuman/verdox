"use client";

import { useState } from "react";
import { use } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  AlertCircle,
  GitBranch,
  GitCommit,
  Clock,
  XCircle,
} from "lucide-react";
import { useTestRunDetail, cancelRun } from "@/hooks/use-tests";
import { Button } from "@/components/ui/button";
import { Card, CardBody } from "@/components/ui/card";
import { RunStatusBadge } from "@/components/test/status-badge";
import { ResultRow } from "@/components/test/result-row";
import { LogViewer } from "@/components/test/log-viewer";
import { formatDate } from "@/lib/utils";

function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

export default function TestRunDetailPage({
  params,
}: {
  params: Promise<{ id: string; runId: string }>;
}) {
  const { id: repoId, runId } = use(params);
  const { run, isLoading, error, refetch } = useTestRunDetail(runId);
  const [cancelling, setCancelling] = useState(false);
  const [showLogs, setShowLogs] = useState(false);

  const handleCancel = async () => {
    setCancelling(true);
    try {
      await cancelRun(runId);
      refetch();
    } catch {
      // Error handled silently
    } finally {
      setCancelling(false);
    }
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="max-w-4xl">
        <div className="h-5 w-32 bg-bg-tertiary rounded animate-pulse mb-4" />
        <div className="h-8 w-64 bg-bg-tertiary rounded animate-pulse mb-2" />
        <div className="h-5 w-48 bg-bg-tertiary rounded animate-pulse mb-8" />
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((n) => (
            <div key={n} className="h-12 bg-bg-tertiary rounded animate-pulse" />
          ))}
        </div>
      </div>
    );
  }

  // Error state
  if (error || !run) {
    return (
      <div className="max-w-4xl">
        <Link
          href={`/repositories/${repoId}`}
          className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Repository
        </Link>
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-8 text-center">
          <AlertCircle className="h-10 w-10 text-danger mx-auto mb-3" />
          <h2 className="text-[18px] font-semibold text-text-primary mb-1">
            Test run not found
          </h2>
          <p className="text-[14px] text-text-secondary">
            {error || "The test run you're looking for doesn't exist."}
          </p>
        </div>
      </div>
    );
  }

  const isTerminal =
    run.status === "passed" || run.status === "failed" || run.status === "cancelled";

  return (
    <div className="max-w-4xl">
      {/* Breadcrumb */}
      <Link
        href={`/repositories/${repoId}`}
        className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        {run.repository_name || "Repository"}
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="font-display text-[28px] leading-[36px] tracking-[-0.01em] text-text-primary">
            {run.suite_name} — Run #{run.run_number}
          </h1>
          <div className="flex items-center gap-3 mt-2">
            <RunStatusBadge status={run.status} />
            <div className="flex items-center gap-1.5 text-[13px] text-text-secondary">
              <GitBranch className="h-3.5 w-3.5" />
              {run.branch}
            </div>
            <div className="flex items-center gap-1.5 text-[13px] text-text-secondary">
              <GitCommit className="h-3.5 w-3.5" />
              <code className="font-mono text-accent">
                {run.commit_hash.substring(0, 7)}
              </code>
            </div>
            {run.triggered_by_username && (
              <span className="text-[13px] text-text-secondary">
                by {run.triggered_by_username}
              </span>
            )}
            <span className="text-[13px] text-text-secondary">
              {formatDate(run.created_at)}
            </span>
            {run.gha_run_url && (
              <a
                href={run.gha_run_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-[13px] text-accent hover:underline"
              >
                View on GitHub
              </a>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {!isTerminal && (
            <Button
              variant="danger"
              size="sm"
              onClick={handleCancel}
              loading={cancelling}
            >
              <XCircle className="h-4 w-4 mr-1" />
              Cancel
            </Button>
          )}
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setShowLogs(!showLogs)}
          >
            {showLogs ? "Hide Logs" : "View Logs"}
          </Button>
        </div>
      </div>

      {/* Running indicator */}
      {!isTerminal && (
        <div className="rounded-[8px] border border-[var(--accent)]/30 bg-[var(--accent-subtle)] p-4 mb-6 flex items-center gap-3">
          <Loader2 className="h-5 w-5 text-accent animate-spin" />
          <p className="text-[14px] text-text-secondary">
            {run.status === "queued"
              ? "Waiting in queue..."
              : "Test run in progress. Results will appear as tests complete."}
          </p>
        </div>
      )}

      {/* Summary */}
      {run.summary && run.summary.total > 0 && (
        <Card className="mb-6">
          <CardBody className="flex items-center gap-6 py-3">
            <div className="text-center">
              <div className="text-[20px] font-semibold text-text-primary">
                {run.summary.total}
              </div>
              <div className="text-[12px] text-text-secondary">Total</div>
            </div>
            <div className="text-center">
              <div className="text-[20px] font-semibold text-[var(--success)]">
                {run.summary.passed}
              </div>
              <div className="text-[12px] text-text-secondary">Passed</div>
            </div>
            <div className="text-center">
              <div className="text-[20px] font-semibold text-[var(--danger)]">
                {run.summary.failed}
              </div>
              <div className="text-[12px] text-text-secondary">Failed</div>
            </div>
            {run.summary.skipped > 0 && (
              <div className="text-center">
                <div className="text-[20px] font-semibold text-[var(--warning)]">
                  {run.summary.skipped}
                </div>
                <div className="text-[12px] text-text-secondary">Skipped</div>
              </div>
            )}
            {run.summary.errors > 0 && (
              <div className="text-center">
                <div className="text-[20px] font-semibold text-[var(--danger)]">
                  {run.summary.errors}
                </div>
                <div className="text-[12px] text-text-secondary">Errors</div>
              </div>
            )}
            <div className="ml-auto flex items-center gap-1.5 text-[13px] text-text-secondary">
              <Clock className="h-3.5 w-3.5" />
              {formatDurationMs(run.summary.duration_ms)}
            </div>
          </CardBody>
        </Card>
      )}

      {/* Results */}
      {run.results && run.results.length > 0 && (
        <Card className="mb-6">
          <div className="divide-y divide-[var(--border)]">
            {run.results.map((result) => (
              <ResultRow key={result.id} result={result} />
            ))}
          </div>
        </Card>
      )}

      {/* Logs viewer */}
      {showLogs && <LogViewer runId={runId} />}
    </div>
  );
}
