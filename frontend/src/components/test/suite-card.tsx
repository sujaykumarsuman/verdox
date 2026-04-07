"use client";

import { useState } from "react";
import { Play, Loader2, CheckCircle2, Trash2, ArrowRight } from "lucide-react";
import { toast } from "sonner";
import { Card, CardBody } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { RunStatusBadge } from "@/components/test/status-badge";
import { triggerRun, deleteTestSuite } from "@/hooks/use-tests";
import { cn } from "@/lib/utils";
import type { TestSuite, TestRun } from "@/types/test";

interface SuiteCardProps {
  suite: TestSuite;
  latestRun?: TestRun | null;
  repoId: string;
  branch: string;
  commitHash: string;
  onRunTriggered: () => void;
  onDeleted: () => void;
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
      <CardBody className="flex flex-col h-full">
        {/* Top: name + badges + delete */}
        <div className="flex items-start justify-between mb-auto">
          <div>
            <h4 className="font-semibold text-text-primary leading-tight">{suite.name}</h4>
            <div className="flex items-center gap-1.5 mt-1.5">
              <Badge variant="neutral">{suite.type}</Badge>
              <Badge variant="info">Fork GHA</Badge>
            </div>
          </div>
          {!confirmDelete ? (
            <button
              onClick={() => setConfirmDelete(true)}
              className="text-text-secondary hover:text-[var(--danger)] transition-colors p-1 -mt-0.5"
              title="Delete suite"
            >
              <Trash2 size={15} />
            </button>
          ) : (
            <div className="flex items-center gap-1.5">
              <Button variant="danger" size="sm" onClick={handleDelete} loading={deleting}>
                Delete
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setConfirmDelete(false)}>
                Cancel
              </Button>
            </div>
          )}
        </div>

        {/* Bottom: run status + action */}
        <div className="flex items-end justify-between mt-4 pt-3 border-t border-[var(--border)]">
          {/* Left: latest run info */}
          <div className="min-w-0">
            {latestRun ? (
              <div className="flex items-center gap-2 text-[13px] text-text-secondary">
                <RunStatusBadge status={latestRun.status} />
                <span className="whitespace-nowrap">Run #{latestRun.run_number}</span>
                {latestRun.branch && (
                  <span className="truncate max-w-[80px] text-[12px] opacity-70">{latestRun.branch}</span>
                )}
              </div>
            ) : (
              <p className="text-[13px] text-text-secondary">No runs yet</p>
            )}
          </div>

          {/* Right: action buttons */}
          <div className="flex items-center gap-2 shrink-0 ml-3">
            {latestRun && (
              <a
                href={`/repositories/${repoId}/runs/${latestRun.id}`}
                className="inline-flex items-center gap-1 text-[13px] text-accent hover:text-accent-light transition-colors"
              >
                View
                <ArrowRight size={12} />
              </a>
            )}
            <Button
              size="sm"
              variant={isActive ? "secondary" : "default"}
              onClick={handleTrigger}
              loading={triggering}
              disabled={runDisabled}
              title={
                isActive
                  ? "A run is already in progress"
                  : sameCommit
                    ? "This commit has already been tested"
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
