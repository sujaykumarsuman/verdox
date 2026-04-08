"use client";

import { useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  CheckCircle2,
  XCircle,
  MinusCircle,
  AlertTriangle,
  ExternalLink,
  RotateCw,
} from "lucide-react";
import { ResultStatusBadge } from "@/components/test/status-badge";
import { cn } from "@/lib/utils";
import type { TestCase } from "@/types/test";

const statusIcons = {
  pass: <CheckCircle2 size={16} className="text-[var(--success)]" />,
  fail: <XCircle size={16} className="text-[var(--danger)]" />,
  skip: <MinusCircle size={16} className="text-[var(--warning)]" />,
  error: <AlertTriangle size={16} className="text-[var(--danger)]" />,
};

function formatDuration(ms: number | null): string {
  if (ms === null) return "-";
  if (ms === 0) return "<1ms";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

interface CaseRowProps {
  testCase: TestCase;
}

export function CaseRow({ testCase }: CaseRowProps) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = testCase.error_message || testCase.stack_trace;

  return (
    <div className="border-b border-[var(--border)] last:border-b-0">
      <button
        onClick={() => hasDetails && setExpanded(!expanded)}
        className={cn(
          "w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-bg-tertiary/50 transition-colors",
          hasDetails && "cursor-pointer"
        )}
      >
        {statusIcons[testCase.status] || statusIcons.error}
        <span className="flex-1 text-[13px] text-text-primary font-mono truncate">
          {testCase.name}
        </span>
        {testCase.retry_count > 0 && (
          <span className="flex items-center gap-1 text-[11px] text-text-secondary">
            <RotateCw size={12} />
            {testCase.retry_count}
          </span>
        )}
        <ResultStatusBadge status={testCase.status} />
        <span className="text-[12px] text-text-secondary w-16 text-right">
          {formatDuration(testCase.duration_ms)}
        </span>
        {testCase.logs_url && (
          <a
            href={testCase.logs_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent hover:text-accent/80"
            onClick={(e) => e.stopPropagation()}
          >
            <ExternalLink size={14} />
          </a>
        )}
        {hasDetails ? (
          expanded ? (
            <ChevronDown size={16} className="text-text-secondary" />
          ) : (
            <ChevronRight size={16} className="text-text-secondary" />
          )
        ) : (
          <div className="w-4" />
        )}
      </button>

      {expanded && (
        <div className="px-4 pb-3 space-y-2">
          {testCase.error_message && (
            <div>
              <div className="text-[11px] font-medium text-text-secondary uppercase tracking-wider mb-1">
                Error
              </div>
              <pre className="bg-bg-tertiary rounded-[6px] p-3 text-[13px] font-mono text-[var(--danger)] overflow-x-auto whitespace-pre-wrap">
                {testCase.error_message}
              </pre>
            </div>
          )}
          {testCase.stack_trace && (
            <div>
              <div className="text-[11px] font-medium text-text-secondary uppercase tracking-wider mb-1">
                Stack Trace
              </div>
              <pre className="bg-bg-tertiary rounded-[6px] p-3 text-[12px] font-mono text-text-secondary overflow-x-auto whitespace-pre-wrap max-h-64 overflow-y-auto">
                {testCase.stack_trace}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
