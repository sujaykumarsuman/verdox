"use client";

import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { X, Loader2, FileCode, FileText, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { listWorkflowFiles, importSuite } from "@/hooks/use-tests";
import type { WorkflowFile } from "@/types/test";

interface ImportSuiteDialogProps {
  repoId: string;
  open: boolean;
  onClose: () => void;
}

type Mode = "pick" | "paste";

export function ImportSuiteDialog({ repoId, open, onClose }: ImportSuiteDialogProps) {
  const router = useRouter();
  const [mode, setMode] = useState<Mode>("pick");
  const [workflowFiles, setWorkflowFiles] = useState<WorkflowFile[]>([]);
  const [filesLoading, setFilesLoading] = useState(false);
  const [fileInput, setFileInput] = useState("");
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [pastedYaml, setPastedYaml] = useState("");
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionsRef = useRef<HTMLDivElement>(null);

  // Fetch workflow files when dialog opens
  useEffect(() => {
    if (!open) return;
    setError(null);
    setFileInput("");
    setPastedYaml("");

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

  if (!open) return null;

  // Filter suggestions based on input
  const filteredFiles = workflowFiles.filter((f) => {
    if (!fileInput) return true;
    const q = fileInput.toLowerCase();
    return f.path.toLowerCase().includes(q) || f.name.toLowerCase().includes(q);
  });

  const canSubmit =
    !importing &&
    ((mode === "pick" && fileInput.trim() !== "") ||
      (mode === "paste" && pastedYaml.trim() !== ""));

  const handleImport = async () => {
    setImporting(true);
    setError(null);
    try {
      const payload =
        mode === "pick"
          ? { workflow_file: fileInput.trim() }
          : { workflow_yaml: pastedYaml };

      const result = await importSuite(repoId, payload);

      sessionStorage.setItem("verdox_import_suite", JSON.stringify(result));
      toast.success("Workflow analysed — review and create your suite");
      onClose();
      router.push(`/repositories/${repoId}/suites/new`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Import failed";
      setError(msg);
      toast.error(msg);
    } finally {
      setImporting(false);
    }
  };

  const isYamlValid =
    pastedYaml.trim() !== "" &&
    (pastedYaml.includes("on:") || pastedYaml.includes("on :")) &&
    pastedYaml.includes("jobs:");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      <div className="relative z-10 w-full max-w-lg rounded-[8px] border bg-bg-secondary shadow-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="flex items-center gap-2 text-[18px] font-semibold text-text-primary">
            <Sparkles className="h-5 w-5 text-accent" />
            Generate Suite from Workflow
          </h2>
          <button
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <p className="text-[13px] text-text-secondary mb-4">
          Provide an existing GitHub Actions workflow and Verdox will generate a
          compatible version that collects structured test results.
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

            {/* Suggestions dropdown */}
            {showSuggestions && !filesLoading && filteredFiles.length > 0 && (
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
                No .yml/.yaml files found in .github/ — type a path manually
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
              rows={10}
              className="w-full px-3 py-2 text-[13px] font-mono rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary placeholder:text-text-secondary/50 focus:outline-none focus:ring-1 focus:ring-[var(--accent)] resize-y"
            />
            {pastedYaml.trim() !== "" && !isYamlValid && (
              <p className="text-[12px] text-yellow-500 mt-1">
                Workflow should contain &apos;on:&apos; triggers and &apos;jobs:&apos; definitions
              </p>
            )}
          </div>
        )}

        {/* Error */}
        {error && (
          <p className="text-[13px] text-danger mt-3">{error}</p>
        )}

        {/* Actions */}
        <div className="flex justify-end gap-3 mt-6">
          <Button variant="ghost" onClick={onClose} disabled={importing}>
            Cancel
          </Button>
          <Button onClick={handleImport} disabled={!canSubmit} loading={importing}>
            {importing ? "Generating..." : "Generate"}
          </Button>
        </div>
      </div>
    </div>
  );
}
