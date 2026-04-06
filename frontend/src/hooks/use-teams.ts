"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";

export interface Team {
  id: string;
  name: string;
  slug: string;
  created_at: string;
}

export function useTeams() {
  const [teams, setTeams] = useState<Team[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTeams = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<Team[]>("/v1/teams");
      setTeams(data || []);
    } catch {
      setError("Failed to load teams");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTeams();
  }, [fetchTeams]);

  return { teams, isLoading, error, refetch: fetchTeams };
}

export async function createTeam(name: string, slug: string): Promise<Team> {
  return api<Team>("/v1/teams", {
    method: "POST",
    body: JSON.stringify({ name, slug }),
  });
}

export async function deleteTeam(id: string): Promise<void> {
  return api<void>(`/v1/teams/${id}`, { method: "DELETE" });
}
