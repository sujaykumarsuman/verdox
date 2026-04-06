"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { inviteMember } from "@/hooks/use-teams";
import type { TeamRole } from "@/types/team";

interface InviteMemberDialogProps {
  teamId: string;
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const roles: { value: TeamRole; label: string }[] = [
  { value: "viewer", label: "Viewer" },
  { value: "maintainer", label: "Maintainer" },
  { value: "admin", label: "Admin" },
];

export function InviteMemberDialog({
  teamId,
  open,
  onClose,
  onSuccess,
}: InviteMemberDialogProps) {
  const [userId, setUserId] = useState("");
  const [role, setRole] = useState<TeamRole>("viewer");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (!open) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!userId.trim()) return;

    setLoading(true);
    setError(null);
    try {
      await inviteMember(teamId, userId.trim(), role);
      setUserId("");
      setRole("viewer");
      toast.success("Invitation sent");
      onSuccess();
      onClose();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to invite member";
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-bg-primary rounded-[8px] border border-border shadow-xl w-full max-w-md p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-[18px] font-semibold text-text-primary">Invite Member</h2>
          <button onClick={onClose} className="text-text-secondary hover:text-text-primary">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="User ID"
            placeholder="Enter user UUID"
            value={userId}
            onChange={(e) => {
              setUserId(e.target.value);
              setError(null);
            }}
            error={error || undefined}
          />

          <div>
            <label className="block text-[14px] font-medium text-text-primary mb-1.5">Role</label>
            <div className="flex gap-2">
              {roles.map((r) => (
                <button
                  key={r.value}
                  type="button"
                  onClick={() => setRole(r.value)}
                  className={`px-3 py-1.5 text-[13px] rounded-[4px] border transition-colors ${
                    role === r.value
                      ? "border-accent bg-accent/10 text-accent font-medium"
                      : "border-border text-text-secondary hover:bg-bg-secondary"
                  }`}
                >
                  {r.label}
                </button>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <Button variant="ghost" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" loading={loading} disabled={!userId.trim()}>
              Send Invite
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
