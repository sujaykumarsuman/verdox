"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { Play, Loader2, CheckCircle2, Trash2, ArrowRight } from "lucide-react";
import { toast } from "sonner";
import { Card, CardBody } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { triggerRun, deleteTestSuite } from "@/hooks/use-tests";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import type { TestSuite, TestRun, TestRunDetailV2, TestRunListResponse, RunSummaryV2 } from "@/types/test";

interface SuiteCardProps {
  suite: TestSuite;
  latestRun?: TestRun | null;
  repoId: string;
  branch: string;
  commitHash: string;
  onRunTriggered: () => void;
  onDeleted: () => void;
}

// Circular gauge SVG
function CircularGauge({ passed, total, size = 44 }: { passed: number; total: number; size?: number }) {
  const hasData = total > 0;
  const pct = hasData ? passed / total : 0;
  const r = (size - 5) / 2;
  const circ = 2 * Math.PI * r;
  const offset = circ * (1 - pct);
  const color = !hasData ? "var(--border)" : passed === total ? "var(--success)" : "var(--danger)";

  return (
    <div className="relative" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--border)" strokeWidth={3.5} opacity={0.4} />
        {hasData && (
          <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke={color} strokeWidth={3.5}
            strokeDasharray={circ} strokeDashoffset={offset} strokeLinecap="round" />
        )}
      </svg>
      <div className="absolute inset-0 flex items-center justify-center">
        <span className={cn("font-semibold", hasData ? "text-[11px] text-text-primary" : "text-[10px] text-text-secondary")}>
          {hasData ? `${Math.round(pct * 100)}%` : "--"}
        </span>
      </div>
    </div>
  );
}

export function SuiteCard({
  suite,
  latestRun,
  repoId,
  branch,
  commitHash,
  onRunTriggered,
  onDeleted,
}: SuiteCardProps) {
  const [triggering, setTriggering] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [summary, setSummary] = useState<RunSummaryV2 | null>(null);

  // Fetch run detail for summary gauge (only for terminal runs).
  // If the latest run has no summary (e.g. setup failure), look at recent runs
  // to find one that does.
  const fetchSummary = useCallback(async () => {
    if (!latestRun || latestRun.status === "queued" || latestRun.status === "running") {
      setSummary(null);
      return;
    }
    try {
      // Try the latest run first
      const data = await api<TestRunDetailV2>(`/v1/runs/${latestRun.id}`);
      if (data.summary_v2) {
        setSummary(data.summary_v2);
        return;
      }
      // No summary on latest run — check recent runs for one with results
      const recent = await api<TestRunListResponse>(`/v1/suites/${suite.id}/runs?page=1&per_page=5`);
      if (recent.runs) {
        for (const run of recent.runs) {
          if (run.id === latestRun.id) continue;
          if (run.status !== "passed" && run.status !== "failed") continue;
          const detail = await api<TestRunDetailV2>(`/v1/runs/${run.id}`);
          if (detail.summary_v2) {
            setSummary(detail.summary_v2);
            return;
          }
        }
      }
    } catch { /* */ }
  }, [latestRun, suite.id]);

  useEffect(() => { fetchSummary(); }, [fetchSummary]);

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await deleteTestSuite(suite.id);
      toast.success("Test suite deleted");
      onDeleted();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete suite");
    } finally {
      setDeleting(false);
      setConfirmDelete(false);
    }
  };

  const handleTrigger = async () => {
    if (!branch || !commitHash) return;
    setTriggering(true);
    try {
      await triggerRun(suite.id, branch, commitHash);
      toast.success("Test run triggered");
      onRunTriggered();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to trigger run");
    } finally {
      setTriggering(false);
    }
  };

  const isActive = latestRun?.status === "queued" || latestRun?.status === "running";
  const sameCommit = latestRun?.commit_hash === commitHash && !!commitHash;
  const runDisabled = !branch || !commitHash || isActive || sameCommit;

  const statusColor = latestRun?.status === "passed"
    ? "border-l-[var(--success)]"
    : latestRun?.status === "failed"
      ? "border-l-[var(--danger)]"
      : isActive
        ? "border-l-[var(--accent)]"
        : "";

  return (
    <Card className={cn(latestRun && "border-l-[3px]", statusColor)}>
      <CardBody>
        {/* Row 1: Name + type + delete */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h4 className="text-[17px] font-semibold text-text-primary">{suite.name}</h4>
            <Badge variant="neutral">{suite.type}</Badge>
          </div>
          {!confirmDelete ? (
            <button
              onClick={() => setConfirmDelete(true)}
              className="text-text-secondary hover:text-[var(--danger)] transition-colors p-1"
              title="Delete suite"
            >
              <Trash2 size={15} />
            </button>
          ) : (
            <div className="flex items-center gap-1.5">
              <Button variant="danger" size="sm" onClick={handleDelete} loading={deleting}>Delete</Button>
              <Button variant="ghost" size="sm" onClick={() => setConfirmDelete(false)}>Cancel</Button>
            </div>
          )}
        </div>

        {/* Row 2: Gauge + stats (only when data available) */}
        <div className="flex items-center gap-4 mt-3">
          <CircularGauge
            passed={summary?.passed ?? 0}
            total={summary?.total_cases ?? 0}
          />
          <div className="flex-1 min-w-0">
            {summary ? (
              <div className="flex items-center gap-4 text-[13px]">
                <span className="text-text-secondary">{summary.total_jobs} jobs</span>
                <span className="text-[var(--success)]">{summary.passed} passed</span>
                {summary.failed > 0 && <span className="text-[var(--danger)]">{summary.failed} failed</span>}
                {summary.skipped > 0 && <span className="text-[var(--warning)]">{summary.skipped} skipped</span>}
                <span className="text-text-secondary">{summary.total_cases} cases</span>
              </div>
            ) : latestRun && isActive ? (
              <div className="flex items-center gap-2 text-[13px] text-text-secondary">
                <Loader2 size={13} className="animate-spin" />
                {latestRun.status === "queued" ? "Queued..." : "Running..."}
              </div>
            ) : (
              <span className="text-[13px] text-text-secondary">No results yet</span>
            )}
          </div>
        </div>

        {/* Row 3: Run info + actions */}
        <div className="flex items-center justify-between mt-3 pt-3 border-t border-[var(--border)]">
          <div className="flex items-center gap-2 text-[13px] text-text-secondary">
            {latestRun ? (
              <>
                <span>Run #{latestRun.run_number}</span>
                <span className="opacity-60">{latestRun.branch}</span>
              </>
            ) : (
              <span>No runs yet</span>
            )}
          </div>
          <div className="flex items-center gap-3">
            <Link
              href={`/repositories/${repoId}/suites/${suite.id}`}
              className="inline-flex items-center gap-1 text-[13px] text-accent hover:text-accent-light transition-colors"
            >
              View <ArrowRight size={12} />
            </Link>
            <Button
              size="sm"
              variant={isActive ? "secondary" : "primary"}
              onClick={handleTrigger}
              loading={triggering}
              disabled={runDisabled}
              title={
                isActive ? "A run is already in progress"
                  : sameCommit ? "This commit has already been tested"
                  : undefined
              }
            >
              {isActive ? (
                <><Loader2 size={14} className="mr-1 animate-spin" />{latestRun?.status === "queued" ? "Queued" : "Running"}</>
              ) : sameCommit ? (
                <><CheckCircle2 size={14} className="mr-1" />Tested</>
              ) : (
                <><Play size={14} className="mr-1" />Run</>
              )}
            </Button>
          </div>
        </div>
      </CardBody>
    </Card>
  );
}
