"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { assignRepo } from "@/hooks/use-teams";

interface AssignRepoDialogProps {
  teamId: string;
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function AssignRepoDialog({
  teamId,
  open,
  onClose,
  onSuccess,
}: AssignRepoDialogProps) {
  const [repoId, setRepoId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (!open) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!repoId.trim()) return;

    setLoading(true);
    setError(null);
    try {
      await assignRepo(teamId, repoId.trim());
      setRepoId("");
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to assign repository");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-bg-primary rounded-[8px] border border-border shadow-xl w-full max-w-md p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-[18px] font-semibold text-text-primary">Assign Repository</h2>
          <button onClick={onClose} className="text-text-secondary hover:text-text-primary">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Repository ID"
            placeholder="Enter repository UUID"
            value={repoId}
            onChange={(e) => {
              setRepoId(e.target.value);
              setError(null);
            }}
            error={error || undefined}
          />

          <p className="text-[13px] text-text-secondary">
            Enter the UUID of a repository to assign it to this team.
          </p>

          <div className="flex justify-end gap-3 pt-2">
            <Button variant="ghost" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" loading={loading} disabled={!repoId.trim()}>
              Assign
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
