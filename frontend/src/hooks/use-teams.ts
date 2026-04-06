"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import type {
  Team,
  TeamDetail,
  TeamMember,
  TeamJoinRequest,
  TeamRepo,
  DiscoverableTeam,
  TeamRole,
} from "@/types/team";

// Re-export types for backwards compatibility
export type { Team };

// --- Hooks ---

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

export function useTeamDetail(teamId: string) {
  const [team, setTeam] = useState<TeamDetail | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTeam = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<TeamDetail>(`/v1/teams/${teamId}`);
      setTeam(data);
    } catch {
      setError("Failed to load team");
    } finally {
      setIsLoading(false);
    }
  }, [teamId]);

  useEffect(() => {
    fetchTeam();
  }, [fetchTeam]);

  return { team, isLoading, error, refetch: fetchTeam };
}

export function useDiscoverableTeams() {
  const [teams, setTeams] = useState<DiscoverableTeam[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTeams = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<DiscoverableTeam[]>("/v1/teams/discover");
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

export function useJoinRequests(teamId: string, status?: string) {
  const [requests, setRequests] = useState<TeamJoinRequest[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const params = status ? `?status=${status}` : "";
      const data = await api<TeamJoinRequest[]>(
        `/v1/teams/${teamId}/join-requests${params}`
      );
      setRequests(data || []);
    } catch {
      setError("Failed to load join requests");
    } finally {
      setIsLoading(false);
    }
  }, [teamId, status]);

  useEffect(() => {
    fetchRequests();
  }, [fetchRequests]);

  return { requests, isLoading, error, refetch: fetchRequests };
}

// --- Mutations ---

export async function createTeam(name: string, slug: string): Promise<Team> {
  return api<Team>("/v1/teams", {
    method: "POST",
    body: JSON.stringify({ name, slug }),
  });
}

export async function updateTeam(id: string, name: string): Promise<Team> {
  return api<Team>(`/v1/teams/${id}`, {
    method: "PUT",
    body: JSON.stringify({ name }),
  });
}

export async function deleteTeam(id: string): Promise<void> {
  return api<void>(`/v1/teams/${id}`, { method: "DELETE" });
}

export async function inviteMember(
  teamId: string,
  userId: string,
  role: TeamRole
): Promise<TeamMember> {
  return api<TeamMember>(`/v1/teams/${teamId}/members`, {
    method: "POST",
    body: JSON.stringify({ user_id: userId, role }),
  });
}

export async function updateMember(
  teamId: string,
  userId: string,
  data: { role?: TeamRole; status?: "approved" | "rejected" }
): Promise<void> {
  return api<void>(`/v1/teams/${teamId}/members/${userId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function removeMember(
  teamId: string,
  userId: string
): Promise<void> {
  return api<void>(`/v1/teams/${teamId}/members/${userId}`, {
    method: "DELETE",
  });
}

export async function assignRepo(
  teamId: string,
  repositoryId: string
): Promise<TeamRepo> {
  return api<TeamRepo>(`/v1/teams/${teamId}/repositories`, {
    method: "POST",
    body: JSON.stringify({ repository_id: repositoryId }),
  });
}

export async function unassignRepo(
  teamId: string,
  repoId: string
): Promise<void> {
  return api<void>(`/v1/teams/${teamId}/repositories/${repoId}`, {
    method: "DELETE",
  });
}

export async function submitJoinRequest(
  teamId: string,
  message?: string
): Promise<void> {
  return api<void>(`/v1/teams/${teamId}/join-requests`, {
    method: "POST",
    body: JSON.stringify({ message: message || "" }),
  });
}

export async function reviewJoinRequest(
  teamId: string,
  requestId: string,
  status: "approved" | "rejected",
  role?: TeamRole
): Promise<void> {
  return api<void>(`/v1/teams/${teamId}/join-requests/${requestId}`, {
    method: "PATCH",
    body: JSON.stringify({ status, role }),
  });
}
