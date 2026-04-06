"use client";

import { useEffect, useRef } from "react";
import { useRunLogs } from "@/hooks/use-tests";
import { Skeleton } from "@/components/ui/skeleton";

interface LogViewerProps {
  runId: string;
}

// Strip ANSI escape codes for v1
function stripAnsi(text: string): string {
  return text.replace(/\x1B\[[0-9;]*m/g, "");
}

export function LogViewer({ runId }: LogViewerProps) {
  const { logs, isLoading, error } = useRunLogs(runId);
  const containerRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs]);

  if (isLoading) {
    return (
      <div className="space-y-2 p-4">
        <Skeleton width="100%" height="16px" />
        <Skeleton width="80%" height="16px" />
        <Skeleton width="90%" height="16px" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 text-sm text-[var(--danger)]">
        Failed to load logs: {error}
      </div>
    );
  }

  if (!logs || logs.logs.length === 0) {
    return (
      <div className="p-4 text-sm text-text-secondary">No logs available.</div>
    );
  }

  return (
    <div
      ref={containerRef}
      className="bg-bg-tertiary rounded-[8px] p-4 font-mono text-[13px] overflow-auto max-h-[600px]"
    >
      {logs.logs.map((entry, idx) => (
        <div key={idx} className="mb-4">
          <div className="flex items-center gap-2 mb-1 text-text-primary font-semibold">
            <span>{entry.test_name}</span>
            <span className="text-text-secondary font-normal text-[12px]">
              {entry.status}
              {entry.duration_ms !== null && ` (${entry.duration_ms}ms)`}
            </span>
          </div>
          {entry.log_output && (
            <pre className="text-text-secondary whitespace-pre-wrap break-all">
              {stripAnsi(entry.log_output)}
            </pre>
          )}
        </div>
      ))}
    </div>
  );
}
