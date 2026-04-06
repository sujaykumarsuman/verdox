"use client";

import { useState } from "react";
import { X, Container, GitBranch } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { createTestSuite } from "@/hooks/use-tests";
import { cn } from "@/lib/utils";
import type { DiscoverySuggestion } from "@/types/test";

interface CreateSuiteDialogProps {
  repoId: string;
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
  prefill?: DiscoverySuggestion;
}

const TYPE_SUGGESTIONS = ["unit", "integration", "e2e", "lint", "smoke", "build"];

export function CreateSuiteDialog({
  repoId,
  open,
  onClose,
  onSuccess,
  prefill,
}: CreateSuiteDialogProps) {
  const [step, setStep] = useState(1);
  const [name, setName] = useState(prefill?.name || "");
  const [type, setType] = useState(prefill?.type || "unit");
  const [executionMode, setExecutionMode] = useState<"container" | "gha">(
    (prefill?.execution_mode as "container" | "gha") || "container"
  );
  const [dockerImage, setDockerImage] = useState(prefill?.docker_image || "");
  const [testCommand, setTestCommand] = useState(prefill?.test_command || "");
  const [ghaWorkflowId, setGhaWorkflowId] = useState(prefill?.gha_workflow || "");
  const [timeout, setTimeout] = useState("300");
  const [configPath, setConfigPath] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) return null;

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);
    try {
      await createTestSuite(repoId, {
        name: name.trim(),
        type: type.trim(),
        execution_mode: executionMode,
        ...(dockerImage.trim() ? { docker_image: dockerImage.trim() } : {}),
        ...(testCommand.trim() ? { test_command: testCommand.trim() } : {}),
        ...(ghaWorkflowId.trim() ? { gha_workflow_id: ghaWorkflowId.trim() } : {}),
        ...(configPath.trim() ? { config_path: configPath.trim() } : {}),
        ...(timeout ? { timeout_seconds: parseInt(timeout, 10) } : {}),
      });
      resetForm();
      onSuccess();
      onClose();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create test suite"
      );
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setStep(1);
    setName("");
    setType("unit");
    setExecutionMode("container");
    setDockerImage("");
    setTestCommand("");
    setGhaWorkflowId("");
    setTimeout("300");
    setConfigPath("");
  };

  const canProceedStep1 = name.trim().length > 0 && type.trim().length > 0;
  const canSubmit = executionMode === "container"
    ? testCommand.trim().length > 0
    : ghaWorkflowId.trim().length > 0;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      <div className="relative z-10 w-full max-w-lg rounded-[8px] border bg-bg-secondary shadow-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-text-primary">
            Create Test Suite
            <span className="ml-2 text-sm font-normal text-text-secondary">
              Step {step} of 3
            </span>
          </h3>
          <button
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary"
          >
            <X size={20} />
          </button>
        </div>

        {/* Step indicators */}
        <div className="flex gap-2 mb-6">
          {[1, 2, 3].map((s) => (
            <div
              key={s}
              className={cn(
                "h-1 flex-1 rounded-full",
                s <= step ? "bg-accent" : "bg-bg-tertiary"
              )}
            />
          ))}
        </div>

        {/* Step 1: Basics */}
        {step === 1 && (
          <div className="space-y-4">
            <Input
              label="Suite Name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Unit Tests, Lint Check, E2E Tests"
              required
            />
            <div>
              <label className="block text-sm font-medium text-text-primary mb-1">
                Type
              </label>
              <input
                type="text"
                value={type}
                onChange={(e) => setType(e.target.value)}
                list="suite-types"
                className="w-full h-9 rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/30"
                placeholder="e.g., unit, integration, e2e, lint..."
              />
              <datalist id="suite-types">
                {TYPE_SUGGESTIONS.map((t) => (
                  <option key={t} value={t} />
                ))}
              </datalist>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <Button variant="secondary" size="sm" onClick={onClose}>
                Cancel
              </Button>
              <Button size="sm" disabled={!canProceedStep1} onClick={() => setStep(2)}>
                Next
              </Button>
            </div>
          </div>
        )}

        {/* Step 2: Execution Mode */}
        {step === 2 && (
          <div className="space-y-4">
            <p className="text-sm text-text-secondary">
              How should this test suite be executed?
            </p>
            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setExecutionMode("container")}
                className={cn(
                  "p-4 rounded-[8px] border text-left transition-colors",
                  executionMode === "container"
                    ? "border-accent bg-[var(--accent-subtle)]"
                    : "border-[var(--border)] hover:border-accent/50"
                )}
              >
                <Container className="h-6 w-6 text-accent mb-2" />
                <div className="font-medium text-text-primary text-sm">Container</div>
                <p className="text-[12px] text-text-secondary mt-1">
                  Run command in Docker container. Simple scripts and tests.
                </p>
              </button>
              <button
                type="button"
                onClick={() => setExecutionMode("gha")}
                className={cn(
                  "p-4 rounded-[8px] border text-left transition-colors",
                  executionMode === "gha"
                    ? "border-accent bg-[var(--accent-subtle)]"
                    : "border-[var(--border)] hover:border-accent/50"
                )}
              >
                <GitBranch className="h-6 w-6 text-accent mb-2" />
                <div className="font-medium text-text-primary text-sm">GitHub Actions</div>
                <p className="text-[12px] text-text-secondary mt-1">
                  Trigger a GHA workflow. Full environment setup.
                </p>
              </button>
            </div>
            <div className="flex justify-between pt-2">
              <Button variant="ghost" size="sm" onClick={() => setStep(1)}>
                Back
              </Button>
              <Button size="sm" onClick={() => setStep(3)}>
                Next
              </Button>
            </div>
          </div>
        )}

        {/* Step 3: Configuration */}
        {step === 3 && (
          <div className="space-y-4">
            {executionMode === "container" ? (
              <>
                <Input
                  label="Docker Image"
                  value={dockerImage}
                  onChange={(e) => setDockerImage(e.target.value)}
                  placeholder="e.g., golang:1.26-alpine, node:22-alpine"
                />
                <div>
                  <label className="block text-sm font-medium text-text-primary mb-1">
                    Test Command
                  </label>
                  <textarea
                    value={testCommand}
                    onChange={(e) => setTestCommand(e.target.value)}
                    className="w-full min-h-[80px] rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 py-2 text-sm font-mono text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/30"
                    placeholder="e.g., go test -v -json ./..."
                    required
                  />
                </div>
                <Input
                  label="Config Path (optional)"
                  value={configPath}
                  onChange={(e) => setConfigPath(e.target.value)}
                  placeholder="e.g., verdox.yaml"
                />
              </>
            ) : (
              <>
                <Input
                  label="Workflow File"
                  value={ghaWorkflowId}
                  onChange={(e) => setGhaWorkflowId(e.target.value)}
                  placeholder="e.g., test.yml"
                  required
                />
                <p className="text-[12px] text-text-secondary">
                  The workflow file name in .github/workflows/. The workflow must accept
                  verdox_run_id, branch, and commit_hash inputs.
                </p>
              </>
            )}

            <Input
              label="Timeout (seconds)"
              type="number"
              value={timeout}
              onChange={(e) => setTimeout(e.target.value)}
              placeholder="300"
            />

            {error && (
              <p className="text-sm text-[var(--danger)]">{error}</p>
            )}

            <div className="flex justify-between pt-2">
              <Button variant="ghost" size="sm" onClick={() => setStep(2)}>
                Back
              </Button>
              <Button
                size="sm"
                onClick={handleSubmit}
                loading={loading}
                disabled={!canSubmit}
              >
                Create Suite
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
