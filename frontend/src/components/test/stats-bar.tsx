"use client";

interface StatsBarProps {
  passed: number;
  failed: number;
  skipped: number;
  total: number;
  className?: string;
}

export function StatsBar({ passed, failed, skipped, total, className }: StatsBarProps) {
  if (total === 0) return null;

  const passedPct = (passed / total) * 100;
  const failedPct = (failed / total) * 100;
  const skippedPct = (skipped / total) * 100;

  return (
    <div className={`flex h-2 w-full rounded-full overflow-hidden bg-bg-tertiary ${className || ""}`}>
      {passedPct > 0 && (
        <div
          className="bg-[var(--success)] transition-all"
          style={{ width: `${passedPct}%` }}
        />
      )}
      {failedPct > 0 && (
        <div
          className="bg-[var(--danger)] transition-all"
          style={{ width: `${failedPct}%` }}
        />
      )}
      {skippedPct > 0 && (
        <div
          className="bg-[var(--warning)] transition-all"
          style={{ width: `${skippedPct}%` }}
        />
      )}
    </div>
  );
}
