"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { Plus, Search, FolderGit2, Users } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useRepositories } from "@/hooks/use-repos";
import { useTeams } from "@/hooks/use-teams";
import { Button } from "@/components/ui/button";
import { RepoCard, RepoCardSkeleton } from "@/components/repository/repo-card";
import { AddRepoDialog } from "@/components/repository/add-repo-dialog";

export default function DashboardPage() {
  const { user } = useAuth();
  const { teams, isLoading: teamsLoading } = useTeams();
  const [selectedTeamId, setSelectedTeamId] = useState<string>("");
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);
  const [showAddDialog, setShowAddDialog] = useState(false);

  // Auto-select first team
  useEffect(() => {
    if (teams.length > 0 && !selectedTeamId) {
      setSelectedTeamId(teams[0].id);
    }
  }, [teams, selectedTeamId]);

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const { repos, meta, isLoading, error, refetch } = useRepositories(
    selectedTeamId,
    debouncedSearch,
    page
  );

  const handleAddSuccess = useCallback(() => {
    refetch();
  }, [refetch]);

  // Loading teams
  if (teamsLoading) {
    return (
      <div className="max-w-6xl">
        <div className="h-8 w-48 bg-bg-tertiary rounded animate-pulse mb-6" />
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {Array.from({ length: 6 }).map((_, i) => (
            <RepoCardSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  // No teams
  if (teams.length === 0) {
    return (
      <div className="max-w-3xl">
        <h1 className="font-display text-[30px] leading-[38px] tracking-[-0.01em] text-text-primary mb-2">
          Welcome to Verdox
        </h1>
        <p className="text-[16px] text-text-secondary mb-8">
          {user ? `Hello, ${user.username}. ` : ""}
          Create a team to start adding repositories and running tests.
        </p>
        <div className="rounded-[8px] border border-dashed bg-bg-secondary p-12 text-center">
          <Users className="h-12 w-12 text-text-secondary mx-auto mb-4" />
          <h3 className="text-[16px] font-medium text-text-primary mb-1">
            No teams yet
          </h3>
          <p className="text-[14px] text-text-secondary mb-4">
            Teams let you manage repositories and collaborate with others.
          </p>
          <Link href="/teams">
            <Button>
              <Plus className="h-4 w-4" />
              Create a team
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="font-display text-[30px] leading-[38px] tracking-[-0.01em] text-text-primary">
            Repositories
          </h1>
          <p className="text-[14px] text-text-secondary mt-1">
            Manage your team&apos;s GitHub repositories
          </p>
        </div>
        <Button onClick={() => setShowAddDialog(true)}>
          <Plus className="h-4 w-4" />
          Add Repository
        </Button>
      </div>

      {/* Search + Team filter */}
      <div className="flex items-center gap-3 mb-6">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-secondary" />
          <input
            type="text"
            placeholder="Search repositories..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-[4px] border bg-bg-primary pl-10 pr-3 py-2 text-[14px] text-text-primary placeholder:text-text-secondary focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20"
          />
        </div>
        {teams.length > 1 && (
          <select
            value={selectedTeamId}
            onChange={(e) => {
              setSelectedTeamId(e.target.value);
              setPage(1);
            }}
            className="rounded-[4px] border bg-bg-primary px-3 py-2 text-[14px] text-text-primary focus:border-accent focus:outline-none"
          >
            {teams.map((team) => (
              <option key={team.id} value={team.id}>
                {team.name}
              </option>
            ))}
          </select>
        )}
      </div>

      {/* Error state */}
      {error && (
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-4 mb-6">
          <p className="text-[14px] text-danger">{error}</p>
          <Button variant="ghost" size="sm" onClick={refetch} className="mt-2">
            Retry
          </Button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {Array.from({ length: 6 }).map((_, i) => (
            <RepoCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !error && repos.length === 0 && (
        <div className="rounded-[8px] border border-dashed bg-bg-secondary p-12 text-center">
          <FolderGit2 className="h-12 w-12 text-text-secondary mx-auto mb-4" />
          <h3 className="text-[16px] font-medium text-text-primary mb-1">
            No repositories yet
          </h3>
          <p className="text-[14px] text-text-secondary mb-4">
            Add your first GitHub repository to get started with test orchestration.
          </p>
          <Button onClick={() => setShowAddDialog(true)}>
            <Plus className="h-4 w-4" />
            Add your first repository
          </Button>
        </div>
      )}

      {/* Repo grid */}
      {!isLoading && !error && repos.length > 0 && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
            {repos.map((repo) => (
              <RepoCard key={repo.id} repo={repo} />
            ))}
          </div>

          {/* Pagination */}
          {meta.total_pages > 1 && (
            <div className="flex items-center justify-between mt-6 pt-4 border-t border-border">
              <p className="text-[13px] text-text-secondary">
                Page {meta.page} of {meta.total_pages} ({meta.total} repositories)
              </p>
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage(page - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={page >= meta.total_pages}
                  onClick={() => setPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Add repo dialog */}
      <AddRepoDialog
        teamId={selectedTeamId}
        open={showAddDialog}
        onClose={() => setShowAddDialog(false)}
        onSuccess={handleAddSuccess}
      />
    </div>
  );
}
