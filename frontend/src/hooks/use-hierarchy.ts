"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import type {
  TestGroup,
  GroupCasesResponse,
  TestCaseItem,
  ReportResponse,
} from "@/types/test";
import type { PaginationMeta } from "@/types/repository";

export function useRunGroups(runId: string) {
  const [groups, setGroups] = useState<TestGroup[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchGroups = useCallback(async () => {
    if (!runId) {
      setIsLoading(false);
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<TestGroup[]>(`/v1/runs/${runId}/groups`);
      setGroups(data || []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load test groups"
      );
    } finally {
      setIsLoading(false);
    }
  }, [runId]);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  return { groups, isLoading, error, refetch: fetchGroups };
}

export function useGroupCases(
  runId: string,
  groupId: string,
  page: number = 1,
  perPage: number = 100
) {
  const [cases, setCases] = useState<TestCaseItem[]>([]);
  const [meta, setMeta] = useState<PaginationMeta>({
    page: 1,
    per_page: 100,
    total: 0,
    total_pages: 0,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCases = useCallback(async () => {
    if (!runId || !groupId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<GroupCasesResponse>(
        `/v1/runs/${runId}/groups/${groupId}/cases?page=${page}&per_page=${perPage}`
      );
      setCases(data.cases || []);
      setMeta(data.meta);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load test cases"
      );
    } finally {
      setIsLoading(false);
    }
  }, [runId, groupId, page, perPage]);

  useEffect(() => {
    fetchCases();
  }, [fetchCases]);

  return { cases, meta, isLoading, error, refetch: fetchCases };
}

export function useFailedCases(runId: string) {
  const [cases, setCases] = useState<TestCaseItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCases = useCallback(async () => {
    if (!runId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<TestCaseItem[]>(
        `/v1/runs/${runId}/cases/failed`
      );
      setCases(data || []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load failed cases"
      );
    } finally {
      setIsLoading(false);
    }
  }, [runId]);

  useEffect(() => {
    fetchCases();
  }, [fetchCases]);

  return { cases, isLoading, error, refetch: fetchCases };
}

export function useReport(reportId: string | null) {
  const [report, setReport] = useState<ReportResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchReport = useCallback(async () => {
    if (!reportId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await api<ReportResponse>(`/v1/reports/${reportId}`);
      setReport(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load report"
      );
    } finally {
      setIsLoading(false);
    }
  }, [reportId]);

  useEffect(() => {
    fetchReport();
  }, [fetchReport]);

  return { report, isLoading, error, refetch: fetchReport };
}
