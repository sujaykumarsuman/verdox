"use client";

import { useState } from "react";
import { use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  ExternalLink,
  RefreshCw,
  Loader2,
  AlertCircle,
  Clock,
  TestTubes,
  Trash2,
  RotateCcw,
  GitCommit,
  GitBranch,
  GitFork,
  CheckCircle2,
  Sparkles,
} from "lucide-react";
import { toast } from "sonner";
import { useRepository, useBranches, useCommits, resyncRepository, deleteRepository } from "@/hooks/use-repos";
import { api } from "@/lib/api";
import { useTestSuites, useLatestRuns, triggerRun } from "@/hooks/use-tests";
import { Button } from "@/components/ui/button";
import { Card, CardBody } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { BranchSelector } from "@/components/repository/branch-selector";
import { SuiteCard } from "@/components/test/suite-card";
import { ImportSuiteDialog } from "@/components/test/import-suite-dialog";
import { cn } from "@/lib/utils";

export default function RepositoryDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { repo, isLoading: repoLoading, error: repoError, refetch: refetchRepo } = useRepository(id);
  const isForkReady = repo?.fork_status === "ready";
  const { branches, isLoading: branchesLoading } = useBranches(id, isForkReady);
  const router = useRouter();
  const [selectedBranch, setSelectedBranch] = useState<string>("");
  const [resyncing, setResyncing] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [runningAll, setRunningAll] = useState(false);
  const { suites, isLoading: suitesLoading, refetch: refetchSuites } = useTestSuites(id);
  const [showImportDialog, setShowImportDialog] = useState(false);
  const [forkingRepo, setForkingRepo] = useState(false);

  // Compute active branch early — needed by hooks below
  const activeBranch = selectedBranch || repo?.default_branch || "";
  const { commits } = useCommits(id, isForkReady ? activeBranch : "");
  const latestCommit = commits.length > 0 ? commits[0] : null;

  // Latest runs filtered by the currently selected branch
  const { latestRuns, refetch: refetchRuns } = useLatestRuns(suites.map((s) => s.id), activeBranch);

  const handleForkSetup = async () => {
    setForkingRepo(true);
    try {
      await api(`/v1/repositories/${id}/fork`, { method: "POST" });
      toast.success("Fork setup initiated — this may take a moment");
      refetchRepo();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to setup fork");
    } finally {
      setForkingRepo(false);
    }
  };

  const handleResync = async () => {
    setResyncing(true);
    try {
      await resyncRepository(id);
      toast.success("Repository re-synced");
      refetchRepo();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to re-sync");
    } finally {
      setResyncing(false);
    }
  };

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await deleteRepository(id);
      toast.success("Repository deleted");
      router.push("/dashboard");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete repository");
      setDeleting(false);
    }
  };

  // Loading state
  if (repoLoading) {
    return (
      <div className="max-w-6xl">
        <div className="h-5 w-32 bg-bg-tertiary rounded animate-pulse mb-4" />
        <div className="h-8 w-64 bg-bg-tertiary rounded animate-pulse mb-2" />
        <div className="h-5 w-48 bg-bg-tertiary rounded animate-pulse mb-8" />
        <div className="h-24 bg-bg-tertiary rounded animate-pulse mb-6" />
        <div className="h-48 bg-bg-tertiary rounded animate-pulse" />
      </div>
    );
  }

  // Error state
  if (repoError || !repo) {
    return (
      <div className="max-w-6xl">
        <Link
          href="/dashboard"
          className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Dashboard
        </Link>
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-8 text-center">
          <AlertCircle className="h-10 w-10 text-danger mx-auto mb-3" />
          <h2 className="text-[18px] font-semibold text-text-primary mb-1">
            Repository not found
          </h2>
          <p className="text-[14px] text-text-secondary">
            {repoError || "The repository you're looking for doesn't exist or you don't have access."}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl">
      {/* Breadcrumb */}
      <Link
        href="/dashboard"
        className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        Repositories
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="font-display text-[28px] leading-[36px] tracking-[-0.01em] text-text-primary">
            {repo.name}
          </h1>
          <div className="flex items-center gap-3 mt-1">
            <a
              href={`https://github.com/${repo.github_full_name}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-[14px] text-accent hover:underline"
            >
              {repo.github_full_name}
              <ExternalLink className="h-3.5 w-3.5" />
            </a>
            <ForkStatusBadge status={repo.fork_status} forkFullName={repo.fork_full_name} />
          </div>
          {repo.description && (
            <p className="text-[14px] text-text-secondary mt-2">
              {repo.description}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {isForkReady && (
            <Button
              variant="secondary"
              size="sm"
              onClick={handleResync}
              loading={resyncing}
            >
              <RefreshCw className={cn("h-4 w-4", resyncing && "animate-spin")} />
              Re-sync
            </Button>
          )}
          {!showDeleteConfirm ? (
            <Button
              variant="danger"
              size="sm"
              onClick={() => setShowDeleteConfirm(true)}
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </Button>
          ) : (
            <div className="flex items-center gap-2">
              <span className="text-[13px] text-danger">Are you sure?</span>
              <Button variant="danger" size="sm" onClick={handleDelete} loading={deleting}>
                Yes, delete
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setShowDeleteConfirm(false)}>
                Cancel
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Fork status banners — only show for non-ready states */}
      {repo.fork_status === "forking" && (
        <div className="rounded-[8px] border border-accent/30 bg-accent/5 p-4 mb-6 flex items-center gap-3">
          <Loader2 className="h-5 w-5 text-accent animate-spin" />
          <p className="text-[14px] text-text-secondary">
            Setting up fork... This may take a moment.
          </p>
        </div>
      )}
      {repo.fork_status === "failed" && (
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-4 mb-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <AlertCircle className="h-5 w-5 text-danger" />
            <p className="text-[14px] text-danger">Fork setup failed.</p>
          </div>
          <Button variant="secondary" size="sm" onClick={handleForkSetup} loading={forkingRepo}>
            <RotateCcw className="h-4 w-4" />
            Retry
          </Button>
        </div>
      )}
      {repo.fork_status === "none" && (
        <div className="rounded-[8px] border p-4 mb-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <GitFork className="h-5 w-5 text-text-secondary" />
            <p className="text-[14px] text-text-secondary">
              No fork configured. Fork this repo to use GitHub Actions testing.
            </p>
          </div>
          <Button variant="secondary" size="sm" onClick={handleForkSetup} loading={forkingRepo}>
            <GitFork className="h-4 w-4" />
            Fork &amp; Setup
          </Button>
        </div>
      )}

      {/* Repository info bar: branch selector + latest commit */}
      {isForkReady && (
        <Card className="mb-6">
          <CardBody className="flex items-center justify-between gap-4 py-3">
            <div className="flex items-center gap-4">
              <BranchSelector
                branches={branches}
                selected={activeBranch}
                onSelect={setSelectedBranch}
                isLoading={branchesLoading}
                defaultBranch={repo.default_branch}
              />
              <div className="flex items-center gap-1.5 text-[13px] text-text-secondary">
                <GitBranch className="h-3.5 w-3.5" />
                <span>{branches.length} branch{branches.length !== 1 ? "es" : ""}</span>
              </div>
            </div>
            {latestCommit && (
              <div className="flex items-center gap-2 text-[13px] text-text-secondary min-w-0">
                <GitCommit className="h-3.5 w-3.5 shrink-0" />
                <code className="font-mono text-accent">{latestCommit.sha.substring(0, 7)}</code>
                <span className="truncate max-w-[300px]">{latestCommit.message.split("\n")[0]}</span>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {/* Test suites — primary content area */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-[16px] font-semibold text-text-primary">
            Test Suites
          </h3>
          <div className="flex items-center gap-2">
            {isForkReady && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowImportDialog(true)}
              >
                <Sparkles className="h-3.5 w-3.5" />
                Generate Suite
              </Button>
            )}
            {suites.length > 0 && isForkReady && latestCommit && (() => {
              // Suites that can be triggered: not active, not already tested this commit
              const runnableSuites = suites.filter((s) => {
                const run = latestRuns[s.id];
                if (run && (run.status === "queued" || run.status === "running")) return false;
                if (run && run.commit_hash === latestCommit.sha) return false;
                return true;
              });

              return (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={async () => {
                    setRunningAll(true);
                    try {
                      let triggered = 0;
                      await Promise.all(
                        runnableSuites.map(async (s) => {
                          try {
                            await triggerRun(s.id, activeBranch, latestCommit.sha);
                            triggered++;
                          } catch (err) {
                            toast.error(`${s.name}: ${err instanceof Error ? err.message : "Failed"}`);
                          }
                        })
                      );
                      if (triggered > 0) {
                        toast.success(`${triggered} test suite${triggered > 1 ? "s" : ""} triggered`);
                      }
                      refetchSuites();
                      refetchRuns();
                    } catch (err) {
                      toast.error(err instanceof Error ? err.message : "Failed to run suites");
                    }
                    setRunningAll(false);
                  }}
                  loading={runningAll}
                  disabled={runnableSuites.length === 0}
                  title={
                    runnableSuites.length === 0
                      ? "All suites already tested or in progress"
                      : `Run ${runnableSuites.length} untested suite${runnableSuites.length > 1 ? "s" : ""}`
                  }
                >
                  Run All{runnableSuites.length < suites.length && runnableSuites.length > 0
                    ? ` (${runnableSuites.length})`
                    : ""}
                </Button>
              );
            })()}
            {isForkReady && (
              <Button size="sm" onClick={() => router.push(`/repositories/${id}/suites/new`)}>
                Create Suite
              </Button>
            )}
          </div>
        </div>

        {suitesLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[1, 2].map((n) => (
              <Card key={n}>
                <CardBody>
                  <Skeleton width="60%" height="20px" />
                  <Skeleton width="40%" height="16px" />
                  <Skeleton width="80%" height="16px" />
                </CardBody>
              </Card>
            ))}
          </div>
        ) : suites.length === 0 ? (
          <div className="rounded-[8px] border border-dashed bg-bg-secondary p-12 text-center">
            <TestTubes className="h-10 w-10 text-text-secondary mx-auto mb-3" />
            <h4 className="text-[15px] font-medium text-text-primary mb-1">
              No test suites configured
            </h4>
            <p className="text-[14px] text-text-secondary mb-4">
              Create your first test suite to start running tests against this repository.
            </p>
            {isForkReady && (
              <Button onClick={() => router.push(`/repositories/${id}/suites/new`)}>
                Create Suite
              </Button>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {suites.map((suite) => (
              <SuiteCard
                key={suite.id}
                suite={suite}
                latestRun={latestRuns[suite.id] || null}
                repoId={id}
                branch={activeBranch}
                commitHash={latestCommit?.sha || ""}
                onRunTriggered={() => { refetchSuites(); refetchRuns(); }}
                onDeleted={refetchSuites}
              />
            ))}
          </div>
        )}
      </div>

      <ImportSuiteDialog
        repoId={id}
        open={showImportDialog}
        onClose={() => setShowImportDialog(false)}
      />
    </div>
  );
}

function ForkStatusBadge({ status, forkFullName }: { status: string; forkFullName?: string | null }) {
  const config: Record<string, { bg: string; text: string; label: string }> = {
    ready: { bg: "bg-green-500/10", text: "text-green-600", label: "Fork Ready" },
    none: { bg: "bg-gray-500/10", text: "text-gray-600", label: "No Fork" },
    forking: { bg: "bg-yellow-500/10", text: "text-yellow-600", label: "Forking" },
    failed: { bg: "bg-red-500/10", text: "text-red-600", label: "Fork Failed" },
  };

  const c = config[status] || config.none;

  if (status === "ready" && forkFullName) {
    return (
      <a
        href={`https://github.com/${forkFullName}`}
        target="_blank"
        rel="noopener noreferrer"
        className={cn("inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[12px] font-medium hover:opacity-80 transition-opacity", c.bg, c.text)}
      >
        <ExternalLink className="h-3 w-3" />
        Fork
      </a>
    );
  }

  return (
    <span className={cn("inline-flex items-center px-2 py-0.5 rounded-full text-[12px] font-medium", c.bg, c.text)}>
      {c.label}
    </span>
  );
}
