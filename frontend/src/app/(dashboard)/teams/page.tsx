"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import { Users, ArrowRight, Plus, GitBranch, Search } from "lucide-react";
import { useTeams } from "@/hooks/use-teams";
import { Button } from "@/components/ui/button";
import { Card, CardBody } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { CreateTeamDialog } from "@/components/team/create-team-dialog";
import { RoleBadge } from "@/components/team/role-badge";

export default function TeamsPage() {
  const { teams, isLoading, error, refetch } = useTeams();
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  const handleCreateSuccess = useCallback(() => {
    refetch();
  }, [refetch]);

  return (
    <div className="max-w-3xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="font-display text-[30px] leading-[38px] tracking-[-0.01em] text-text-primary mb-2">
            Teams
          </h1>
          <p className="text-[14px] text-text-secondary">
            Your team memberships. Select a team to manage settings and GitHub PAT.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Link href="/teams/discover">
            <Button variant="secondary">
              <Search className="h-4 w-4" />
              Discover
            </Button>
          </Link>
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4" />
            Create Team
          </Button>
        </div>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i}>
              <CardBody className="flex items-center gap-4">
                <Skeleton className="h-10 w-10 rounded-full" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-5 w-40" />
                  <Skeleton className="h-4 w-24" />
                </div>
              </CardBody>
            </Card>
          ))}
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-4">
          <p className="text-[14px] text-danger">{error}</p>
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !error && teams.length === 0 && (
        <div className="rounded-[8px] border border-dashed bg-bg-secondary p-12 text-center">
          <Users className="h-12 w-12 text-text-secondary mx-auto mb-4" />
          <h3 className="text-[16px] font-medium text-text-primary mb-1">
            No teams yet
          </h3>
          <p className="text-[14px] text-text-secondary mb-4">
            Create a team to start adding GitHub repositories and running tests.
          </p>
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4" />
            Create your first team
          </Button>
        </div>
      )}

      {/* Team list */}
      {!isLoading && !error && teams.length > 0 && (
        <div className="space-y-3">
          {teams.map((team) => (
            <Link key={team.id} href={`/teams/${team.id}`}>
              <Card hoverable className="cursor-pointer">
                <CardBody className="flex items-center gap-4">
                  <div className="h-10 w-10 rounded-full bg-accent/10 flex items-center justify-center shrink-0">
                    <Users className="h-5 w-5 text-accent" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h3 className="text-[16px] font-semibold text-text-primary">
                        {team.name}
                      </h3>
                      {team.my_role && <RoleBadge role={team.my_role} />}
                    </div>
                    <div className="flex items-center gap-3 mt-0.5">
                      <span className="text-[13px] text-text-secondary">/{team.slug}</span>
                      <span className="flex items-center gap-1 text-[12px] text-text-secondary">
                        <Users className="h-3 w-3" />{team.member_count ?? 0}
                      </span>
                      <span className="flex items-center gap-1 text-[12px] text-text-secondary">
                        <GitBranch className="h-3 w-3" />{team.repo_count ?? 0}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 text-text-secondary">
                    <ArrowRight className="h-4 w-4" />
                  </div>
                </CardBody>
              </Card>
            </Link>
          ))}
        </div>
      )}

      {/* Create team dialog */}
      <CreateTeamDialog
        open={showCreateDialog}
        onClose={() => setShowCreateDialog(false)}
        onSuccess={handleCreateSuccess}
      />
    </div>
  );
}
