"use client";

import { useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  Package,
  Loader2,
} from "lucide-react";
import { ResultStatusBadge } from "@/components/test/status-badge";
import { StatsBar } from "@/components/test/stats-bar";
import { CaseRow } from "@/components/test/case-row";
import { useGroupCases } from "@/hooks/use-hierarchy";
import type { TestGroup } from "@/types/test";

function formatDuration(ms: number | null): string {
  if (ms === null) return "-";
  if (ms === 0) return "<1ms";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

interface GroupRowProps {
  runId: string;
  group: TestGroup;
}

export function GroupRow({ runId, group }: GroupRowProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border-b border-[var(--border)] last:border-b-0">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-bg-tertiary/50 transition-colors cursor-pointer"
      >
        {expanded ? (
          <ChevronDown size={16} className="text-text-secondary shrink-0" />
        ) : (
          <ChevronRight size={16} className="text-text-secondary shrink-0" />
        )}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-[14px] font-medium text-text-primary truncate">
              {group.name}
            </span>
            <ResultStatusBadge status={group.status} />
          </div>
          {group.package && (
            <div className="flex items-center gap-1.5 mt-0.5">
              <Package size={12} className="text-text-secondary shrink-0" />
              <span className="text-[12px] text-text-secondary font-mono truncate">
                {group.package}
              </span>
            </div>
          )}
        </div>
        <div className="flex items-center gap-4 shrink-0">
          <div className="flex items-center gap-2 text-[12px]">
            <span className="text-[var(--success)]">{group.passed}</span>
            <span className="text-text-secondary">/</span>
            <span className="text-text-primary">{group.total}</span>
          </div>
          <div className="w-24">
            <StatsBar
              passed={group.passed}
              failed={group.failed}
              skipped={group.skipped}
              total={group.total}
            />
          </div>
          <span className="text-[12px] text-text-secondary w-16 text-right">
            {formatDuration(group.duration_ms)}
          </span>
          {group.pass_rate !== null && (
            <span className="text-[12px] text-text-secondary w-14 text-right">
              {group.pass_rate.toFixed(1)}%
            </span>
          )}
        </div>
      </button>

      {expanded && (
        <GroupCasesPanel runId={runId} groupId={group.id} />
      )}
    </div>
  );
}

function GroupCasesPanel({ runId, groupId }: { runId: string; groupId: string }) {
  const { cases, isLoading, error } = useGroupCases(runId, groupId);

  if (isLoading) {
    return (
      <div className="px-8 py-4 flex items-center gap-2 text-[13px] text-text-secondary">
        <Loader2 size={14} className="animate-spin" />
        Loading cases...
      </div>
    );
  }

  if (error) {
    return (
      <div className="px-8 py-4 text-[13px] text-[var(--danger)]">
        Failed to load cases
      </div>
    );
  }

  if (cases.length === 0) {
    return (
      <div className="px-8 py-4 text-[13px] text-text-secondary">
        No test cases
      </div>
    );
  }

  return (
    <div className="ml-8 border-l border-[var(--border)]">
      {cases.map((tc) => (
        <CaseRow key={tc.id} testCase={tc} />
      ))}
    </div>
  );
}
