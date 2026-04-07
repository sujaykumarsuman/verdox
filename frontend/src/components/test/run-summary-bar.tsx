"use client";

import { Clock } from "lucide-react";
import { StatsBar } from "./stats-bar";
import type { RunSummaryV2 } from "@/types/test";

function formatDurationMs(ms: number): string {
  if (ms === 0) return "<1ms";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

interface RunSummaryBarProps {
  summary: RunSummaryV2;
}

export function RunSummaryBar({ summary }: RunSummaryBarProps) {
  return (
    <div className="rounded-[8px] border border-[var(--border)] bg-bg-secondary p-4 mb-6">
      <div className="flex items-center gap-6 mb-3">
        <div className="text-center">
          <div className="text-[20px] font-semibold text-text-primary">
            {summary.total_jobs}
          </div>
          <div className="text-[12px] text-text-secondary">Jobs</div>
        </div>
        <div className="text-center">
          <div className="text-[20px] font-semibold text-text-primary">
            {summary.total_cases}
          </div>
          <div className="text-[12px] text-text-secondary">Cases</div>
        </div>
        <div className="text-center">
          <div className="text-[20px] font-semibold text-[var(--success)]">
            {summary.passed}
          </div>
          <div className="text-[12px] text-text-secondary">Passed</div>
        </div>
        <div className="text-center">
          <div className="text-[20px] font-semibold text-[var(--danger)]">
            {summary.failed}
          </div>
          <div className="text-[12px] text-text-secondary">Failed</div>
        </div>
        {summary.skipped > 0 && (
          <div className="text-center">
            <div className="text-[20px] font-semibold text-[var(--warning)]">
              {summary.skipped}
            </div>
            <div className="text-[12px] text-text-secondary">Skipped</div>
          </div>
        )}
        <div className="ml-auto flex flex-col items-end gap-1">
          <div className="text-[14px] font-semibold text-text-primary">
            {summary.pass_rate.toFixed(1)}%
          </div>
          <div className="flex items-center gap-1.5 text-[13px] text-text-secondary">
            <Clock className="h-3.5 w-3.5" />
            {formatDurationMs(summary.duration_ms)}
          </div>
        </div>
      </div>
      <StatsBar
        passed={summary.passed}
        failed={summary.failed}
        skipped={summary.skipped}
        total={summary.total_cases}
      />
    </div>
  );
}
