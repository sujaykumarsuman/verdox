"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { createTeam } from "@/hooks/use-teams";

interface CreateTeamDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function CreateTeamDialog({ open, onClose, onSuccess }: CreateTeamDialogProps) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) return null;

  const handleNameChange = (val: string) => {
    setName(val);
    // Auto-generate slug from name
    setSlug(
      val
        .toLowerCase()
        .replace(/[^a-z0-9\s-]/g, "")
        .replace(/\s+/g, "-")
        .replace(/-+/g, "-")
        .slice(0, 128)
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !slug.trim()) return;

    setLoading(true);
    setError(null);
    try {
      await createTeam(name.trim(), slug.trim());
      setName("");
      setSlug("");
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create team");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md rounded-[8px] border bg-bg-secondary shadow-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-[18px] font-semibold text-text-primary">
            Create Team
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
            label="Team Name"
            placeholder="My Team"
            value={name}
            onChange={(e) => handleNameChange(e.target.value)}
            autoFocus
          />

          <Input
            label="Slug"
            placeholder="my-team"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            error={error || undefined}
          />

          <p className="text-[13px] text-text-secondary">
            The slug is used in URLs and must be unique. You&apos;ll be added as
            the team admin automatically.
          </p>

          <div className="flex justify-end gap-3 pt-2">
            <Button variant="ghost" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              loading={loading}
              disabled={!name.trim() || !slug.trim()}
            >
              Create Team
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
