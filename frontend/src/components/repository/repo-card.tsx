"use client";

import Link from "next/link";
import { GitBranch, Clock } from "lucide-react";
import { Card, CardBody } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { Repository, ForkStatus } from "@/types/repository";

const statusConfig: Record<ForkStatus, { color: string; label: string; pulse?: boolean }> = {
  ready: { color: "bg-green-500", label: "Fork Ready" },
  none: { color: "bg-gray-400", label: "No Fork" },
  forking: { color: "bg-yellow-500", label: "Forking", pulse: true },
  failed: { color: "bg-red-500", label: "Fork Failed" },
};

interface RepoCardProps {
  repo: Repository;
}

export function RepoCard({ repo }: RepoCardProps) {
  const status = statusConfig[repo.fork_status];

  return (
    <Link href={`/repositories/${repo.id}`}>
      <Card hoverable className="h-full cursor-pointer">
        <CardBody className="flex flex-col gap-3">
          {/* Header */}
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              <h3 className="text-[16px] font-semibold text-text-primary truncate">
                {repo.name}
              </h3>
              <p className="text-[13px] text-text-secondary truncate">
                {repo.github_full_name}
              </p>
            </div>
            <div className="flex items-center gap-1.5 shrink-0">
              <span
                className={cn(
                  "h-2 w-2 rounded-full",
                  status.color,
                  status.pulse && "animate-pulse"
                )}
              />
              <span className="text-[12px] text-text-secondary">{status.label}</span>
            </div>
          </div>

          {/* Description */}
          {repo.description && (
            <p className="text-[14px] text-text-secondary line-clamp-2">
              {repo.description}
            </p>
          )}

          {/* Footer */}
          <div className="flex items-center gap-4 mt-auto pt-2 border-t border-border">
            <div className="flex items-center gap-1.5 text-[13px] text-text-secondary">
              <GitBranch className="h-3.5 w-3.5" />
              <span>{repo.default_branch}</span>
            </div>
            <div className="flex items-center gap-1.5 text-[13px] text-text-secondary">
              <Clock className="h-3.5 w-3.5" />
              <span>{new Date(repo.updated_at).toLocaleDateString()}</span>
            </div>
          </div>
        </CardBody>
      </Card>
    </Link>
  );
}

export function RepoCardSkeleton() {
  return (
    <Card>
      <CardBody className="flex flex-col gap-3">
        <div className="flex items-start justify-between">
          <div className="space-y-2 flex-1">
            <div className="h-5 w-32 bg-bg-tertiary rounded animate-pulse" />
            <div className="h-4 w-48 bg-bg-tertiary rounded animate-pulse" />
          </div>
          <div className="h-4 w-16 bg-bg-tertiary rounded animate-pulse" />
        </div>
        <div className="h-4 w-full bg-bg-tertiary rounded animate-pulse" />
        <div className="flex gap-4 pt-2 border-t border-border">
          <div className="h-4 w-16 bg-bg-tertiary rounded animate-pulse" />
          <div className="h-4 w-24 bg-bg-tertiary rounded animate-pulse" />
        </div>
      </CardBody>
    </Card>
  );
}
