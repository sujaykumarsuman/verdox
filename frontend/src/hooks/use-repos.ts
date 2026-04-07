"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import type {
  Repository,
  RepositoryListResponse,
  Branch,
  Commit,
  PaginationMeta,
} from "@/types/repository";

export function useRepositories(
  teamId: string,
  search: string = "",
  page: number = 1
) {
  const [repos, setRepos] = useState<Repository[]>([]);
  const [meta, setMeta] = useState<PaginationMeta>({
    page: 1,
    per_page: 20,
    total: 0,
    total_pages: 0,
  });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRepos = useCallback(async () => {
    if (!teamId) return;
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({
        team_id: teamId,
        page: String(page),
        per_page: "20",
      });
      if (search) params.set("search", search);

      const data = await api<RepositoryListResponse>(
        `/v1/repositories?${params.toString()}`
      );
      setRepos(data.repositories || []);
      setMeta(data.meta);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load repositories");
    } finally {
      setIsLoading(false);
    }
  }, [teamId, search, page]);

  useEffect(() => {
    fetchRepos();
  }, [fetchRepos]);

  return { repos, meta, isLoading, error, refetch: fetchRepos };
}

export function useRepository(id: string) {
  const [repo, setRepo] = useState<Repository | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRepo = useCallback(async () => {
    if (!id) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<Repository>(`/v1/repositories/${id}`);
      setRepo(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load repository");
    } finally {
      setIsLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchRepo();
  }, [fetchRepo]);

  // Poll every 5s while fork status is non-terminal
  useEffect(() => {
    if (!repo) return;
    if (repo.fork_status !== "forking") return;

    const interval = setInterval(fetchRepo, 5000);
    return () => clearInterval(interval);
  }, [repo?.fork_status, fetchRepo]);

  return { repo, isLoading, error, refetch: fetchRepo };
}

export function useBranches(repoId: string, enabled: boolean) {
  const [branches, setBranches] = useState<Branch[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchBranches = useCallback(async () => {
    if (!repoId || !enabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<Branch[]>(`/v1/repositories/${repoId}/branches`);
      setBranches(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load branches");
    } finally {
      setIsLoading(false);
    }
  }, [repoId, enabled]);

  useEffect(() => {
    fetchBranches();
  }, [fetchBranches]);

  return { branches, isLoading, error, refetch: fetchBranches };
}

export function useCommits(repoId: string, branch: string) {
  const [commits, setCommits] = useState<Commit[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCommits = useCallback(async () => {
    if (!repoId || !branch) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<Commit[]>(
        `/v1/repositories/${repoId}/commits?branch=${encodeURIComponent(branch)}`
      );
      setCommits(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load commits");
    } finally {
      setIsLoading(false);
    }
  }, [repoId, branch]);

  useEffect(() => {
    fetchCommits();
  }, [fetchCommits]);

  return { commits, isLoading, error, refetch: fetchCommits };
}

export async function addRepository(githubUrl: string, teamId: string): Promise<Repository> {
  return api<Repository>("/v1/repositories", {
    method: "POST",
    body: JSON.stringify({ github_url: githubUrl, team_id: teamId }),
  });
}

export async function deleteRepository(id: string): Promise<void> {
  return api<void>(`/v1/repositories/${id}`, { method: "DELETE" });
}

export async function resyncRepository(id: string): Promise<void> {
  return api<void>(`/v1/repositories/${id}/resync`, { method: "POST" });
}
