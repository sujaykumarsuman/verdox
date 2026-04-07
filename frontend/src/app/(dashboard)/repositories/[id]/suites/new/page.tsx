"use client";

import { use, useState, useMemo, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  Plus,
  Trash2,
  ChevronDown,
  ChevronUp,
  Copy,
  Check,
  Terminal,
  Puzzle,
  GripVertical,
} from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { generateWorkflowYaml } from "@/lib/generate-workflow-yaml";
import type {
  WorkflowConfig,
  WorkflowService,
  WorkflowStep,
} from "@/types/test";

// ─── Constants ────────────────────────────────────────────────────────────────

const TYPE_SUGGESTIONS = ["unit", "integration", "e2e", "lint", "smoke", "build"];

const RUNNER_OPTIONS = [
  { value: "ubuntu-latest", label: "Ubuntu (latest)" },
  { value: "ubuntu-22.04", label: "Ubuntu 22.04" },
  { value: "ubuntu-24.04", label: "Ubuntu 24.04" },
  { value: "macos-latest", label: "macOS (latest)" },
  { value: "windows-latest", label: "Windows (latest)" },
];

const SERVICE_PRESETS: Record<
  string,
  { name: string; image: string; ports: string[]; env: Record<string, string> }
> = {
  PostgreSQL: {
    name: "postgres",
    image: "postgres:16-alpine",
    ports: ["5432:5432"],
    env: {
      POSTGRES_USER: "test",
      POSTGRES_PASSWORD: "test",
      POSTGRES_DB: "test_db",
    },
  },
  Redis: {
    name: "redis",
    image: "redis:7-alpine",
    ports: ["6379:6379"],
    env: {},
  },
  MySQL: {
    name: "mysql",
    image: "mysql:8.0",
    ports: ["3306:3306"],
    env: {
      MYSQL_ROOT_PASSWORD: "test",
      MYSQL_DATABASE: "test_db",
    },
  },
};

const STEP_PRESETS: Record<
  string,
  { name: string; uses: string; with: Record<string, string> }
> = {
  "Setup Node.js": {
    name: "Setup Node.js",
    uses: "actions/setup-node@v4",
    with: { "node-version": "20" },
  },
  "Setup Python": {
    name: "Setup Python",
    uses: "actions/setup-python@v5",
    with: { "python-version": "3.12" },
  },
  "Setup Go": {
    name: "Setup Go",
    uses: "actions/setup-go@v5",
    with: { "go-version": "1.22" },
  },
};

// ─── Helper types for form state ──────────────────────────────────────────────

interface FormService {
  id: string;
  name: string;
  image: string;
  ports: string[];
  env: { key: string; value: string }[];
}

interface FormStep {
  id: string;
  name: string;
  mode: "run" | "uses";
  run: string;
  uses: string;
  with: { key: string; value: string }[];
}

let idCounter = 0;
function uniqueId() {
  return `_${++idCounter}_${Date.now()}`;
}

// ─── Page Component ───────────────────────────────────────────────────────────

