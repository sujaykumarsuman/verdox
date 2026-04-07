"use client";

import { useState, useEffect } from "react";
import { X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { addRepository } from "@/hooks/use-repos";
import { api } from "@/lib/api";

interface Team {
  id: string;
  name: string;
  slug: string;
}

interface AddRepoDialogProps {
  teamId?: string; // optional — if not provided, user selects
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function AddRepoDialog({ teamId: initialTeamId, open, onClose, onSuccess }: AddRepoDialogProps) {
  const [url, setUrl] = useState("");
  const [selectedTeamId, setSelectedTeamId] = useState(initialTeamId || "");
  const [teams, setTeams] = useState<Team[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch user's teams for selection
  useEffect(() => {
    if (!open) return;
    const fetchTeams = async () => {
      try {
        const data = await api<Team[]>("/v1/teams");
        setTeams(data || []);
        // Auto-select if only one team or if initialTeamId is provided
        if (initialTeamId) {
          setSelectedTeamId(initialTeamId);
        } else if (data && data.length === 1) {
          setSelectedTeamId(data[0].id);
        }
      } catch {
        // Ignore
      }
    };
    fetchTeams();
  }, [open, initialTeamId]);

  if (!open) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!url.trim() || !selectedTeamId) return;

    setLoading(true);
    setError(null);
    try {
      await addRepository(url.trim(), selectedTeamId);
      setUrl("");
      toast.success("Repository added — forking in background");
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
          {/* Team selector */}
          <div>
            <label className="block text-sm font-medium text-text-primary mb-1">
              Team
            </label>
            {teams.length === 0 ? (
              <p className="text-[13px] text-text-secondary">
                You need to create or join a team first.
              </p>
            ) : (
              <select
                value={selectedTeamId}
                onChange={(e) => setSelectedTeamId(e.target.value)}
                className="w-full h-9 rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/30"
                required
              >
                <option value="">Select a team</option>
                {teams.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            )}
          </div>

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
            The selected team&apos;s GitHub PAT will be used to access it.
          </p>

          <div className="flex justify-end gap-3 pt-2">
            <Button variant="ghost" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" loading={loading} disabled={!url.trim() || !selectedTeamId}>
              Add Repository
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
