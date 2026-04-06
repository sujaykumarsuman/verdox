"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { addRepository } from "@/hooks/use-repos";

interface AddRepoDialogProps {
  teamId: string;
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function AddRepoDialog({ teamId, open, onClose, onSuccess }: AddRepoDialogProps) {
  const [url, setUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!url.trim()) return;

    setLoading(true);
    setError(null);
    try {
      await addRepository(url.trim(), teamId);
      setUrl("");
      toast.success("Repository added — cloning in background");
      onSuccess();
      onClose();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to add repository";
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
      />

      {/* Dialog */}
      <div className="relative z-10 w-full max-w-md rounded-[8px] border bg-bg-secondary shadow-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-[18px] font-semibold text-text-primary">
            Add Repository
          </h2>
          <button
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <Input
            label="GitHub Repository URL"
            placeholder="https://github.com/owner/repo"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            error={error || undefined}
            autoFocus
          />

          <p className="text-[13px] text-text-secondary">
            Enter the full GitHub URL of the repository you want to add.
            Your team&apos;s GitHub PAT will be used to access it.
          </p>

          <div className="flex justify-end gap-3 pt-2">
            <Button variant="ghost" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" loading={loading} disabled={!url.trim()}>
              Add Repository
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
