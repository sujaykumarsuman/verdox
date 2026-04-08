"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  X,
  Loader2,
  FileCode,
  FileText,
  Sparkles,
  CheckCircle2,
  Circle,
  XCircle,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { listWorkflowFiles, generateSuite } from "@/hooks/use-tests";
import type { WorkflowFile } from "@/types/test";

interface GenerateSuitePanelProps {
  repoId: string;
  open: boolean;
  onClose: () => void;
}

type Mode = "pick" | "paste";
type Phase = "idle" | "fetching" | "analyzing" | "processing";

const AI_MODELS = [
  { value: "gpt-4o", label: "GPT-4o" },
  { value: "gpt-4o-mini", label: "GPT-4o Mini" },
  { value: "gpt-4-turbo", label: "GPT-4 Turbo" },
  { value: "gpt-5.4-mini", label: "GPT-5.4 Mini" },
  { value: "gpt-5.4", label: "GPT-5.4" },
  { value: "o3-mini", label: "o3-mini (Reasoning)" },
];

const TIMEOUT_OPTIONS = [
  { value: 120, label: "2 min" },
  { value: 300, label: "5 min" },
  { value: 600, label: "10 min" },
];

const PHASE_LABELS: Record<Exclude<Phase, "idle">, string> = {
  fetching: "Fetching workflow...",
  analyzing: "Analyzing with AI...",
  processing: "Processing response...",
};

const PHASE_ORDER: Exclude<Phase, "idle">[] = [
  "fetching",
  "analyzing",
  "processing",
];

function formatElapsed(s: number): string {
  const mins = Math.floor(s / 60);
  const secs = s % 60;
  return `${mins}:${secs.toString().padStart(2, "0")}`;
}

