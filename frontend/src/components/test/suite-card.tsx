"use client";

import { useState } from "react";
import { Play, Trash2 } from "lucide-react";
import { Card, CardBody } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { RunStatusBadge } from "@/components/test/status-badge";
import { triggerRun, deleteTestSuite } from "@/hooks/use-tests";
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
      onDeleted();
    } catch {
      // Error handled silently; user can retry
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
      onRunTriggered();
    } catch {
      // Error handled silently; user can retry
    } finally {
      setTriggering(false);
    }
  };

  return (
    <Card>
      <CardBody>
        <div className="flex items-start justify-between mb-3">
          <div>
            <h4 className="font-semibold text-text-primary">{suite.name}</h4>
            <div className="flex items-center gap-1.5 mt-1">
              <Badge variant="neutral">{suite.type}</Badge>
              <Badge variant={suite.execution_mode === "gha" ? "warning" : "info"}>
                {suite.execution_mode === "gha" ? "GHA" : "Container"}
              </Badge>
            </div>
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
              <Button variant="danger" size="sm" onClick={handleDelete} loading={deleting}>
                Delete
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setConfirmDelete(false)}>
                Cancel
              </Button>
            </div>
          )}
        </div>

        {latestRun ? (
          <div className="flex items-center gap-2 text-sm text-text-secondary mb-3">
            <RunStatusBadge status={latestRun.status} />
            <span>Run #{latestRun.run_number}</span>
            {latestRun.branch && (
              <span className="truncate max-w-[120px]">{latestRun.branch}</span>
            )}
          </div>
        ) : (
          <p className="text-sm text-text-secondary mb-3">No runs yet</p>
        )}

        <div className="flex items-center gap-2">
          <Button
            size="sm"
            onClick={handleTrigger}
            loading={triggering}
            disabled={!branch || !commitHash}
          >
            <Play size={14} className="mr-1" />
            Run
          </Button>
          {latestRun && (
            <a
              href={`/repositories/${repoId}/runs/${latestRun.id}`}
              className="text-sm text-accent hover:text-accent-light"
            >
              View Runs
            </a>
          )}
        </div>
      </CardBody>
    </Card>
  );
}
