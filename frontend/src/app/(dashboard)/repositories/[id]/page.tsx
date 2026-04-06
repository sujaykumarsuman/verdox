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
} from "lucide-react";
import { useRepository, useBranches, useCommits, resyncRepository, retryClone, deleteRepository } from "@/hooks/use-repos";
import { useTestSuites, runAllSuites } from "@/hooks/use-tests";
import { Button } from "@/components/ui/button";
import { Card, CardBody } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { BranchSelector } from "@/components/repository/branch-selector";
import { SuiteCard } from "@/components/test/suite-card";
import { CreateSuiteDialog } from "@/components/test/create-suite-dialog";
import { cn } from "@/lib/utils";

export default function RepositoryDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { repo, isLoading: repoLoading, error: repoError, refetch: refetchRepo } = useRepository(id);
  const isCloneReady = repo?.clone_status === "ready";
  const { branches, isLoading: branchesLoading } = useBranches(id, isCloneReady);
  const router = useRouter();
  const [selectedBranch, setSelectedBranch] = useState<string>("");
  const [resyncing, setResyncing] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showCreateSuite, setShowCreateSuite] = useState(false);
  const [runningAll, setRunningAll] = useState(false);
  const { suites, isLoading: suitesLoading, refetch: refetchSuites } = useTestSuites(id);

  // Auto-select default branch when branches load
  const activeBranch = selectedBranch || repo?.default_branch || "";
  const { commits } = useCommits(id, isCloneReady ? activeBranch : "");
  const latestCommit = commits.length > 0 ? commits[0] : null;

  const handleResync = async () => {
    setResyncing(true);
    try {
      await resyncRepository(id);
      refetchRepo();
    } catch {
      // Error handled by toast in future
    } finally {
      setResyncing(false);
    }
  };

  const handleRetryClone = async () => {
    setRetrying(true);
    try {
      await retryClone(id);
      refetchRepo();
    } catch {
      // Error handled by toast in future
    } finally {
      setRetrying(false);
    }
  };

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await deleteRepository(id);
      router.push("/dashboard");
    } catch {
      setDeleting(false);
    }
  };

  // Loading state
  if (repoLoading) {
    return (
      <div className="max-w-4xl">
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
      <div className="max-w-4xl">
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
    <div className="max-w-4xl">
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
            <CloneStatusBadge status={repo.clone_status} />
          </div>
          {repo.description && (
            <p className="text-[14px] text-text-secondary mt-2">
              {repo.description}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {isCloneReady && (
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

      {/* Clone status banner */}
      {repo.clone_status === "pending" && (
        <div className="rounded-[8px] border border-yellow-300/30 bg-yellow-50/5 p-4 mb-6 flex items-center gap-3">
          <Clock className="h-5 w-5 text-yellow-500" />
          <p className="text-[14px] text-text-secondary">
            Clone is pending. The repository will be cloned shortly.
          </p>
        </div>
      )}
      {repo.clone_status === "cloning" && (
        <div className="rounded-[8px] border border-yellow-300/30 bg-yellow-50/5 p-4 mb-6 flex items-center gap-3">
          <Loader2 className="h-5 w-5 text-yellow-500 animate-spin" />
          <p className="text-[14px] text-text-secondary">
            Cloning repository... This may take a minute.
          </p>
        </div>
      )}
      {repo.clone_status === "failed" && (
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-4 mb-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <AlertCircle className="h-5 w-5 text-danger shrink-0" />
            <p className="text-[14px] text-danger">
              Clone failed. Check that the team PAT has access to this repository.
            </p>
          </div>
          <Button
            variant="secondary"
            size="sm"
            onClick={handleRetryClone}
            loading={retrying}
          >
            <RotateCcw className="h-4 w-4" />
            Retry
          </Button>
        </div>
      )}

      {/* Repository info bar: branch selector + latest commit */}
      {isCloneReady && (
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
            {suites.length > 0 && isCloneReady && latestCommit && (
              <Button
                variant="secondary"
                size="sm"
                onClick={async () => {
                  setRunningAll(true);
                  try {
                    await runAllSuites(id, activeBranch, latestCommit.sha);
                    refetchSuites();
                  } catch {}
                  setRunningAll(false);
                }}
                loading={runningAll}
              >
                Run All
              </Button>
            )}
            {isCloneReady && (
              <Button size="sm" onClick={() => setShowCreateSuite(true)}>
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
            {isCloneReady && (
              <Button onClick={() => setShowCreateSuite(true)}>
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
                repoId={id}
                branch={activeBranch}
                commitHash={latestCommit?.sha || ""}
                onRunTriggered={refetchSuites}
              />
            ))}
          </div>
        )}
      </div>

      <CreateSuiteDialog
        repoId={id}
        open={showCreateSuite}
        onClose={() => setShowCreateSuite(false)}
        onSuccess={refetchSuites}
      />
    </div>
  );
}

function CloneStatusBadge({ status }: { status: string }) {
  const config: Record<string, { bg: string; text: string; label: string }> = {
    ready: { bg: "bg-green-500/10", text: "text-green-600", label: "Cloned" },
    pending: { bg: "bg-yellow-500/10", text: "text-yellow-600", label: "Pending" },
    cloning: { bg: "bg-yellow-500/10", text: "text-yellow-600", label: "Cloning" },
    failed: { bg: "bg-red-500/10", text: "text-red-600", label: "Failed" },
    evicted: { bg: "bg-gray-500/10", text: "text-gray-600", label: "Evicted" },
  };

  const c = config[status] || config.pending;

  return (
    <span className={cn("inline-flex items-center px-2 py-0.5 rounded-full text-[12px] font-medium", c.bg, c.text)}>
      {c.label}
    </span>
  );
}
