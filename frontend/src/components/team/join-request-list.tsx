"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "./status-badge";
import { reviewJoinRequest } from "@/hooks/use-teams";
import type { TeamJoinRequest, TeamRole } from "@/types/team";

interface JoinRequestListProps {
  teamId: string;
  requests: TeamJoinRequest[];
  onRefresh: () => void;
}

const roles: { value: TeamRole; label: string }[] = [
  { value: "viewer", label: "Viewer" },
  { value: "maintainer", label: "Maintainer" },
  { value: "admin", label: "Admin" },
];

export function JoinRequestList({ teamId, requests, onRefresh }: JoinRequestListProps) {
  const [loading, setLoading] = useState<string | null>(null);
  const [selectedRole, setSelectedRole] = useState<Record<string, TeamRole>>({});

  const handleApprove = async (requestId: string) => {
    setLoading(requestId);
    try {
      const role = selectedRole[requestId] || "viewer";
      await reviewJoinRequest(teamId, requestId, "approved", role);
      onRefresh();
    } finally {
      setLoading(null);
    }
  };

  const handleReject = async (requestId: string) => {
    setLoading(requestId);
    try {
      await reviewJoinRequest(teamId, requestId, "rejected");
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
            <th className="pb-2 font-medium">Message</th>
            <th className="pb-2 font-medium">Status</th>
            <th className="pb-2 font-medium">Date</th>
            <th className="pb-2 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {requests.map((r) => (
            <tr key={r.id} className="border-b border-border/50">
              <td className="py-3">
                <span className="font-medium text-text-primary">{r.user.username}</span>
                <span className="text-text-secondary ml-2 text-[13px]">{r.user.email}</span>
              </td>
              <td className="py-3 text-text-secondary max-w-[200px] truncate">
                {r.message || "—"}
              </td>
              <td className="py-3">
                <StatusBadge status={r.status} />
              </td>
              <td className="py-3 text-text-secondary text-[13px]">
                {new Date(r.created_at).toLocaleDateString()}
              </td>
              <td className="py-3 text-right">
                {r.status === "pending" && (
                  <div className="flex items-center justify-end gap-2">
                    <select
                      value={selectedRole[r.id] || "viewer"}
                      onChange={(e) =>
                        setSelectedRole((prev) => ({ ...prev, [r.id]: e.target.value as TeamRole }))
                      }
                      className="text-[13px] px-2 py-1 rounded-[4px] border border-border bg-bg-primary text-text-primary"
                    >
                      {roles.map((role) => (
                        <option key={role.value} value={role.value}>
                          {role.label}
                        </option>
                      ))}
                    </select>
                    <Button
                      size="sm"
                      onClick={() => handleApprove(r.id)}
                      loading={loading === r.id}
                    >
                      Approve
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleReject(r.id)}
                      loading={loading === r.id}
                    >
                      Reject
                    </Button>
                  </div>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {requests.length === 0 && (
        <p className="text-center text-text-secondary py-8 text-[14px]">No join requests.</p>
      )}
    </div>
  );
}
