"use client";

import { useState } from "react";
import { Trash2, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { RoleBadge } from "./role-badge";
import { StatusBadge } from "./status-badge";
import { updateMember, removeMember } from "@/hooks/use-teams";
import type { TeamMember, TeamRole } from "@/types/team";

interface MemberListProps {
  teamId: string;
  members: TeamMember[];
  currentUserId: string;
  currentUserRole?: TeamRole;
  onRefresh: () => void;
}

const roles: TeamRole[] = ["admin", "maintainer", "viewer"];

export function MemberList({
  teamId,
  members,
  currentUserId,
  currentUserRole,
  onRefresh,
}: MemberListProps) {
  const [loading, setLoading] = useState<string | null>(null);
  const [roleDropdown, setRoleDropdown] = useState<string | null>(null);
  const isAdmin = currentUserRole === "admin";
  const isAdminOrMaintainer = isAdmin || currentUserRole === "maintainer";

  const handleApprove = async (userId: string) => {
    setLoading(userId);
    try {
      await updateMember(teamId, userId, { status: "approved" });
      onRefresh();
    } finally {
      setLoading(null);
    }
  };

  const handleReject = async (userId: string) => {
    setLoading(userId);
    try {
      await updateMember(teamId, userId, { status: "rejected" });
      onRefresh();
    } finally {
      setLoading(null);
    }
  };

  const handleRoleChange = async (userId: string, role: TeamRole) => {
    setLoading(userId);
    setRoleDropdown(null);
    try {
      await updateMember(teamId, userId, { role });
      onRefresh();
    } finally {
      setLoading(null);
    }
  };

  const handleRemove = async (userId: string) => {
    setLoading(userId);
    try {
      await removeMember(teamId, userId);
      onRefresh();
    } finally {
      setLoading(null);
    }
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-[14px]">
        <thead>
          <tr className="border-b border-border text-left text-text-secondary">
            <th className="pb-2 font-medium">User</th>
            <th className="pb-2 font-medium">Role</th>
            <th className="pb-2 font-medium">Status</th>
            <th className="pb-2 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {members.map((m) => (
            <tr key={m.id} className="border-b border-border/50">
              <td className="py-3">
                <div>
                  <span className="font-medium text-text-primary">{m.username}</span>
                  <span className="text-text-secondary ml-2 text-[13px]">{m.email}</span>
                </div>
              </td>
              <td className="py-3">
                {isAdmin && m.user_id !== currentUserId && m.status === "approved" ? (
                  <div className="relative inline-block">
                    <button
                      onClick={() => setRoleDropdown(roleDropdown === m.user_id ? null : m.user_id)}
                      className="inline-flex items-center gap-1 text-[13px] px-2 py-1 rounded-[4px] border border-border hover:bg-bg-secondary"
                    >
                      {m.role}
                      <ChevronDown className="h-3 w-3" />
                    </button>
                    {roleDropdown === m.user_id && (
                      <div className="absolute z-10 mt-1 bg-bg-primary border border-border rounded-[6px] shadow-lg py-1 min-w-[120px]">
                        {roles.map((r) => (
                          <button
                            key={r}
                            onClick={() => handleRoleChange(m.user_id, r)}
                            className={`block w-full text-left px-3 py-1.5 text-[13px] hover:bg-bg-secondary ${
                              r === m.role ? "font-semibold text-accent" : "text-text-primary"
                            }`}
                          >
                            {r}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                ) : (
                  <RoleBadge role={m.role} />
                )}
              </td>
              <td className="py-3">
                <StatusBadge status={m.status} />
              </td>
              <td className="py-3 text-right">
                <div className="flex items-center justify-end gap-2">
                  {m.status === "pending" && isAdminOrMaintainer && (
                    <>
                      <Button
                        size="sm"
                        onClick={() => handleApprove(m.user_id)}
                        loading={loading === m.user_id}
                      >
                        Approve
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleReject(m.user_id)}
                        loading={loading === m.user_id}
                      >
                        Reject
                      </Button>
                    </>
                  )}
                  {m.status === "approved" &&
                    (isAdminOrMaintainer || m.user_id === currentUserId) && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleRemove(m.user_id)}
                        loading={loading === m.user_id}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    )}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {members.length === 0 && (
        <p className="text-center text-text-secondary py-8 text-[14px]">No members yet.</p>
      )}
    </div>
  );
}
