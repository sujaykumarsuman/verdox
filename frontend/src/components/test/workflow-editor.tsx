"use client";

import { useCallback } from "react";

interface WorkflowEditorProps {
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
}

/**
 * YAML workflow editor.
 * TODO: Replace textarea with @monaco-editor/react for syntax highlighting,
 * YAML validation, and Verdox-specific completions. Install:
 *   npm install @monaco-editor/react
 */
export function WorkflowEditor({ value, onChange, readOnly }: WorkflowEditorProps) {
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      onChange(e.target.value);
    },
    [onChange]
  );

  return (
    <div className="relative rounded-[8px] border border-[var(--border)] overflow-hidden bg-[#1e1e2e]">
      <div className="flex items-center justify-between px-3 py-2 bg-bg-tertiary border-b border-[var(--border)]">
        <span className="text-[12px] font-medium text-text-secondary uppercase tracking-wider">
          workflow.yml
        </span>
        <div className="flex items-center gap-2">
          {hasRequiredStructure(value) ? (
            <span className="flex items-center gap-1 text-[11px] text-[var(--success)]">
              <span className="w-1.5 h-1.5 rounded-full bg-[var(--success)]" />
              Valid Verdox structure
            </span>
          ) : (
            <span className="flex items-center gap-1 text-[11px] text-[var(--warning)]">
              <span className="w-1.5 h-1.5 rounded-full bg-[var(--warning)]" />
              Missing required structure
            </span>
          )}
        </div>
      </div>
      <textarea
        value={value}
        onChange={handleChange}
        readOnly={readOnly}
        spellCheck={false}
        className="w-full min-h-[500px] p-4 font-mono text-[13px] leading-relaxed text-[#cdd6f4] bg-[#1e1e2e] resize-y focus:outline-none"
        style={{ tabSize: 2 }}
      />
    </div>
  );
}

function hasRequiredStructure(yaml: string): boolean {
  return (
    yaml.includes("workflow_dispatch") &&
    yaml.includes("verdox_run_id") &&
    (yaml.includes("verdox-results") || yaml.includes("callback_url"))
  );
}
