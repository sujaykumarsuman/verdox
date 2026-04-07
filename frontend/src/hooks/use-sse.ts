"use client";

import { useEffect, useRef, useCallback } from "react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost/api";

interface SSEEnvelope {
  type: string;
  data: Record<string, unknown>;
  timestamp: string;
}

type SSEEventHandler = (data: Record<string, unknown>) => void;

interface UseSSEOptions {
  enabled: boolean;
  onBanned?: (data: Record<string, unknown>) => void;
  onUnbanned?: (data: Record<string, unknown>) => void;
  onNotificationNew?: (data: Record<string, unknown>) => void;
  onBanReviewRequested?: (data: Record<string, unknown>) => void;
  onTestComplete?: (data: Record<string, unknown>) => void;
}

export function useSSE(options: UseSSEOptions) {
  const { enabled, onBanned, onUnbanned, onNotificationNew, onBanReviewRequested, onTestComplete } = options;
  const esRef = useRef<EventSource | null>(null);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const retryCountRef = useRef(0);
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  // Keep handler refs stable
  const handlersRef = useRef<Record<string, SSEEventHandler | undefined>>({});
  handlersRef.current = {
    banned: onBanned,
    unbanned: onUnbanned,
    notification_new: onNotificationNew,
    ban_review_requested: onBanReviewRequested,
    test_complete: onTestComplete,
  };

  const connect = useCallback(() => {
    if (!enabledRef.current) return;

    // Close existing connection
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }

    const es = new EventSource(`${API_BASE}/v1/sse/stream`, {
      withCredentials: true,
    });

    es.onopen = () => {
      retryCountRef.current = 0;
    };

    // Default message handler (data events without named event type)
    es.onmessage = (event) => {
      try {
        const envelope: SSEEnvelope = JSON.parse(event.data);
        const handler = handlersRef.current[envelope.type];
        if (handler) {
          handler(envelope.data);
        }
      } catch {
        // Ignore malformed events
      }
    };

    es.onerror = () => {
      es.close();
      esRef.current = null;

      if (!enabledRef.current) return;

      // Exponential backoff: 1s -> 2s -> 4s -> ... -> 30s max
      const delay = Math.min(1000 * Math.pow(2, retryCountRef.current), 30000);
      retryCountRef.current++;

      retryRef.current = setTimeout(() => {
        if (enabledRef.current) {
          connect();
        }
      }, delay);
    };

    esRef.current = es;
  }, []);

  useEffect(() => {
    if (enabled) {
      connect();
    }

    return () => {
      if (retryRef.current) {
        clearTimeout(retryRef.current);
        retryRef.current = null;
      }
      if (esRef.current) {
        esRef.current.close();
        esRef.current = null;
      }
    };
  }, [enabled, connect]);
}
