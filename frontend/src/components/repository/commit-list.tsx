"use client";

import { GitCommit } from "lucide-react";
import type { Commit } from "@/types/repository";

interface CommitListProps {
  commits: Commit[];
  isLoading?: boolean;
}

export function CommitList({ commits, isLoading }: CommitListProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <CommitSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (commits.length === 0) {
    return (
      <p className="text-[14px] text-text-secondary py-4">
        No commits found for this branch.
      </p>
    );
  }

  return (
    <div className="space-y-1">
      {commits.map((commit) => (
        <div
          key={commit.sha}
          className="flex items-start gap-3 rounded-[6px] px-3 py-2.5 hover:bg-bg-tertiary transition-colors"
        >
          <GitCommit className="h-4 w-4 text-text-secondary mt-0.5 shrink-0" />
          <div className="min-w-0 flex-1">
            <p className="text-[14px] text-text-primary truncate">
              {commit.message.split("\n")[0]}
            </p>
            <div className="flex items-center gap-2 mt-0.5">
              <code className="text-[12px] font-mono text-accent">
                {commit.sha.substring(0, 7)}
              </code>
              <span className="text-[12px] text-text-secondary">
                {commit.author}
              </span>
              <span className="text-[12px] text-text-secondary">
                {formatRelativeTime(commit.date)}
              </span>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function CommitSkeleton() {
  return (
    <div className="flex items-start gap-3 px-3 py-2.5">
      <div className="h-4 w-4 bg-bg-tertiary rounded animate-pulse mt-0.5" />
      <div className="flex-1 space-y-1.5">
        <div className="h-4 w-3/4 bg-bg-tertiary rounded animate-pulse" />
        <div className="flex gap-2">
          <div className="h-3 w-16 bg-bg-tertiary rounded animate-pulse" />
          <div className="h-3 w-20 bg-bg-tertiary rounded animate-pulse" />
          <div className="h-3 w-16 bg-bg-tertiary rounded animate-pulse" />
        </div>
      </div>
    </div>
  );
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 30) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}
