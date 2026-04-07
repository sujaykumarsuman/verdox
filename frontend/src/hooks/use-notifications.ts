"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { api } from "@/lib/api";
import type { NotificationListResponse, UnreadCountResponse } from "@/types/notification";

export function useNotifications(page: number = 1) {
  const [data, setData] = useState<NotificationListResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [version, setVersion] = useState(0);

  const fetchNotifications = useCallback(async () => {
    try {
      const result = await api<NotificationListResponse>(
        `/v1/notifications?page=${page}&per_page=20`
      );
      setData(result);
    } catch {
      // Ignore errors
    } finally {
      setIsLoading(false);
    }
  }, [page, version]);

  useEffect(() => {
    fetchNotifications();
  }, [fetchNotifications]);

  // Bump version to force refetch
  const refetch = useCallback(() => {
    setVersion((v) => v + 1);
  }, []);

  return { data, isLoading, refetch };
}

export function useUnreadCount() {
  const [count, setCount] = useState(0);
  const [version, setVersion] = useState(0);

  const fetchCount = useCallback(async () => {
    try {
      const result = await api<UnreadCountResponse>("/v1/notifications/unread-count");
      setCount(result.count);
    } catch {
      // Ignore errors
    }
  }, []);

  useEffect(() => {
    fetchCount();
    const interval = setInterval(fetchCount, 30000);
    return () => clearInterval(interval);
  }, [fetchCount, version]);

  const refetch = useCallback(() => {
    setVersion((v) => v + 1);
    fetchCount();
  }, [fetchCount]);

  return { count, refetch };
}

export function useMarkRead() {
  return useCallback(async (id: string) => {
    await api(`/v1/notifications/${id}/read`, { method: "PUT" });
  }, []);
}

export function useMarkAllRead() {
  return useCallback(async () => {
    await api("/v1/notifications/read-all", { method: "PUT" });
  }, []);
}
