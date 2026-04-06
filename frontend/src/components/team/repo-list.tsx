"use client";

import { useState } from "react";
import { GitBranch, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { unassignRepo } from "@/hooks/use-teams";
import type { TeamRepo, TeamRole } from "@/types/team";

interface RepoListProps {
  teamId: string;
  repos: TeamRepo[];
  currentUserRole?: TeamRole;
  onRefresh: () => void;
}

export function RepoList({ teamId, repos, currentUserRole, onRefresh }: RepoListProps) {
  const [loading, setLoading] = useState<string | null>(null);
  const canModify = currentUserRole === "admin" || currentUserRole === "maintainer";

  const handleUnassign = async (repoId: string) => {
    setLoading(repoId);
    try {
      await unassignRepo(teamId, repoId);
      onRefresh();
    } finally {
      setLoading(null);
    }
  };

  return (
    <div className="space-y-2">
      {repos.map((r) => (
        <div
          key={r.id}
          className="flex items-center justify-between p-3 rounded-[6px] border border-border hover:bg-bg-secondary/50"
        >
          <div className="flex items-center gap-3">
            <GitBranch className="h-4 w-4 text-text-secondary" />
            <div>
              <span className="font-medium text-text-primary text-[14px]">
                {r.github_full_name}
              </span>
              {!r.is_active && (
                <Badge variant="danger" className="ml-2">Inactive</Badge>
              )}
            </div>
          </div>
          {canModify && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleUnassign(r.repository_id)}
              loading={loading === r.repository_id}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      ))}
      {repos.length === 0 && (
        <p className="text-center text-text-secondary py-8 text-[14px]">
          No repositories assigned to this team.
        </p>
      )}
    </div>
  );
}
