"use client";

import { useState, useCallback } from "react";
import { api } from "@/lib/api";

interface PATValidation {
  valid: boolean;
  github_username?: string;
  error?: string;
}

interface PATInfo {
  is_configured: boolean;
  github_username?: string;
  set_at?: string;
}

export function usePATValidation(teamId: string) {
  const [patInfo, setPATInfo] = useState<PATValidation | PATInfo | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const validate = useCallback(async () => {
    if (!teamId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<PATValidation | PATInfo>(
        `/v1/teams/${teamId}/pat/validate`
      );
      setPATInfo(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to check PAT status");
    } finally {
      setIsLoading(false);
    }
  }, [teamId]);

  return { patInfo, isLoading, error, validate };
}

export async function setPAT(teamId: string, token: string): Promise<PATInfo> {
  return api<PATInfo>(`/v1/teams/${teamId}/pat`, {
    method: "PUT",
    body: JSON.stringify({ token }),
  });
}

export async function revokePAT(teamId: string): Promise<void> {
  return api<void>(`/v1/teams/${teamId}/pat`, { method: "DELETE" });
}
