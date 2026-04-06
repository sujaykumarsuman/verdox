"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, CheckCircle2, XCircle, MinusCircle, AlertTriangle } from "lucide-react";
import { ResultStatusBadge } from "@/components/test/status-badge";
import { cn } from "@/lib/utils";
import type { TestResult } from "@/types/test";

interface ResultRowProps {
  result: TestResult;
}

const statusIcons = {
  pass: <CheckCircle2 size={16} className="text-[var(--success)]" />,
  fail: <XCircle size={16} className="text-[var(--danger)]" />,
  skip: <MinusCircle size={16} className="text-[var(--warning)]" />,
  error: <AlertTriangle size={16} className="text-[var(--danger)]" />,
};

function formatDuration(ms: number | null): string {
  if (ms === null) return "-";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export function ResultRow({ result }: ResultRowProps) {
  const [expanded, setExpanded] = useState(false);
  const hasLogs = result.error_message;

  return (
    <div className="border-b border-[var(--border)] last:border-b-0">
      <button
        onClick={() => hasLogs && setExpanded(!expanded)}
        className={cn(
          "w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-bg-tertiary/50 transition-colors",
          hasLogs && "cursor-pointer"
        )}
      >
        {statusIcons[result.status] || statusIcons.error}
        <span className="flex-1 text-sm text-text-primary font-mono truncate">
          {result.test_name}
        </span>
        <ResultStatusBadge status={result.status} />
        <span className="text-xs text-text-secondary w-16 text-right">
          {formatDuration(result.duration_ms)}
        </span>
        {hasLogs ? (
          expanded ? (
            <ChevronDown size={16} className="text-text-secondary" />
          ) : (
            <ChevronRight size={16} className="text-text-secondary" />
          )
        ) : (
          <div className="w-4" />
        )}
      </button>

      {expanded && result.error_message && (
        <div className="px-4 pb-3">
          <pre className="bg-bg-tertiary rounded-[6px] p-3 text-[13px] font-mono text-text-secondary overflow-x-auto whitespace-pre-wrap">
            {result.error_message}
          </pre>
        </div>
      )}
    </div>
  );
}
