"use client";

import { use, useState, useCallback, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { WorkflowEditor } from "@/components/test/workflow-editor";
import { TemplateSelector } from "@/components/test/template-selector";
import {
  getDefaultTemplate,
  type WorkflowTemplate,
} from "@/lib/workflow-templates";

const TYPE_OPTIONS = [
  "unit",
  "integration",
  "e2e",
  "lint",
  "smoke",
  "build",
  "race",
  "compatibility",
  "load",
];

export default function CreateTestSuitePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id: repoId } = use(params);
  const router = useRouter();

  const defaultTemplate = getDefaultTemplate();
  const [name, setName] = useState("");
  const [type, setType] = useState("unit");
  const [timeout, setTimeout] = useState(300);
  const [templateName, setTemplateName] = useState(defaultTemplate.name);
  const [workflowYaml, setWorkflowYaml] = useState(defaultTemplate.yaml);
  const [submitting, setSubmitting] = useState(false);

  // Pre-fill from import (sessionStorage bridge from ImportSuiteDialog)
  useEffect(() => {
    const stored = sessionStorage.getItem("verdox_import_suite");
    if (!stored) return;
    sessionStorage.removeItem("verdox_import_suite");
    try {
      const data = JSON.parse(stored) as {
        name?: string;
        type?: string;
        timeout_seconds?: number;
        workflow_yaml?: string;
      };
      if (data.name) setName(data.name);
      if (data.type && TYPE_OPTIONS.includes(data.type)) setType(data.type);
      if (data.timeout_seconds) setTimeout(data.timeout_seconds);
      if (data.workflow_yaml) {
        setWorkflowYaml(data.workflow_yaml);
        setTemplateName("imported");
      }
    } catch {
      // Ignore corrupt data
    }
  }, []);

  const handleTemplateChange = useCallback(
    (template: WorkflowTemplate) => {
      if (workflowYaml !== defaultTemplate.yaml && workflowYaml.trim() !== "") {
        if (
          !window.confirm(
            "Switching templates will replace your current editor content. Continue?"
          )
        ) {
          return;
        }
      }
      setTemplateName(template.name);
      setWorkflowYaml(template.yaml);
    },
    [workflowYaml, defaultTemplate.yaml]
  );

  const handleSubmit = async () => {
    if (!name.trim()) {
      toast.error("Suite name is required");
      return;
    }
    if (!workflowYaml.includes("workflow_dispatch")) {
      toast.error(
        "Workflow must include workflow_dispatch trigger with verdox_run_id input"
      );
      return;
    }

    setSubmitting(true);
    try {
      await api(`/v1/repositories/${repoId}/suites`, {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          type,
          execution_mode: "fork_gha",
          timeout_seconds: timeout,
          workflow_yaml: workflowYaml,
        }),
      });
      toast.success("Test suite created");
      router.push(`/repositories/${repoId}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to create test suite"
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-5xl">
      {/* Breadcrumb */}
      <Link
        href={`/repositories/${repoId}`}
        className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Repository
      </Link>

      <h1 className="font-display text-[28px] leading-[36px] tracking-[-0.01em] text-text-primary mb-6">
        Create Test Suite
      </h1>

      {/* Metadata Fields */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div>
          <label className="block text-[13px] font-medium text-text-secondary mb-1.5">
            Suite Name
          </label>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Unit Tests"
          />
        </div>
        <div>
          <label className="block text-[13px] font-medium text-text-secondary mb-1.5">
            Type
          </label>
          <select
            value={type}
            onChange={(e) => setType(e.target.value)}
            className="w-full px-3 py-2 text-[14px] rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
          >
            {TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-[13px] font-medium text-text-secondary mb-1.5">
            Timeout (seconds)
          </label>
          <Input
            type="number"
            value={timeout}
            onChange={(e) => setTimeout(Number(e.target.value))}
            min={30}
            max={3600}
          />
        </div>
      </div>

      {/* Template Selector + Editor */}
      <div className="flex items-center justify-between mb-3">
        <label className="text-[13px] font-medium text-text-secondary">
          GitHub Actions Workflow
        </label>
        <TemplateSelector current={templateName} onChange={handleTemplateChange} />
      </div>

      <WorkflowEditor value={workflowYaml} onChange={setWorkflowYaml} />

      {/* Submit */}
      <div className="flex items-center justify-end gap-3 mt-6">
        <Link href={`/repositories/${repoId}`}>
          <Button variant="secondary">Cancel</Button>
        </Link>
        <Button onClick={handleSubmit} loading={submitting}>
          Create Suite
        </Button>
      </div>
    </div>
  );
}
