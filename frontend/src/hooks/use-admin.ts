"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import type { AdminStats, AdminUserListResponse, PendingBanReviewsResponse, UserTeamEntry, AdminTeamEntry } from "@/types/admin";

export function useAdminStats() {
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<AdminStats>("/v1/admin/stats");
      setStats(data);
    } catch {
      setError("Failed to load stats");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  return { stats, isLoading, error, refetch: fetchStats };
}

export function useAdminUsers(
  search: string,
  role: string,
  status: string,
  page: number,
  sort: string = "created_at",
  order: string = "desc"
) {
  const [data, setData] = useState<AdminUserListResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchUsers = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (search) params.set("search", search);
      if (role) params.set("role", role);
      if (status) params.set("status", status);
      params.set("page", String(page));
      params.set("per_page", "20");
      params.set("sort", sort);
      params.set("order", order);

      const result = await api<AdminUserListResponse>(
        `/v1/admin/users?${params.toString()}`
      );
      setData(result);
    } catch {
      setError("Failed to load users");
    } finally {
      setIsLoading(false);
    }
  }, [search, role, status, page, sort, order]);

  useEffect(() => {
    fetchUsers();
    // Periodic refresh to pick up cross-session changes (team count, bans, etc.)
    const interval = setInterval(fetchUsers, 30000);
    return () => clearInterval(interval);
  }, [fetchUsers]);

  return { data, isLoading, error, refetch: fetchUsers };
}

export function useBanReviews() {
  const [data, setData] = useState<PendingBanReviewsResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchReviews = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await api<PendingBanReviewsResponse>("/v1/admin/ban-reviews");
      setData(result);
    } catch {
      setError("Failed to load ban reviews");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchReviews();
  }, [fetchReviews]);

  return { data, isLoading, error, refetch: fetchReviews };
}

export async function reviewBan(id: string, status: "approved" | "denied") {
  return api<{ message: string }>(`/v1/admin/ban-reviews/${id}`, {
    method: "PUT",
    body: JSON.stringify({ status }),
  });
}

export async function updateUser(
  id: string,
  body: { role?: string; is_active?: boolean; is_banned?: boolean; ban_reason?: string }
) {
  return api<{ message: string }>(`/v1/admin/users/${id}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export function useUserTeams(userId: string | null) {
  const [teams, setTeams] = useState<UserTeamEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetchTeams = useCallback(async () => {
    if (!userId) return;
    setIsLoading(true);
    try {
      const data = await api<UserTeamEntry[]>(`/v1/admin/users/${userId}/teams`);
      setTeams(data || []);
    } catch {
      setTeams([]);
    } finally {
      setIsLoading(false);
    }
  }, [userId]);

  useEffect(() => {
    fetchTeams();
  }, [fetchTeams]);

  return { teams, isLoading, refetch: fetchTeams };
}

export function useAllTeams() {
  const [teams, setTeams] = useState<AdminTeamEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const fetchTeams = useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await api<AdminTeamEntry[]>("/v1/admin/teams");
      setTeams(data || []);
    } catch {
      setTeams([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTeams();
  }, [fetchTeams]);

  return { teams, isLoading };
}

export async function updateUserTeams(userId: string, teamIds: string[]) {
  return api<{ message: string }>(`/v1/admin/users/${userId}/teams`, {
    method: "PUT",
    body: JSON.stringify({ team_ids: teamIds }),
  });
}