export function GenerateSuitePanel({
  repoId,
  open,
  onClose,
}: GenerateSuitePanelProps) {
  const router = useRouter();

  // Input state
  const [mode, setMode] = useState<Mode>("pick");
  const [workflowFiles, setWorkflowFiles] = useState<WorkflowFile[]>([]);
  const [filesLoading, setFilesLoading] = useState(false);
  const [fileInput, setFileInput] = useState("");
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [pastedYaml, setPastedYaml] = useState("");
  const [selectedModel, setSelectedModel] = useState("gpt-4o");
  const [selectedTimeout, setSelectedTimeout] = useState(300);
  const [error, setError] = useState<string | null>(null);

  // Generation state
  const [importing, setImporting] = useState(false);
  const [phase, setPhase] = useState<Phase>("idle");
  const [elapsedSeconds, setElapsedSeconds] = useState(0);

  // Refs
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionsRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const phaseTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Slide-in animation state
  const [visible, setVisible] = useState(false);

  // Animate in when opened
  useEffect(() => {
    if (open) {
      requestAnimationFrame(() => setVisible(true));
    } else {
      setVisible(false);
    }
  }, [open]);

  const handleClose = useCallback(() => {
    if (importing) return;
    setVisible(false);
    setTimeout(() => onClose(), 300);
  }, [importing, onClose]);

  // Fetch workflow files when dialog opens
  useEffect(() => {
    if (!open) return;
    setError(null);
    setFileInput("");
    setPastedYaml("");
    setPhase("idle");
    setElapsedSeconds(0);

    const fetchFiles = async () => {
      setFilesLoading(true);
      try {
        const data = await listWorkflowFiles(repoId);
        setWorkflowFiles(data.files || []);
      } catch {
        setWorkflowFiles([]);
      } finally {
        setFilesLoading(false);
      }
    };
    fetchFiles();
  }, [open, repoId]);

  // Close suggestions on outside click
  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (
        suggestionsRef.current &&
        !suggestionsRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        setShowSuggestions(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  // Escape key to close (only when not importing)
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !importing) handleClose();
    };
    if (open) {
      document.addEventListener("keydown", handleKey);
      return () => document.removeEventListener("keydown", handleKey);
    }
  }, [open, importing, handleClose]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      abortRef.current?.abort();
      if (timerRef.current) clearInterval(timerRef.current);
      if (phaseTimerRef.current) clearTimeout(phaseTimerRef.current);
    };
  }, []);

  if (!open) return null;

  const filteredFiles = workflowFiles.filter((f) => {
    if (!fileInput) return true;
    const q = fileInput.toLowerCase();
    return (
      f.path.toLowerCase().includes(q) || f.name.toLowerCase().includes(q)
    );
  });

  const canSubmit =
    !importing &&
    ((mode === "pick" && fileInput.trim() !== "") ||
      (mode === "paste" && pastedYaml.trim() !== ""));

  const clearTimers = () => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    if (phaseTimerRef.current) {
      clearTimeout(phaseTimerRef.current);
      phaseTimerRef.current = null;
    }
  };

  const handleImport = async () => {
    setImporting(true);
    setError(null);
    setPhase("fetching");
    setElapsedSeconds(0);

    // Start elapsed timer
    timerRef.current = setInterval(() => {
      setElapsedSeconds((prev) => prev + 1);
    }, 1000);

    // After 3s, transition to analyzing phase
    phaseTimerRef.current = setTimeout(() => {
      setPhase("analyzing");
    }, 3000);

    // Create abort controller
    const controller = new AbortController();
    abortRef.current = controller;

    try {
      const payload =
        mode === "pick"
          ? { workflow_file: fileInput.trim(), model: selectedModel, timeout_seconds: selectedTimeout }
          : { workflow_yaml: pastedYaml, model: selectedModel, timeout_seconds: selectedTimeout };

      const result = await generateSuite(repoId, payload, controller.signal);

      clearTimers();
      setPhase("processing");

      // Brief pause to show processing phase, then navigate
      await new Promise((r) => setTimeout(r, 500));

      sessionStorage.setItem("verdox_generated_suite", JSON.stringify(result));
      toast.success("Workflow analysed — review and create your suite");
      onClose();
      router.push(`/repositories/${repoId}/suites/new`);
    } catch (err) {
      clearTimers();

      // User cancelled
      if (
        (err instanceof Error && err.name === "AbortError") ||
        controller.signal.aborted
      ) {
        setPhase("idle");
        setImporting(false);
        return;
      }

      // Timeout detection
      let msg: string;
      if (err instanceof Error && err.message === "Failed to fetch") {
        msg =
          "Generation timed out. The workflow may be too complex — try pasting a simplified version.";
      } else {
        msg = err instanceof Error ? err.message : "Import failed";
      }

      setError(msg);
      setPhase("idle");
      toast.error(msg);
    } finally {
      setImporting(false);
    }
  };

  const handleCancel = () => {
    abortRef.current?.abort();
    clearTimers();
    setImporting(false);
    setPhase("idle");
  };

  const isYamlValid =
    pastedYaml.trim() !== "" &&
    (pastedYaml.includes("on:") || pastedYaml.includes("on :")) &&
    pastedYaml.includes("jobs:");

  const isGenerating = phase !== "idle";

  return (
    <div className="fixed inset-0 z-50">
      {/* Backdrop */}
      <div
        className={`absolute inset-0 bg-black/50 transition-opacity duration-300 ${
          visible ? "opacity-100" : "opacity-0"
        }`}
        onClick={handleClose}
      />

      {/* Slide-in Panel */}
      <div
        className={`absolute top-0 right-0 h-full w-full max-w-xl bg-bg-secondary border-l border-[var(--border)] shadow-2xl transform transition-transform duration-300 ease-out flex flex-col ${
          visible ? "translate-x-0" : "translate-x-full"
        }`}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border)]">
          <h2 className="flex items-center gap-2 text-[18px] font-semibold text-text-primary">
            <Sparkles className="h-5 w-5 text-accent" />
            Generate Suite from Workflow
          </h2>
          <button
            onClick={handleClose}
            disabled={importing}
            className="text-text-secondary hover:text-text-primary transition-colors disabled:opacity-30"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5">
          {isGenerating ? (
            /* ── Generating State ── */
            <div className="flex flex-col h-full">
              <p className="text-[13px] text-text-secondary mb-6">
                Converting your workflow to a Verdox-compatible suite...
              </p>

              {/* Phase Steps */}
              <div className="space-y-4 mb-8">
                {PHASE_ORDER.map((p) => {
                  const phaseIdx = PHASE_ORDER.indexOf(phase as typeof p);
                  const stepIdx = PHASE_ORDER.indexOf(p);
                  const isDone = stepIdx < phaseIdx;
                  const isActive = p === phase;

                  return (
                    <div key={p} className="flex items-center gap-3">
                      {isDone ? (
                        <CheckCircle2
                          size={20}
                          className="text-[var(--success)] flex-shrink-0"
                        />
                      ) : isActive ? (
                        <Loader2
                          size={20}
                          className="text-accent animate-spin flex-shrink-0"
                        />
                      ) : (
                        <Circle
                          size={20}
                          className="text-text-secondary/30 flex-shrink-0"
                        />
                      )}
                      <span
                        className={`text-[14px] ${
                          isDone
                            ? "text-text-secondary line-through"
                            : isActive
                              ? "text-text-primary font-medium"
                              : "text-text-secondary/50"
                        }`}
                      >
                        {PHASE_LABELS[p]}
                      </span>
                      {isActive && (
                        <span className="text-[13px] text-text-secondary font-mono ml-auto">
                          {formatElapsed(elapsedSeconds)}
                        </span>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Info */}
              <div className="rounded-[8px] bg-bg-primary border border-[var(--border)] p-4 text-[13px] text-text-secondary">
                <p>Using <span className="font-medium text-text-primary">{AI_MODELS.find(m => m.value === selectedModel)?.label}</span> with {TIMEOUT_OPTIONS.find(t => t.value === selectedTimeout)?.label} timeout.</p>
                <p className="mt-1.5 text-text-secondary/60">
                  You can cancel and try again with a different model or simplified workflow.
                </p>
              </div>

              <div className="mt-auto pt-6">
                <Button variant="ghost" onClick={handleCancel} className="w-full">
                  <XCircle className="h-4 w-4 mr-2" />
                  Cancel Generation
                </Button>
              </div>
            </div>
          ) : (
            /* ── Input State ── */
            <>
              <p className="text-[13px] text-text-secondary mb-4">
                Provide an existing GitHub Actions workflow and Verdox will
                generate a compatible version that collects structured test
                results.
              </p>

              {/* Mode Tabs */}
              <div className="flex gap-1 mb-4 p-1 rounded-[6px] bg-bg-primary border">
                <button
                  onClick={() => setMode("pick")}
                  className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-[13px] font-medium rounded-[4px] transition-colors ${
                    mode === "pick"
                      ? "bg-accent text-white"
                      : "text-text-secondary hover:text-text-primary"
                  }`}
                >
                  <FileCode className="h-3.5 w-3.5" />
                  Select from Repo
                </button>
                <button
                  onClick={() => setMode("paste")}
                  className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-[13px] font-medium rounded-[4px] transition-colors ${
                    mode === "paste"
                      ? "bg-accent text-white"
                      : "text-text-secondary hover:text-text-primary"
                  }`}
                >
                  <FileText className="h-3.5 w-3.5" />
                  Paste YAML
                </button>
              </div>

              {/* Mode Content */}
              {mode === "pick" ? (
                <div className="relative">
                  <label className="block text-[13px] font-medium text-text-secondary mb-1.5">
                    Workflow File Path
                  </label>
                  <input
                    ref={inputRef}
                    type="text"
                    value={fileInput}
                    onChange={(e) => {
                      setFileInput(e.target.value);
                      setShowSuggestions(true);
                    }}
                    onFocus={() => setShowSuggestions(true)}
                    placeholder=".github/workflows/ci.yml"
                    className="w-full px-3 py-2 text-[14px] font-mono rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary placeholder:text-text-secondary/50 focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
                    autoComplete="off"
                  />

                  {showSuggestions &&
                    !filesLoading &&
                    filteredFiles.length > 0 && (
                      <div
                        ref={suggestionsRef}
                        className="absolute z-20 mt-1 w-full max-h-40 overflow-y-auto rounded-[6px] border bg-bg-primary shadow-lg"
                      >
                        {filteredFiles.map((f) => (
                          <button
                            key={f.path}
                            type="button"
                            onClick={() => {
                              setFileInput(f.path);
                              setShowSuggestions(false);
                            }}
                            className="w-full text-left px-3 py-1.5 text-[13px] font-mono text-text-primary hover:bg-accent/10 transition-colors"
                          >
                            {f.path}
                          </button>
                        ))}
                      </div>
                    )}

                  {filesLoading && (
                    <div className="flex items-center gap-2 text-[12px] text-text-secondary mt-1.5">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      Loading suggestions...
                    </div>
                  )}

                  {!filesLoading && workflowFiles.length === 0 && (
                    <p className="text-[12px] text-text-secondary mt-1.5">
                      No .yml/.yaml files found in .github/ — type a path
                      manually
                    </p>
                  )}
                </div>
              ) : (
                <div>
                  <label className="block text-[13px] font-medium text-text-secondary mb-1.5">
                    Workflow YAML
                  </label>
                  <textarea
                    value={pastedYaml}
                    onChange={(e) => setPastedYaml(e.target.value)}
                    placeholder="Paste your GitHub Actions workflow YAML here..."
                    rows={12}
                    className="w-full px-3 py-2 text-[13px] font-mono rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary placeholder:text-text-secondary/50 focus:outline-none focus:ring-1 focus:ring-[var(--accent)] resize-y"
                  />
                  {pastedYaml.trim() !== "" && !isYamlValid && (
                    <p className="text-[12px] text-yellow-500 mt-1">
                      Workflow should contain &apos;on:&apos; triggers and
                      &apos;jobs:&apos; definitions
                    </p>
                  )}
                </div>
              )}

              {/* Model & Timeout */}
              <div className="flex gap-3 mt-4">
                <div className="flex-1">
                  <label className="block text-[13px] font-medium text-text-secondary mb-1.5">
                    AI Model
                  </label>
                  <select
                    value={selectedModel}
                    onChange={(e) => setSelectedModel(e.target.value)}
                    className="w-full px-3 py-2 text-[13px] rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
                  >
                    {AI_MODELS.map((m) => (
                      <option key={m.value} value={m.value}>
                        {m.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex-1">
                  <label className="block text-[13px] font-medium text-text-secondary mb-1.5">
                    Timeout
                  </label>
                  <select
                    value={selectedTimeout}
                    onChange={(e) => setSelectedTimeout(Number(e.target.value))}
                    className="w-full px-3 py-2 text-[13px] rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
                  >
                    {TIMEOUT_OPTIONS.map((t) => (
                      <option key={t.value} value={t.value}>
                        {t.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Error */}
              {error && (
                <div className="mt-4 px-3 py-2.5 rounded-[6px] bg-[var(--danger)]/10 border border-[var(--danger)]/20 text-[var(--danger)] text-[13px] flex items-start gap-2">
                  <XCircle className="h-4 w-4 flex-shrink-0 mt-0.5" />
                  <span>{error}</span>
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer (input state only) */}
        {!isGenerating && (
          <div className="flex justify-end gap-3 px-6 py-4 border-t border-[var(--border)]">
            <Button variant="ghost" onClick={handleClose}>
              Cancel
            </Button>
            <Button onClick={handleImport} disabled={!canSubmit}>
              <Sparkles className="h-4 w-4 mr-1.5" />
              Generate
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
