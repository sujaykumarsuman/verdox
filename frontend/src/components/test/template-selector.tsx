"use client";

import { workflowTemplates, type WorkflowTemplate } from "@/lib/workflow-templates";

interface TemplateSelectorProps {
  current: string;
  onChange: (template: WorkflowTemplate) => void;
}

export function TemplateSelector({ current, onChange }: TemplateSelectorProps) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-[13px] text-text-secondary">Template:</span>
      <select
        value={current}
        onChange={(e) => {
          const tmpl = workflowTemplates.find((t) => t.name === e.target.value);
          if (tmpl) onChange(tmpl);
        }}
        className="px-3 py-1.5 text-[13px] rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
      >
        {workflowTemplates.map((t) => (
          <option key={t.name} value={t.name}>
            {t.label}
          </option>
        ))}
      </select>
    </div>
  );
}
