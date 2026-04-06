"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { createTestSuite } from "@/hooks/use-tests";

interface CreateSuiteDialogProps {
  repoId: string;
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function CreateSuiteDialog({
  repoId,
  open,
  onClose,
  onSuccess,
}: CreateSuiteDialogProps) {
  const [name, setName] = useState("");
  const [type, setType] = useState("unit");
  const [configPath, setConfigPath] = useState("");
  const [timeout, setTimeout] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setLoading(true);
    setError(null);
    try {
      await createTestSuite(repoId, {
        name: name.trim(),
        type,
        ...(configPath.trim() ? { config_path: configPath.trim() } : {}),
        ...(timeout ? { timeout_seconds: parseInt(timeout, 10) } : {}),
      });
      setName("");
      setType("unit");
      setConfigPath("");
      setTimeout("");
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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      <div className="relative z-10 w-full max-w-md rounded-[8px] border bg-bg-secondary shadow-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-text-primary">
            Create Test Suite
          </h3>
          <button
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary"
          >
            <X size={20} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Suite Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., Unit Tests"
            required
          />

          <div>
            <label className="block text-sm font-medium text-text-primary mb-1">
              Type
            </label>
            <select
              value={type}
              onChange={(e) => setType(e.target.value)}
              className="w-full h-9 rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/30"
            >
              <option value="unit">Unit</option>
              <option value="integration">Integration</option>
            </select>
          </div>

          <Input
            label="Config Path (optional)"
            value={configPath}
            onChange={(e) => setConfigPath(e.target.value)}
            placeholder="e.g., verdox.yaml"
          />

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

          <div className="flex justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={onClose}
            >
              Cancel
            </Button>
            <Button type="submit" size="sm" loading={loading}>
              Create Suite
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