export default function CreateSuitePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id: repoId } = use(params);
  const router = useRouter();

  // Section 1: Suite Info
  const [name, setName] = useState("");
  const [type, setType] = useState("unit");
  const [timeout, setTimeout] = useState(300);

  // Section 2: Runner
  const [runnerOS, setRunnerOS] = useState("ubuntu-latest");

  // Section 3: Env Vars
  const [envVars, setEnvVars] = useState<{ key: string; value: string }[]>([]);

  // Section 4: Services
  const [services, setServices] = useState<FormService[]>([]);
  const [servicesOpen, setServicesOpen] = useState(true);

  // Section 5: Setup Steps
  const [setupSteps, setSetupSteps] = useState<FormStep[]>([]);

  // Section 6: Test Command
  const [testCommand, setTestCommand] = useState("");

  // Section 7: Advanced
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [matrixEnabled, setMatrixEnabled] = useState(false);
  const [matrixDimensions, setMatrixDimensions] = useState<
    { key: string; values: string }[]
  >([]);
  const [matrixFailFast, setMatrixFailFast] = useState(true);
  const [concurrencyEnabled, setConcurrencyEnabled] = useState(false);
  const [concurrencyGroup, setConcurrencyGroup] = useState(
    "verdox-${{ github.ref }}"
  );
  const [concurrencyCancelInProgress, setConcurrencyCancelInProgress] =
    useState(true);

  // Submit state
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Copy button state
  const [copied, setCopied] = useState(false);

  // ─── Build WorkflowConfig from form state ─────────────────────────────────

  const workflowConfig: WorkflowConfig = useMemo(() => {
    const envObj: Record<string, string> = {};
    for (const e of envVars) {
      if (e.key.trim()) envObj[e.key.trim()] = e.value;
    }

    const svcList: WorkflowService[] = services
      .filter((s) => s.name.trim() && s.image.trim())
      .map((s) => {
        const svcEnv: Record<string, string> = {};
        for (const e of s.env) {
          if (e.key.trim()) svcEnv[e.key.trim()] = e.value;
        }
        return {
          name: s.name.trim(),
          image: s.image.trim(),
          ports: s.ports.filter((p) => p.trim()),
          env: svcEnv,
        };
      });

    const steps: WorkflowStep[] = setupSteps
      .filter((s) => s.name.trim())
      .map((s) => {
        if (s.mode === "uses") {
          const withObj: Record<string, string> = {};
          for (const w of s.with) {
            if (w.key.trim()) withObj[w.key.trim()] = w.value;
          }
          return {
            name: s.name.trim(),
            uses: s.uses.trim(),
            with: withObj,
          };
        }
        return {
          name: s.name.trim(),
          run: s.run,
        };
      });

    const cfg: WorkflowConfig = {
      runner_os: runnerOS,
      env_vars: envObj,
      services: svcList,
      setup_steps: steps,
      matrix: null,
      concurrency: null,
    };

    if (matrixEnabled && matrixDimensions.length > 0) {
      const dims: Record<string, string[]> = {};
      for (const d of matrixDimensions) {
        if (d.key.trim()) {
          dims[d.key.trim()] = d.values.split(",").map((v) => v.trim()).filter(Boolean);
        }
      }
      if (Object.keys(dims).length > 0) {
        cfg.matrix = { dimensions: dims, fail_fast: matrixFailFast };
      }
    }

    if (concurrencyEnabled) {
      cfg.concurrency = {
        group: concurrencyGroup,
        cancel_in_progress: concurrencyCancelInProgress,
      };
    }

    return cfg;
  }, [
    runnerOS,
    envVars,
    services,
    setupSteps,
    matrixEnabled,
    matrixDimensions,
    matrixFailFast,
    concurrencyEnabled,
    concurrencyGroup,
    concurrencyCancelInProgress,
  ]);

  // ─── YAML preview ────────────────────────────────────────────────────────

  const yamlPreview = useMemo(
    () => generateWorkflowYaml(workflowConfig, testCommand || "make test", name || undefined),
    [workflowConfig, testCommand, name]
  );

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(yamlPreview);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  }, [yamlPreview]);

  // ─── Submit ───────────────────────────────────────────────────────────────

  const canSubmit = name.trim().length > 0 && type.trim().length > 0;

  const handleSubmit = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    setError(null);
    try {
      const body = {
        name: name.trim(),
        type: type.trim(),
        execution_mode: "fork_gha",
        test_command: testCommand.trim() || null,
        timeout_seconds: timeout,
        workflow_config: workflowConfig,
      };

      await api(`/v1/repositories/${repoId}/suites`, {
        method: "POST",
        body: JSON.stringify(body),
      });

      toast.success("Test suite created");
      router.push(`/repositories/${repoId}`);
    } catch (err) {
      const msg =
        err instanceof Error ? err.message : "Failed to create test suite";
      setError(msg);
      toast.error(msg);
    } finally {
      setSubmitting(false);
    }
  };

  // ─── Env var helpers ──────────────────────────────────────────────────────

  const addEnvVar = () =>
    setEnvVars((prev) => [...prev, { key: "", value: "" }]);
  const removeEnvVar = (idx: number) =>
    setEnvVars((prev) => prev.filter((_, i) => i !== idx));
  const updateEnvVar = (
    idx: number,
    field: "key" | "value",
    val: string
  ) =>
    setEnvVars((prev) =>
      prev.map((e, i) => (i === idx ? { ...e, [field]: val } : e))
    );

  // ─── Service helpers ──────────────────────────────────────────────────────

  const addService = () =>
    setServices((prev) => [
      ...prev,
      { id: uniqueId(), name: "", image: "", ports: [""], env: [] },
    ]);

  const addServicePreset = (presetName: string) => {
    const p = SERVICE_PRESETS[presetName];
    if (!p) return;
    setServices((prev) => [
      ...prev,
      {
        id: uniqueId(),
        name: p.name,
        image: p.image,
        ports: [...p.ports],
        env: Object.entries(p.env).map(([key, value]) => ({ key, value })),
      },
    ]);
  };

  const removeService = (svcId: string) =>
    setServices((prev) => prev.filter((s) => s.id !== svcId));

  const updateService = (
    svcId: string,
    field: "name" | "image",
    val: string
  ) =>
    setServices((prev) =>
      prev.map((s) => (s.id === svcId ? { ...s, [field]: val } : s))
    );

  const addServicePort = (svcId: string) =>
    setServices((prev) =>
      prev.map((s) =>
        s.id === svcId ? { ...s, ports: [...s.ports, ""] } : s
      )
    );

  const removeServicePort = (svcId: string, idx: number) =>
    setServices((prev) =>
      prev.map((s) =>
        s.id === svcId
          ? { ...s, ports: s.ports.filter((_, i) => i !== idx) }
          : s
      )
    );

  const updateServicePort = (svcId: string, idx: number, val: string) =>
    setServices((prev) =>
      prev.map((s) =>
        s.id === svcId
          ? { ...s, ports: s.ports.map((p, i) => (i === idx ? val : p)) }
          : s
      )
    );

  const addServiceEnv = (svcId: string) =>
    setServices((prev) =>
      prev.map((s) =>
        s.id === svcId
          ? { ...s, env: [...s.env, { key: "", value: "" }] }
          : s
      )
    );

  const removeServiceEnv = (svcId: string, idx: number) =>
    setServices((prev) =>
      prev.map((s) =>
        s.id === svcId
          ? { ...s, env: s.env.filter((_, i) => i !== idx) }
          : s
      )
    );

  const updateServiceEnv = (
    svcId: string,
    idx: number,
    field: "key" | "value",
    val: string
  ) =>
    setServices((prev) =>
      prev.map((s) =>
        s.id === svcId
          ? {
              ...s,
              env: s.env.map((e, i) =>
                i === idx ? { ...e, [field]: val } : e
              ),
            }
          : s
      )
    );

  // ─── Step helpers ─────────────────────────────────────────────────────────

  const addStep = () =>
    setSetupSteps((prev) => [
      ...prev,
      {
        id: uniqueId(),
        name: "",
        mode: "run" as const,
        run: "",
        uses: "",
        with: [],
      },
    ]);

  const addStepPreset = (presetName: string) => {
    const p = STEP_PRESETS[presetName];
    if (!p) return;
    setSetupSteps((prev) => [
      ...prev,
      {
        id: uniqueId(),
        name: p.name,
        mode: "uses" as const,
        run: "",
        uses: p.uses,
        with: Object.entries(p.with).map(([key, value]) => ({ key, value })),
      },
    ]);
  };

  const removeStep = (stepId: string) =>
    setSetupSteps((prev) => prev.filter((s) => s.id !== stepId));

  const updateStep = (
    stepId: string,
    field: "name" | "mode" | "run" | "uses",
    val: string
  ) =>
    setSetupSteps((prev) =>
      prev.map((s) => (s.id === stepId ? { ...s, [field]: val } : s))
    );

  const addStepWith = (stepId: string) =>
    setSetupSteps((prev) =>
      prev.map((s) =>
        s.id === stepId
          ? { ...s, with: [...s.with, { key: "", value: "" }] }
          : s
      )
    );

  const removeStepWith = (stepId: string, idx: number) =>
    setSetupSteps((prev) =>
      prev.map((s) =>
        s.id === stepId
          ? { ...s, with: s.with.filter((_, i) => i !== idx) }
          : s
      )
    );

  const updateStepWith = (
    stepId: string,
    idx: number,
    field: "key" | "value",
    val: string
  ) =>
    setSetupSteps((prev) =>
      prev.map((s) =>
        s.id === stepId
          ? {
              ...s,
              with: s.with.map((w, i) =>
                i === idx ? { ...w, [field]: val } : w
              ),
            }
          : s
      )
    );

  const moveStep = (stepId: string, direction: "up" | "down") =>
    setSetupSteps((prev) => {
      const idx = prev.findIndex((s) => s.id === stepId);
      if (idx < 0) return prev;
      const targetIdx = direction === "up" ? idx - 1 : idx + 1;
      if (targetIdx < 0 || targetIdx >= prev.length) return prev;
      const next = [...prev];
      [next[idx], next[targetIdx]] = [next[targetIdx], next[idx]];
      return next;
    });

  // ─── Matrix helpers ───────────────────────────────────────────────────────

  const addMatrixDimension = () =>
    setMatrixDimensions((prev) => [...prev, { key: "", values: "" }]);
  const removeMatrixDimension = (idx: number) =>
    setMatrixDimensions((prev) => prev.filter((_, i) => i !== idx));
  const updateMatrixDimension = (
    idx: number,
    field: "key" | "values",
    val: string
  ) =>
    setMatrixDimensions((prev) =>
      prev.map((d, i) => (i === idx ? { ...d, [field]: val } : d))
    );

  // ─── Shared styles ───────────────────────────────────────────────────────

  const selectClass =
    "w-full h-9 rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/30 appearance-none";
  const textareaClass =
    "w-full min-h-[80px] rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 py-2 text-sm font-mono text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/30 resize-y";
  const miniInputClass =
    "h-8 rounded-[4px] border border-[var(--border)] bg-bg-primary px-2 text-[13px] text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/30";
  const sectionTitle =
    "text-[15px] font-semibold text-text-primary flex items-center gap-2";
  const sectionDesc = "text-[13px] text-text-secondary mt-0.5 mb-3";

  // ─── Render ───────────────────────────────────────────────────────────────

  return (
    <div className="flex gap-0 min-h-[calc(100vh-64px)]">
      {/* Left panel: Form */}
      <div className="w-[55%] min-w-0 overflow-y-auto px-8 py-6 border-r border-[var(--border)]">
        {/* Breadcrumb */}
        <Link
          href={`/repositories/${repoId}`}
          className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Repository
        </Link>

        <h1 className="font-display text-[24px] leading-[32px] tracking-[-0.01em] text-text-primary mb-6">
          Create Test Suite
        </h1>

        <div className="space-y-8 pb-8">
          {/* ───── Section 1: Suite Info ───── */}
          <section>
            <h2 className={sectionTitle}>Suite Info</h2>
            <p className={sectionDesc}>Basic information about the test suite.</p>
            <div className="space-y-4">
              <Input
                label="Name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Unit Tests, Lint Check, E2E Tests"
                required
              />
              <div>
                <label className="block text-[14px] font-medium text-text-primary mb-1.5">
                  Type
                </label>
                <input
                  type="text"
                  value={type}
                  onChange={(e) => setType(e.target.value)}
                  list="suite-types"
                  className={selectClass}
                  placeholder="e.g., unit, integration, e2e, lint..."
                />
                <datalist id="suite-types">
                  {TYPE_SUGGESTIONS.map((t) => (
                    <option key={t} value={t} />
                  ))}
                </datalist>
              </div>
              <div>
                <label className="block text-[14px] font-medium text-text-primary mb-1.5">
                  Timeout (seconds)
                </label>
                <input
                  type="number"
                  value={timeout}
                  onChange={(e) =>
                    setTimeout(Math.max(30, Math.min(3600, Number(e.target.value) || 300)))
                  }
                  min={30}
                  max={3600}
                  className={cn(selectClass, "w-32")}
                />
                <p className="text-[12px] text-text-secondary mt-1">
                  Between 30 and 3600 seconds.
                </p>
              </div>
            </div>
          </section>

          {/* ───── Section 2: Runner ───── */}
          <section>
            <h2 className={sectionTitle}>Runner Configuration</h2>
            <p className={sectionDesc}>
              Choose the GitHub Actions runner environment.
            </p>
            <div>
              <label className="block text-[14px] font-medium text-text-primary mb-1.5">
                Runner OS
              </label>
              <select
                value={runnerOS}
                onChange={(e) => setRunnerOS(e.target.value)}
                className={selectClass}
              >
                {RUNNER_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </section>

          {/* ───── Section 3: Env Vars ───── */}
          <section>
            <h2 className={sectionTitle}>Environment Variables</h2>
            <p className={sectionDesc}>
              Global environment variables available to all steps.
            </p>
            {envVars.map((ev, idx) => (
              <div key={idx} className="flex items-center gap-2 mb-2">
                <input
                  type="text"
                  value={ev.key}
                  onChange={(e) => updateEnvVar(idx, "key", e.target.value)}
                  placeholder="KEY"
                  className={cn(miniInputClass, "flex-1 font-mono")}
                />
                <input
                  type="text"
                  value={ev.value}
                  onChange={(e) => updateEnvVar(idx, "value", e.target.value)}
                  placeholder="value"
                  className={cn(miniInputClass, "flex-1 font-mono")}
                />
                <button
                  type="button"
                  onClick={() => removeEnvVar(idx)}
                  className="p-1 text-text-secondary hover:text-danger transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
            <Button variant="ghost" size="sm" onClick={addEnvVar}>
              <Plus className="h-3.5 w-3.5" />
              Add Variable
            </Button>
          </section>

          {/* ───── Section 4: Services ───── */}
          <section>
            <button
              type="button"
              onClick={() => setServicesOpen(!servicesOpen)}
              className="w-full flex items-center justify-between"
            >
              <h2 className={sectionTitle}>
                Service Containers
                {services.length > 0 && (
                  <span className="text-[12px] font-normal text-text-secondary">
                    ({services.length})
                  </span>
                )}
              </h2>
              {servicesOpen ? (
                <ChevronUp className="h-4 w-4 text-text-secondary" />
              ) : (
                <ChevronDown className="h-4 w-4 text-text-secondary" />
              )}
            </button>
            {servicesOpen && (
              <div className="mt-2">
                <p className={sectionDesc}>
                  Add service containers like databases for your test environment.
                </p>

                {/* Preset buttons */}
                <div className="flex items-center gap-2 mb-3">
                  <span className="text-[12px] text-text-secondary">Presets:</span>
                  {Object.keys(SERVICE_PRESETS).map((preset) => (
                    <button
                      key={preset}
                      type="button"
                      onClick={() => addServicePreset(preset)}
                      className="px-2 py-1 rounded-[4px] border border-[var(--border)] text-[12px] text-text-secondary hover:text-text-primary hover:border-accent/50 transition-colors"
                    >
                      {preset}
                    </button>
                  ))}
                </div>

                {services.map((svc) => (
                  <div
                    key={svc.id}
                    className="rounded-[8px] border border-[var(--border)] bg-bg-primary p-4 mb-3"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex-1 grid grid-cols-2 gap-3">
                        <div>
                          <label className="block text-[12px] font-medium text-text-secondary mb-1">
                            Service Name
                          </label>
                          <input
                            type="text"
                            value={svc.name}
                            onChange={(e) =>
                              updateService(svc.id, "name", e.target.value)
                            }
                            placeholder="e.g., postgres"
                            className={miniInputClass + " w-full"}
                          />
                        </div>
                        <div>
                          <label className="block text-[12px] font-medium text-text-secondary mb-1">
                            Docker Image
                          </label>
                          <input
                            type="text"
                            value={svc.image}
                            onChange={(e) =>
                              updateService(svc.id, "image", e.target.value)
                            }
                            placeholder="e.g., postgres:16-alpine"
                            className={miniInputClass + " w-full"}
                          />
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => removeService(svc.id)}
                        className="ml-3 p-1 text-text-secondary hover:text-danger transition-colors"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>

                    {/* Ports */}
                    <div className="mb-2">
                      <label className="block text-[12px] font-medium text-text-secondary mb-1">
                        Ports
                      </label>
                      {svc.ports.map((port, pIdx) => (
                        <div key={pIdx} className="flex items-center gap-2 mb-1">
                          <input
                            type="text"
                            value={port}
                            onChange={(e) =>
                              updateServicePort(svc.id, pIdx, e.target.value)
                            }
                            placeholder="host:container (e.g., 5432:5432)"
                            className={cn(miniInputClass, "flex-1 font-mono")}
                          />
                          <button
                            type="button"
                            onClick={() => removeServicePort(svc.id, pIdx)}
                            className="p-1 text-text-secondary hover:text-danger transition-colors"
                          >
                            <Trash2 className="h-3 w-3" />
                          </button>
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={() => addServicePort(svc.id)}
                        className="text-[12px] text-accent hover:underline"
                      >
                        + Add Port
                      </button>
                    </div>

                    {/* Service Env Vars */}
                    <div>
                      <label className="block text-[12px] font-medium text-text-secondary mb-1">
                        Environment Variables
                      </label>
                      {svc.env.map((ev, eIdx) => (
                        <div key={eIdx} className="flex items-center gap-2 mb-1">
                          <input
                            type="text"
                            value={ev.key}
                            onChange={(e) =>
                              updateServiceEnv(
                                svc.id,
                                eIdx,
                                "key",
                                e.target.value
                              )
                            }
                            placeholder="KEY"
                            className={cn(miniInputClass, "flex-1 font-mono")}
                          />
                          <input
                            type="text"
                            value={ev.value}
                            onChange={(e) =>
                              updateServiceEnv(
                                svc.id,
                                eIdx,
                                "value",
                                e.target.value
                              )
                            }
                            placeholder="value"
                            className={cn(miniInputClass, "flex-1 font-mono")}
                          />
                          <button
                            type="button"
                            onClick={() => removeServiceEnv(svc.id, eIdx)}
                            className="p-1 text-text-secondary hover:text-danger transition-colors"
                          >
                            <Trash2 className="h-3 w-3" />
                          </button>
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={() => addServiceEnv(svc.id)}
                        className="text-[12px] text-accent hover:underline"
                      >
                        + Add Env Var
                      </button>
                    </div>
                  </div>
                ))}

                <Button variant="ghost" size="sm" onClick={addService}>
                  <Plus className="h-3.5 w-3.5" />
                  Add Service
                </Button>
              </div>
            )}
          </section>

          {/* ───── Section 5: Setup Steps ───── */}
          <section>
            <h2 className={sectionTitle}>Setup Steps</h2>
            <p className={sectionDesc}>
              Steps to run before the test command (e.g., install dependencies, set up language runtime).
            </p>

            {/* Step Presets */}
            <div className="flex items-center gap-2 mb-3">
              <span className="text-[12px] text-text-secondary">Presets:</span>
              {Object.keys(STEP_PRESETS).map((preset) => (
                <button
                  key={preset}
                  type="button"
                  onClick={() => addStepPreset(preset)}
                  className="px-2 py-1 rounded-[4px] border border-[var(--border)] text-[12px] text-text-secondary hover:text-text-primary hover:border-accent/50 transition-colors"
                >
                  {preset}
                </button>
              ))}
            </div>

            {setupSteps.map((step, stepIdx) => (
              <div
                key={step.id}
                className="rounded-[8px] border border-[var(--border)] bg-bg-primary p-4 mb-3"
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <GripVertical className="h-4 w-4 text-text-secondary/50" />
                    <input
                      type="text"
                      value={step.name}
                      onChange={(e) =>
                        updateStep(step.id, "name", e.target.value)
                      }
                      placeholder="Step name"
                      className={cn(miniInputClass, "w-56")}
                    />
                  </div>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => moveStep(step.id, "up")}
                      disabled={stepIdx === 0}
                      className="p-1 text-text-secondary hover:text-text-primary disabled:opacity-30 transition-colors"
                    >
                      <ChevronUp className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => moveStep(step.id, "down")}
                      disabled={stepIdx === setupSteps.length - 1}
                      className="p-1 text-text-secondary hover:text-text-primary disabled:opacity-30 transition-colors"
                    >
                      <ChevronDown className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => removeStep(step.id)}
                      className="p-1 text-text-secondary hover:text-danger transition-colors"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>

                {/* Mode toggle */}
                <div className="flex items-center gap-1 mb-3">
                  <button
                    type="button"
                    onClick={() => updateStep(step.id, "mode", "run")}
                    className={cn(
                      "flex items-center gap-1.5 px-3 py-1.5 rounded-[4px] text-[12px] font-medium transition-colors",
                      step.mode === "run"
                        ? "bg-accent text-white"
                        : "bg-bg-secondary text-text-secondary hover:text-text-primary"
                    )}
                  >
                    <Terminal className="h-3 w-3" />
                    Run command
                  </button>
                  <button
                    type="button"
                    onClick={() => updateStep(step.id, "mode", "uses")}
                    className={cn(
                      "flex items-center gap-1.5 px-3 py-1.5 rounded-[4px] text-[12px] font-medium transition-colors",
                      step.mode === "uses"
                        ? "bg-accent text-white"
                        : "bg-bg-secondary text-text-secondary hover:text-text-primary"
                    )}
                  >
                    <Puzzle className="h-3 w-3" />
                    Use action
                  </button>
                </div>

                {step.mode === "run" ? (
                  <textarea
                    value={step.run}
                    onChange={(e) =>
                      updateStep(step.id, "run", e.target.value)
                    }
                    placeholder={"e.g.,\nnpm install\nnpm run build"}
                    className={textareaClass}
                  />
                ) : (
                  <div className="space-y-2">
                    <input
                      type="text"
                      value={step.uses}
                      onChange={(e) =>
                        updateStep(step.id, "uses", e.target.value)
                      }
                      placeholder="e.g., actions/setup-node@v4"
                      className={cn(miniInputClass, "w-full font-mono")}
                    />
                    <div>
                      <label className="block text-[12px] font-medium text-text-secondary mb-1">
                        With
                      </label>
                      {step.with.map((w, wIdx) => (
                        <div
                          key={wIdx}
                          className="flex items-center gap-2 mb-1"
                        >
                          <input
                            type="text"
                            value={w.key}
                            onChange={(e) =>
                              updateStepWith(
                                step.id,
                                wIdx,
                                "key",
                                e.target.value
                              )
                            }
                            placeholder="key"
                            className={cn(miniInputClass, "flex-1 font-mono")}
                          />
                          <input
                            type="text"
                            value={w.value}
                            onChange={(e) =>
                              updateStepWith(
                                step.id,
                                wIdx,
                                "value",
                                e.target.value
                              )
                            }
                            placeholder="value"
                            className={cn(miniInputClass, "flex-1 font-mono")}
                          />
                          <button
                            type="button"
                            onClick={() => removeStepWith(step.id, wIdx)}
                            className="p-1 text-text-secondary hover:text-danger transition-colors"
                          >
                            <Trash2 className="h-3 w-3" />
                          </button>
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={() => addStepWith(step.id)}
                        className="text-[12px] text-accent hover:underline"
                      >
                        + Add parameter
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}

            <Button variant="ghost" size="sm" onClick={addStep}>
              <Plus className="h-3.5 w-3.5" />
              Add Step
            </Button>
          </section>

          {/* ───── Section 6: Test Command ───── */}
          <section>
            <h2 className={sectionTitle}>Test Command</h2>
            <p className={sectionDesc}>
              The command to execute your tests. This will be injected into the workflow.
            </p>
            <textarea
              value={testCommand}
              onChange={(e) => setTestCommand(e.target.value)}
              placeholder="e.g., make test, npm test, go test ./..."
              className={textareaClass}
            />
          </section>

          {/* ───── Section 7: Advanced ───── */}
          <section>
            <button
              type="button"
              onClick={() => setAdvancedOpen(!advancedOpen)}
              className="w-full flex items-center justify-between"
            >
              <h2 className={sectionTitle}>Advanced</h2>
              {advancedOpen ? (
                <ChevronUp className="h-4 w-4 text-text-secondary" />
              ) : (
                <ChevronDown className="h-4 w-4 text-text-secondary" />
              )}
            </button>
            {advancedOpen && (
              <div className="mt-3 space-y-6">
                {/* Matrix Strategy */}
                <div className="rounded-[8px] border border-[var(--border)] bg-bg-primary p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-[14px] font-medium text-text-primary">
                      Matrix Strategy
                    </h3>
                    <ToggleSwitch
                      enabled={matrixEnabled}
                      onChange={setMatrixEnabled}
                    />
                  </div>
                  {matrixEnabled && (
                    <div className="mt-3 space-y-3">
                      <p className="text-[12px] text-text-secondary">
                        Define matrix dimensions. Values are comma-separated.
                      </p>
                      {matrixDimensions.map((dim, idx) => (
                        <div key={idx} className="flex items-center gap-2">
                          <input
                            type="text"
                            value={dim.key}
                            onChange={(e) =>
                              updateMatrixDimension(idx, "key", e.target.value)
                            }
                            placeholder="Dimension key (e.g., node-version)"
                            className={cn(miniInputClass, "w-44 font-mono")}
                          />
                          <input
                            type="text"
                            value={dim.values}
                            onChange={(e) =>
                              updateMatrixDimension(idx, "values", e.target.value)
                            }
                            placeholder="Values (e.g., 18, 20, 22)"
                            className={cn(miniInputClass, "flex-1 font-mono")}
                          />
                          <button
                            type="button"
                            onClick={() => removeMatrixDimension(idx)}
                            className="p-1 text-text-secondary hover:text-danger transition-colors"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={addMatrixDimension}
                        className="text-[12px] text-accent hover:underline"
                      >
                        + Add Dimension
                      </button>
                      <div className="flex items-center gap-2 pt-1">
                        <ToggleSwitch
                          enabled={matrixFailFast}
                          onChange={setMatrixFailFast}
                          size="sm"
                        />
                        <span className="text-[13px] text-text-secondary">
                          Fail fast (cancel remaining jobs on first failure)
                        </span>
                      </div>
                    </div>
                  )}
                </div>

                {/* Concurrency */}
                <div className="rounded-[8px] border border-[var(--border)] bg-bg-primary p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-[14px] font-medium text-text-primary">
                      Concurrency
                    </h3>
                    <ToggleSwitch
                      enabled={concurrencyEnabled}
                      onChange={setConcurrencyEnabled}
                    />
                  </div>
                  {concurrencyEnabled && (
                    <div className="mt-3 space-y-3">
                      <div>
                        <label className="block text-[12px] font-medium text-text-secondary mb-1">
                          Group Name
                        </label>
                        <input
                          type="text"
                          value={concurrencyGroup}
                          onChange={(e) =>
                            setConcurrencyGroup(e.target.value)
                          }
                          className={cn(miniInputClass, "w-full font-mono")}
                        />
                      </div>
                      <div className="flex items-center gap-2">
                        <ToggleSwitch
                          enabled={concurrencyCancelInProgress}
                          onChange={setConcurrencyCancelInProgress}
                          size="sm"
                        />
                        <span className="text-[13px] text-text-secondary">
                          Cancel in-progress runs
                        </span>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </section>

          {/* ───── Error ───── */}
          {error && (
            <div className="rounded-[6px] border border-danger/30 bg-danger/5 px-4 py-3">
              <p className="text-[13px] text-danger">{error}</p>
            </div>
          )}

          {/* ───── Submit ───── */}
          <div className="flex items-center gap-3 pt-2">
            <Button
              onClick={handleSubmit}
              loading={submitting}
              disabled={!canSubmit}
            >
              Create Suite
            </Button>
            <Button
              variant="ghost"
              onClick={() => router.push(`/repositories/${repoId}`)}
            >
              Cancel
            </Button>
          </div>
        </div>
      </div>

      {/* Right panel: YAML Preview */}
      <div className="w-[45%] min-w-0 sticky top-0 h-screen overflow-y-auto bg-[#1a1a2e]">
        <div className="flex items-center justify-between px-5 py-3 border-b border-white/10">
          <h2 className="text-[14px] font-semibold text-gray-300">
            Workflow Preview
          </h2>
          <button
            type="button"
            onClick={handleCopy}
            className="flex items-center gap-1.5 px-2 py-1 rounded-[4px] text-[12px] text-gray-400 hover:text-white hover:bg-white/10 transition-colors"
          >
            {copied ? (
              <>
                <Check className="h-3.5 w-3.5" />
                Copied
              </>
            ) : (
              <>
                <Copy className="h-3.5 w-3.5" />
                Copy
              </>
            )}
          </button>
        </div>
        <pre className="px-5 py-4 text-[12px] leading-[1.6] text-gray-300 font-mono whitespace-pre overflow-x-auto">
          {yamlPreview}
        </pre>
      </div>
    </div>
  );
}

// ─── Toggle Switch Component ──────────────────────────────────────────────────

function ToggleSwitch({
  enabled,
  onChange,
  size = "md",
}: {
  enabled: boolean;
  onChange: (val: boolean) => void;
  size?: "sm" | "md";
}) {
  const dims = size === "sm" ? "h-4 w-7" : "h-5 w-9";
  const dotDims = size === "sm" ? "h-3 w-3" : "h-3.5 w-3.5";
  const translate =
    size === "sm"
      ? enabled
        ? "translate-x-3"
        : "translate-x-0.5"
      : enabled
        ? "translate-x-4"
        : "translate-x-0.5";

  return (
    <button
      type="button"
      role="switch"
      aria-checked={enabled}
      onClick={() => onChange(!enabled)}
      className={cn(
        "relative inline-flex items-center rounded-full transition-colors",
        dims,
        enabled ? "bg-accent" : "bg-bg-tertiary"
      )}
    >
      <span
        className={cn(
          "inline-block rounded-full bg-white transition-transform",
          dotDims,
          translate
        )}
      />
    </button>
  );
}
