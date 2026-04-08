"use client";

import { use, useState, useEffect, useCallback, useMemo } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  AlertCircle,
  GitBranch,
  GitCommit,
  Clock,
  CheckCircle2,
  XCircle,
  MinusCircle,
  Package,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  RotateCw,
  Pencil,
  RotateCcw,
  X,
} from "lucide-react";
import { toast } from "sonner";
import { useTestRuns, updateTestSuite, rerunRun } from "@/hooks/use-tests";
import { useRunGroups, useGroupCases } from "@/hooks/use-hierarchy";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { RunStatusBadge, ResultStatusBadge } from "@/components/test/status-badge";
import { WorkflowEditor } from "@/components/test/workflow-editor";
import { StatsBar } from "@/components/test/stats-bar";
import { formatDate, cn } from "@/lib/utils";
import { api } from "@/lib/api";
import type {
  TestSuite,
  TestRun,
  TestRunDetailV2,
  TestGroup,
  TestCase,
  RunSummaryV2,
} from "@/types/test";

// ─── Helpers ───────────────────────────────────────────────────────────

function formatDuration(ms: number | null): string {
  if (ms === null) return "-";
  if (ms === 0) return "<1ms";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

function formatDurationSec(s: number | null): string {
  if (s === null || s === 0) return "-";
  if (s < 1) return `${Math.round(s * 1000)}ms`;
  if (s < 60) return `${s.toFixed(1)}s`;
  return `${Math.floor(s / 60)}m ${Math.floor(s % 60)}s`;
}

const suiteTypeColors: Record<string, string> = {
  unit: "bg-blue-500/10 text-blue-400",
  integration: "bg-purple-500/10 text-purple-400",
  e2e: "bg-indigo-500/10 text-indigo-400",
  lint: "bg-amber-500/10 text-amber-400",
  build: "bg-emerald-500/10 text-emerald-400",
  race: "bg-red-500/10 text-red-400",
};

// ─── Hooks ─────────────────────────────────────────────────────────────

function useSuiteDetail(suiteId: string) {
  const [suite, setSuite] = useState<TestSuite | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchSuite = useCallback(async () => {
    if (!suiteId) return;
    setIsLoading(true);
    try {
      setSuite(await api<TestSuite>(`/v1/suites/${suiteId}`));
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load suite");
    } finally {
      setIsLoading(false);
    }
  }, [suiteId]);
  useEffect(() => { fetchSuite(); }, [fetchSuite]);
  return { suite, isLoading, error, refetch: fetchSuite };
}

function useRunDetail(runId: string) {
  const [run, setRun] = useState<TestRunDetailV2 | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  useEffect(() => {
    if (!runId) return;
    let cancelled = false;
    setIsLoading(true);
    (async () => {
      try {
        const data = await api<TestRunDetailV2>(`/v1/runs/${runId}`);
        if (!cancelled) setRun(data);
      } catch { /* */ }
      finally { if (!cancelled) setIsLoading(false); }
    })();
    return () => { cancelled = true; };
  }, [runId]);
  return { run, isLoading };
}

// ─── Run Selector (GitHub-style) ───────────────────────────────────────

const statusDotColor: Record<string, string> = {
  passed: "bg-[var(--success)]",
  failed: "bg-[var(--danger)]",
  running: "bg-[var(--accent)]",
  queued: "bg-[var(--warning)]",
  cancelled: "bg-text-secondary",
};

function RunSelector({ runs, selectedId, onSelect }: { runs: TestRun[]; selectedId: string; onSelect: (id: string) => void }) {
  const [open, setOpen] = useState(false);
  const selected = runs.find((r) => r.id === selectedId);

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 px-3 py-1.5 text-[13px] rounded-[6px] border border-[var(--border)] bg-bg-secondary text-text-primary hover:bg-bg-tertiary transition-colors"
      >
        <Clock size={13} className="text-text-secondary" />
        {selected && <span className={`w-2 h-2 rounded-full ${statusDotColor[selected.status] || "bg-text-secondary"}`} />}
        <span>#{selected?.run_number ?? "?"}</span>
        <ChevronDown size={13} className="text-text-secondary" />
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full mt-1 z-20 w-64 rounded-[8px] border border-[var(--border)] bg-bg-secondary shadow-lg overflow-hidden">
            {runs.map((r) => (
              <button
                key={r.id}
                onClick={() => { onSelect(r.id); setOpen(false); }}
                className={cn(
                  "w-full flex items-start gap-3 px-4 py-2.5 text-left hover:bg-bg-tertiary transition-colors",
                  r.id === selectedId && "bg-bg-tertiary"
                )}
              >
                <span className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${statusDotColor[r.status] || "bg-text-secondary"}`} />
                <div className="min-w-0">
                  <div className="text-[13px] font-medium text-text-primary">
                    {r.id === selectedId && "✓ "}Run #{r.run_number}
                  </div>
                  <div className="text-[12px] text-text-secondary">
                    {r.status} · {formatDate(r.created_at)}
                  </div>
                </div>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

// ─── Page ──────────────────────────────────────────────────────────────

export default function SuiteDetailPage({
  params,
}: {
  params: Promise<{ id: string; suiteId: string }>;
}) {
  const { id: repoId, suiteId } = use(params);
  const { suite, isLoading: suiteLoading, error: suiteError, refetch: refetchSuite } = useSuiteDetail(suiteId);
  const { runs, isLoading: runsLoading, refetch: refetchRuns } = useTestRuns(suiteId, 1);
  const [selectedRunId, setSelectedRunId] = useState<string>("");
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [rerunning, setRerunning] = useState(false);

  // Auto-select latest run
  useEffect(() => {
    if (runs.length > 0 && !selectedRunId) {
      setSelectedRunId(runs[0].id);
    }
  }, [runs, selectedRunId]);

  const selectedRun = runs.find((r) => r.id === selectedRunId);
  const { run: runDetail, isLoading: detailLoading } = useRunDetail(selectedRunId);
  const hasHierarchy = !!runDetail?.summary_v2;

  // Fetch groups for hierarchical runs
  const { groups, isLoading: groupsLoading } = useRunGroups(
    hasHierarchy ? selectedRunId : ""
  );

  // ─── Loading / Error states ───

  if (suiteLoading) {
    return (
      <div className="max-w-6xl">
        <div className="h-5 w-32 bg-bg-tertiary rounded animate-pulse mb-4" />
        <div className="h-8 w-64 bg-bg-tertiary rounded animate-pulse mb-2" />
        <div className="h-48 bg-bg-tertiary rounded animate-pulse mt-6" />
      </div>
    );
  }

  if (suiteError || !suite) {
    return (
      <div className="max-w-6xl">
        <Link href={`/repositories/${repoId}`} className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4">
          <ArrowLeft className="h-4 w-4" /> Back to Repository
        </Link>
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-8 text-center">
          <AlertCircle className="h-10 w-10 text-danger mx-auto mb-3" />
          <h2 className="text-[18px] font-semibold text-text-primary mb-1">Suite not found</h2>
          <p className="text-[14px] text-text-secondary">{suiteError}</p>
        </div>
      </div>
    );
  }

  const isTerminal = selectedRun && ["passed", "failed", "cancelled"].includes(selectedRun.status);

  return (
    <div className="max-w-6xl">
      {/* Breadcrumb */}
      <Link href={`/repositories/${repoId}`} className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4">
        <ArrowLeft className="h-4 w-4" /> Back to Repository
      </Link>

      {/* ── Header: suite name + run selector ── */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <div className="flex items-center gap-3 mb-1">
            <h1 className="font-display text-[28px] leading-[36px] tracking-[-0.01em] text-text-primary">
              {suite.name}
            </h1>
            <Badge variant="neutral">{suite.type}</Badge>
            <button
              onClick={() => setShowEditDialog(true)}
              className="text-text-secondary hover:text-text-primary transition-colors"
              title="Edit suite"
            >
              <Pencil className="h-4 w-4" />
            </button>
          </div>
          {selectedRun && (
            <div className="flex items-center gap-3 text-[13px] text-text-secondary">
              <span className="flex items-center gap-1"><GitBranch size={13} /> {selectedRun.branch}</span>
              <span className="flex items-center gap-1"><GitCommit size={13} /> <code className="font-mono text-accent">{selectedRun.commit_hash.substring(0, 7)}</code></span>
              <span>{formatDate(selectedRun.created_at)}</span>
              {selectedRun.finished_at && selectedRun.started_at && (
                <span className="flex items-center gap-1">
                  <Clock size={13} />
                  {formatDuration(new Date(selectedRun.finished_at).getTime() - new Date(selectedRun.started_at).getTime())}
                </span>
              )}
              {runDetail?.gha_run_url && (
                <a href={runDetail.gha_run_url} target="_blank" rel="noopener noreferrer" className="text-accent hover:underline flex items-center gap-1">
                  GitHub <ExternalLink size={11} />
                </a>
              )}
            </div>
          )}
        </div>

        {/* Run selector + Rerun */}
        <div className="flex items-center gap-2 shrink-0">
          {isTerminal && selectedRun?.status !== "passed" && (
            <Button
              variant="secondary"
              size="sm"
              onClick={async () => {
                setRerunning(true);
                try {
                  await rerunRun(selectedRunId);
                  toast.success("Rerun triggered");
                  refetchRuns();
                } catch (err) {
                  toast.error(err instanceof Error ? err.message : "Rerun failed");
                } finally {
                  setRerunning(false);
                }
              }}
              loading={rerunning}
            >
              <RotateCcw className="h-3.5 w-3.5" />
              Rerun
            </Button>
          )}
          {runsLoading ? (
            <div className="w-24 h-8 bg-bg-tertiary rounded animate-pulse" />
          ) : runs.length > 0 ? (
            <RunSelector runs={runs} selectedId={selectedRunId} onSelect={setSelectedRunId} />
          ) : (
            <span className="text-[13px] text-text-secondary">No runs</span>
          )}
        </div>
      </div>

      {/* ── Run in progress ── */}
      {selectedRun && !isTerminal && (
        <div className="rounded-[8px] border border-[var(--accent)]/30 bg-[var(--accent-subtle)] p-4 mb-6 flex items-center gap-3">
          <Loader2 className="h-5 w-5 text-accent animate-spin" />
          <p className="text-[14px] text-text-secondary">
            {selectedRun.status === "queued" ? "Waiting in queue..." : "Test run in progress. Results will appear when complete."}
          </p>
        </div>
      )}

      {/* ── Results section ── */}
      {isTerminal && detailLoading && (
        <div className="flex items-center gap-2 text-[14px] text-text-secondary py-12 justify-center">
          <Loader2 size={16} className="animate-spin" /> Loading results...
        </div>
      )}

      {isTerminal && !detailLoading && runDetail && (
        <>
          {/* Hierarchical results (new runs with jobs data) */}
          {hasHierarchy && runDetail.summary_v2 && (
            <>
              <SummaryBar summary={runDetail.summary_v2} />
              {groupsLoading ? (
                <div className="flex items-center gap-2 text-[14px] text-text-secondary py-8 justify-center">
                  <Loader2 size={16} className="animate-spin" /> Loading jobs...
                </div>
              ) : (groups || []).length > 0 ? (
                <div className="space-y-4 mt-6">
                  {(groups || []).map((group) => (
                    <SuiteSection key={group.id} group={group} runId={selectedRunId} />
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-[14px] text-text-secondary">No job results found</div>
              )}
            </>
          )}

          {/* Flat results (old runs without hierarchy) */}
          {!hasHierarchy && (
            <>
              {runDetail.summary && runDetail.summary.total > 0 && (
                <div className="flex items-center gap-5 mb-4 px-1 text-[13px]">
                  <span className="text-text-primary font-medium">{runDetail.summary.total} tests</span>
                  <span className="text-[var(--success)]">{runDetail.summary.passed} passed</span>
                  <span className="text-[var(--danger)]">{runDetail.summary.failed} failed</span>
                  {runDetail.summary.skipped > 0 && <span className="text-[var(--warning)]">{runDetail.summary.skipped} skipped</span>}
                  <span className="ml-auto text-text-secondary">{formatDuration(runDetail.summary.duration_ms)}</span>
                </div>
              )}
              {runDetail.results && runDetail.results.length > 0 ? (
                <div className="rounded-[8px] border border-[var(--border)] overflow-hidden">
                  {runDetail.results.map((r) => (
                    <FlatResultRow key={r.id} result={r} />
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-[14px] text-text-secondary">No test results available for this run</div>
              )}
            </>
          )}
        </>
      )}

      {/* No runs */}
      {!runsLoading && runs.length === 0 && (
        <div className="text-center py-12 text-[14px] text-text-secondary">No runs yet — trigger a run from the repository page</div>
      )}

      {/* Edit Suite Dialog */}
      {showEditDialog && suite && (
        <EditSuiteDialog
          suite={suite}
          onClose={() => setShowEditDialog(false)}
          onSaved={() => { setShowEditDialog(false); refetchSuite(); }}
        />
      )}
    </div>
  );
}

// ─── Edit Suite Dialog ────────────────────────────────────────────────

const TYPE_OPTIONS = ["unit", "integration", "e2e", "lint", "smoke", "build", "race", "compatibility", "load"];

function EditSuiteDialog({ suite, onClose, onSaved }: { suite: TestSuite; onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState(suite.name);
  const [type, setType] = useState(suite.type);
  const [timeout, setTimeout] = useState(suite.timeout_seconds || 300);
  const [workflowYaml, setWorkflowYaml] = useState(suite.workflow_yaml || "");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    if (!name.trim()) { toast.error("Suite name is required"); return; }
    setSaving(true);
    try {
      await updateTestSuite(suite.id, {
        name: name.trim(),
        type,
        timeout_seconds: timeout,
        workflow_yaml: workflowYaml,
      });
      toast.success("Suite updated");
      onSaved();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update suite");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-3xl max-h-[90vh] overflow-y-auto rounded-[8px] border bg-bg-secondary shadow-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-[18px] font-semibold text-text-primary">Edit Suite</h2>
          <button onClick={onClose} className="text-text-secondary hover:text-text-primary transition-colors">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="grid grid-cols-3 gap-4 mb-4">
          <div>
            <label className="block text-[13px] font-medium text-text-secondary mb-1.5">Suite Name</label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-text-secondary mb-1.5">Type</label>
            <select
              value={type}
              onChange={(e) => setType(e.target.value)}
              className="w-full px-3 py-2 text-[14px] rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
            >
              {TYPE_OPTIONS.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-[13px] font-medium text-text-secondary mb-1.5">Timeout (seconds)</label>
            <Input type="number" value={timeout} onChange={(e) => setTimeout(Number(e.target.value))} min={30} max={3600} />
          </div>
        </div>

        <div className="mb-4">
          <label className="block text-[13px] font-medium text-text-secondary mb-1.5">Workflow YAML</label>
          <WorkflowEditor value={workflowYaml} onChange={setWorkflowYaml} />
        </div>

        <div className="flex justify-end gap-3">
          <Button variant="ghost" onClick={onClose} disabled={saving}>Cancel</Button>
          <Button onClick={handleSave} loading={saving}>Save Changes</Button>
        </div>
      </div>
    </div>
  );
}

// ─── Summary Bar ───────────────────────────────────────────────────────

function SummaryBar({ summary }: { summary: RunSummaryV2 }) {
  return (
    <div className="rounded-[8px] border border-[var(--border)] bg-bg-secondary p-5">
      <div className="flex items-center gap-8 mb-3">
        <Stat label="Jobs" value={summary.total_jobs} />
        <Stat label="Test Cases" value={summary.total_cases} />
        <Stat label="Passed" value={summary.passed} color="var(--success)" />
        <Stat label="Failed" value={summary.failed} color="var(--danger)" />
        {summary.skipped > 0 && <Stat label="Skipped" value={summary.skipped} color="var(--warning)" />}
        <div className="ml-auto text-right">
          <div className="text-[22px] font-semibold text-text-primary">{summary.pass_rate.toFixed(1)}%</div>
          <div className="text-[12px] text-text-secondary">Pass Rate</div>
        </div>
        <div className="text-right">
          <div className="text-[14px] font-medium text-text-primary flex items-center gap-1"><Clock size={13} /> {formatDuration(summary.duration_ms)}</div>
          <div className="text-[12px] text-text-secondary">Duration</div>
        </div>
      </div>
      <StatsBar passed={summary.passed} failed={summary.failed} skipped={summary.skipped} total={summary.total_cases} className="h-2.5" />
    </div>
  );
}

function Stat({ label, value, color }: { label: string; value: number; color?: string }) {
  return (
    <div className="text-center">
      <div className="text-[22px] font-semibold" style={color ? { color } : undefined}>{value}</div>
      <div className="text-[12px] text-text-secondary">{label}</div>
    </div>
  );
}

// ─── Suite Section (one per group — shows as a card with tests inside) ─

function SuiteSection({ group, runId }: { group: TestGroup; runId: string }) {
  const [expanded, setExpanded] = useState(group.failed > 0); // auto-expand if failures
  const typeColor = suiteTypeColors[group.name.toLowerCase().includes("lint") ? "lint" : group.name.toLowerCase().includes("build") ? "build" : "unit"] || "bg-bg-tertiary text-text-secondary";

  const statusIcon = group.status === "pass"
    ? <CheckCircle2 size={18} className="text-[var(--success)]" />
    : group.status === "fail"
      ? <XCircle size={18} className="text-[var(--danger)]" />
      : <MinusCircle size={18} className="text-[var(--warning)]" />;

  return (
    <div className="rounded-[8px] border border-[var(--border)] bg-bg-secondary overflow-hidden">
      {/* Suite header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-3 px-5 py-3.5 hover:bg-bg-tertiary/30 transition-colors text-left"
      >
        {expanded
          ? <ChevronDown size={16} className="text-text-secondary shrink-0" />
          : <ChevronRight size={16} className="text-text-secondary shrink-0" />}
        {statusIcon}
        <span className="text-[15px] font-semibold text-text-primary">{group.name}</span>
        {group.package && (
          <span className="text-[12px] text-text-secondary font-mono flex items-center gap-1"><Package size={11} /> {group.package}</span>
        )}
        <div className="ml-auto flex items-center gap-4">
          <span className="text-[13px]">
            <span className="text-[var(--success)]">{group.passed}</span>
            <span className="text-text-secondary"> / </span>
            <span className="text-text-primary">{group.total}</span>
          </span>
          <div className="w-20"><StatsBar passed={group.passed} failed={group.failed} skipped={group.skipped} total={group.total} /></div>
          {group.pass_rate !== null && (
            <span className="text-[12px] text-text-secondary w-12 text-right">{group.pass_rate.toFixed(0)}%</span>
          )}
          <span className="text-[12px] text-text-secondary w-14 text-right">{formatDuration(group.duration_ms)}</span>
        </div>
      </button>

      {/* Expanded: show test cases */}
      {expanded && (
        <div className="border-t border-[var(--border)]">
          <CasesPanel runId={runId} groupId={group.id} />
        </div>
      )}
    </div>
  );
}

// ─── Cases Panel (lazy-loads cases for a group) ────────────────────────

function CasesPanel({ runId, groupId }: { runId: string; groupId: string }) {
  const { cases, isLoading } = useGroupCases(runId, groupId);

  if (isLoading) {
    return (
      <div className="px-5 py-4 flex items-center gap-2 text-[13px] text-text-secondary">
        <Loader2 size={14} className="animate-spin" /> Loading cases...
      </div>
    );
  }

  if (cases.length === 0) {
    return <div className="px-5 py-4 text-[13px] text-text-secondary">No test cases</div>;
  }

  return (
    <div>
      {/* Table header */}
      <div className="flex items-center gap-3 px-5 py-2 text-[11px] uppercase tracking-wider text-text-secondary border-b border-[var(--border)] bg-bg-tertiary/30">
        <span className="w-5" />
        <span className="flex-1">Test Case</span>
        <span className="w-14 text-center">Status</span>
        <span className="w-16 text-right">Duration</span>
        <span className="w-5" />
      </div>
      {cases.map((tc) => (
        <CaseRow key={tc.id} testCase={tc} />
      ))}
    </div>
  );
}

// ─── Case Row ──────────────────────────────────────────────────────────

function CaseRow({ testCase }: { testCase: TestCase }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = testCase.error_message || testCase.stack_trace;

  const icon = testCase.status === "pass"
    ? <CheckCircle2 size={14} className="text-[var(--success)]" />
    : testCase.status === "fail"
      ? <XCircle size={14} className="text-[var(--danger)]" />
      : testCase.status === "skip"
        ? <MinusCircle size={14} className="text-[var(--warning)]" />
        : <MinusCircle size={14} className="text-text-secondary" />;

  return (
    <div className="border-b border-[var(--border)] last:border-b-0">
      <button
        onClick={() => hasDetails && setExpanded(!expanded)}
        className={`w-full flex items-center gap-3 px-5 py-2.5 text-left hover:bg-bg-tertiary/30 transition-colors ${hasDetails ? "cursor-pointer" : ""}`}
      >
        {icon}
        <span className="flex-1 text-[13px] font-mono text-text-primary truncate">{testCase.name}</span>
        {testCase.retry_count > 0 && (
          <span className="flex items-center gap-1 text-[11px] text-text-secondary"><RotateCw size={10} /> {testCase.retry_count}</span>
        )}
        <ResultStatusBadge status={testCase.status} />
        <span className="w-16 text-right text-[12px] text-text-secondary">{formatDuration(testCase.duration_ms)}</span>
        {testCase.logs_url ? (
          <a href={testCase.logs_url} target="_blank" rel="noopener noreferrer" className="text-accent" onClick={(e) => e.stopPropagation()}><ExternalLink size={12} /></a>
        ) : (
          <span className="w-3" />
        )}
        {hasDetails ? (
          expanded ? <ChevronDown size={14} className="text-text-secondary" /> : <ChevronRight size={14} className="text-text-secondary" />
        ) : <span className="w-3.5" />}
      </button>
      {expanded && (
        <div className="px-5 pb-3 space-y-2">
          {testCase.error_message && (
            <div>
              <div className="text-[11px] font-medium text-text-secondary uppercase tracking-wider mb-1">Error</div>
              <pre className="bg-bg-tertiary rounded-[6px] p-3 text-[12px] font-mono text-[var(--danger)] overflow-x-auto whitespace-pre-wrap max-h-48 overflow-y-auto">{testCase.error_message}</pre>
            </div>
          )}
          {testCase.stack_trace && (
            <div>
              <div className="text-[11px] font-medium text-text-secondary uppercase tracking-wider mb-1">Stack Trace</div>
              <pre className="bg-bg-tertiary rounded-[6px] p-3 text-[11px] font-mono text-text-secondary overflow-x-auto whitespace-pre-wrap max-h-48 overflow-y-auto">{testCase.stack_trace}</pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Flat Result Row (for old runs without hierarchy) ──────────────────

function FlatResultRow({ result }: { result: { id: string; test_name: string; status: string; duration_ms: number | null; error_message: string | null; log_output?: string | null } }) {
  const [expanded, setExpanded] = useState(false);
  const detail = result.error_message || result.log_output;
  const icon = result.status === "pass"
    ? <CheckCircle2 size={14} className="text-[var(--success)]" />
    : result.status === "fail" || result.status === "error"
      ? <XCircle size={14} className="text-[var(--danger)]" />
      : <MinusCircle size={14} className="text-[var(--warning)]" />;

  return (
    <div className="border-b border-[var(--border)] last:border-b-0">
      <button
        onClick={() => detail && setExpanded(!expanded)}
        className={`w-full flex items-center gap-3 px-5 py-2.5 text-left hover:bg-bg-tertiary/30 ${detail ? "cursor-pointer" : ""}`}
      >
        {icon}
        <span className="flex-1 text-[13px] font-mono text-text-primary truncate">{result.test_name}</span>
        <Badge variant={result.status === "pass" ? "success" : result.status === "fail" || result.status === "error" ? "danger" : "warning"}>{result.status}</Badge>
        <span className="w-16 text-right text-[12px] text-text-secondary">{formatDuration(result.duration_ms)}</span>
      </button>
      {expanded && detail && (
        <div className="px-5 pb-3">
          <pre className="bg-bg-tertiary rounded-[6px] p-3 text-[12px] font-mono text-[var(--danger)] overflow-x-auto whitespace-pre-wrap max-h-96 overflow-y-auto">{detail}</pre>
        </div>
      )}
    </div>
  );
}
