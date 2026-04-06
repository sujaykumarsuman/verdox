"use client";

import { useState, useEffect, useMemo } from "react";
import { X, Search, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useUserTeams, useAllTeams, updateUserTeams } from "@/hooks/use-admin";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

interface TeamManageModalProps {
  open: boolean;
  userId: string;
  username: string;
  onClose: () => void;
  onSuccess: () => void;
}

export function TeamManageModal({
  open,
  userId,
  username,
  onClose,
  onSuccess,
}: TeamManageModalProps) {
  const { teams: userTeams, isLoading: userTeamsLoading } = useUserTeams(open ? userId : null);
  const { teams: allTeams, isLoading: allTeamsLoading } = useAllTeams();

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");
  const [saving, setSaving] = useState(false);
  const [initialized, setInitialized] = useState(false);

  // Initialize selected set from current memberships
  useEffect(() => {
    if (!userTeamsLoading && userTeams.length >= 0 && !initialized) {
      setSelected(new Set(userTeams.map((t) => t.team_id)));
      setInitialized(true);
    }
  }, [userTeams, userTeamsLoading, initialized]);

  // Reset when modal reopens
  useEffect(() => {
    if (open) {
      setInitialized(false);
      setSearch("");
    }
  }, [open]);

  const filteredTeams = useMemo(() => {
    if (!search) return allTeams;
    const q = search.toLowerCase();
    return allTeams.filter(
      (t) => t.name.toLowerCase().includes(q) || t.slug.toLowerCase().includes(q)
    );
  }, [allTeams, search]);

  const toggleTeam = (teamId: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(teamId)) {
        next.delete(teamId);
      } else {
        next.add(teamId);
      }
      return next;
    });
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await updateUserTeams(userId, Array.from(selected));
      toast.success(`Team memberships updated for ${username}`);
      onSuccess();
      onClose();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update teams");
    } finally {
      setSaving(false);
    }
  };

  if (!open) return null;

  const isLoading = userTeamsLoading || allTeamsLoading;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-50 w-full max-w-lg mx-4 rounded-[12px] border bg-bg-primary shadow-[var(--shadow-lg)] flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between p-6 pb-0">
          <div>
            <h3 className="text-[16px] font-semibold text-text-primary">
              Manage Teams
            </h3>
            <p className="text-[13px] text-text-secondary mt-0.5">
              Select teams for {username}
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Search */}
        <div className="px-6 pt-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-secondary" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search teams..."
              className="w-full h-9 rounded-[6px] border bg-bg-primary text-text-primary text-[14px] pl-9 pr-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20"
            />
          </div>
        </div>

        {/* Team list */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-1">
          {isLoading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 py-2">
                <Skeleton className="h-4 w-4 rounded" />
                <Skeleton className="h-4 w-40" />
              </div>
            ))
          ) : filteredTeams.length === 0 ? (
            <p className="text-[14px] text-text-secondary text-center py-8">
              {search ? "No teams match your search" : "No teams available"}
            </p>
          ) : (
            filteredTeams.map((team) => (
              <label
                key={team.id}
                className="flex items-center gap-3 py-2 px-2 rounded-[6px] hover:bg-bg-secondary cursor-pointer transition-colors"
              >
                <input
                  type="checkbox"
                  checked={selected.has(team.id)}
                  onChange={() => toggleTeam(team.id)}
                  className="h-4 w-4 rounded border-border text-accent focus:ring-accent/20"
                />
                <div className="flex-1 min-w-0">
                  <span className="text-[14px] text-text-primary font-medium">
                    {team.name}
                  </span>
                  <span className="text-[12px] text-text-secondary ml-2">
                    /{team.slug}
                  </span>
                </div>
                <span className="text-[12px] text-text-tertiary shrink-0">
                  {team.member_count} members
                </span>
              </label>
            ))
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between p-6 pt-4 border-t">
          <p className="text-[13px] text-text-secondary">
            {selected.size} team{selected.size !== 1 ? "s" : ""} selected
          </p>
          <div className="flex gap-3">
            <Button variant="secondary" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={saving}>
              {saving && <Loader2 className="h-4 w-4 animate-spin" />}
              Save
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
