"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, ChevronLeft, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { useAdminUsers, updateUser } from "@/hooks/use-admin";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { TeamManageModal } from "./team-manage-modal";
import type { AdminUser } from "@/types/admin";
import type { UserRole } from "@/types/user";

export function UserTable() {
  const { user: currentUser } = useAuth();
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [page, setPage] = useState(1);

  // Confirmation modal state
  const [modalOpen, setModalOpen] = useState(false);
  const [modalConfig, setModalConfig] = useState<{
    title: string;
    description: string;
    confirmLabel: string;
    variant: "danger" | "default";
    action: () => Promise<void>;
    inputLabel?: string;
    inputPlaceholder?: string;
    inputRequired?: boolean;
  } | null>(null);
  const [modalLoading, setModalLoading] = useState(false);
  const [modalInputValue, setModalInputValue] = useState("");

  // Team management modal state
  const [teamModalUser, setTeamModalUser] = useState<AdminUser | null>(null);

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const { data, isLoading, refetch } = useAdminUsers(
    debouncedSearch,
    roleFilter,
    statusFilter,
    page
  );


  const handleRoleChange = useCallback(
    (user: AdminUser, newRole: UserRole) => {
      setModalConfig({
        title: "Change User Role",
        description: `Change ${user.username}'s role from "${user.role}" to "${newRole}"?`,
        confirmLabel: "Change Role",
        variant: "default",
        action: async () => {
          await updateUser(user.id, { role: newRole });
          toast.success(`${user.username}'s role changed to ${newRole}`);
          refetch();
        },
      });
      setModalOpen(true);
    },
    [refetch]
  );

  const handleToggleActive = useCallback(
    (user: AdminUser) => {
      if (user.is_active) {
        setModalConfig({
          title: "Deactivate User",
          description: `Deactivate ${user.username}? They will be logged out and unable to sign in.`,
          confirmLabel: "Deactivate",
          variant: "danger",
          action: async () => {
            await updateUser(user.id, { is_active: false });
            toast.success(`${user.username} has been deactivated`);
            refetch();
          },
        });
      } else {
        setModalConfig({
          title: "Reactivate User",
          description: `Reactivate ${user.username}? They will be able to sign in again.`,
          confirmLabel: "Reactivate",
          variant: "default",
          action: async () => {
            await updateUser(user.id, { is_active: true });
            toast.success(`${user.username} has been reactivated`);
            refetch();
          },
        });
      }
      setModalOpen(true);
    },
    [refetch]
  );

  // Store ban target so confirm handler can use current modalInputValue
  const [banTarget, setBanTarget] = useState<AdminUser | null>(null);

  const handleBan = useCallback(
    (user: AdminUser) => {
      setModalInputValue("");
      setBanTarget(user);
      setModalConfig({
        title: "Ban User",
        description: `Ban ${user.username}? They will be immediately logged out and permanently blocked from signing in until unbanned.`,
        confirmLabel: "Ban User",
        variant: "danger",
        inputLabel: "Ban Reason (required)",
        inputPlaceholder: "Explain why this user is being banned...",
        inputRequired: true,
        action: async () => {}, // overridden in handleConfirm
      });
      setModalOpen(true);
    },
    []
  );

  const handleUnban = useCallback(
    (user: AdminUser) => {
      setModalConfig({
        title: "Unban User",
        description: `Unban ${user.username}? Their account will be reactivated and they will be able to sign in again.`,
        confirmLabel: "Unban",
        variant: "default",
        action: async () => {
          await updateUser(user.id, { is_banned: false });
          toast.success(`${user.username} has been unbanned`);
          refetch();
        },
      });
      setModalOpen(true);
    },
    [refetch]
  );

  const handleConfirm = async () => {
    if (!modalConfig) return;
    setModalLoading(true);
    try {
      // Special handling for ban (needs current input value)
      if (banTarget && modalConfig.inputRequired) {
        await updateUser(banTarget.id, { is_banned: true, ban_reason: modalInputValue });
        toast.success(`${banTarget.username} has been banned`);
        refetch();
      } else {
        await modalConfig.action();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed");
    } finally {
      setModalLoading(false);
      setModalOpen(false);
      setModalConfig(null);
      setBanTarget(null);
      setModalInputValue("");
    }
  };

  const handleCancel = () => {
    setModalOpen(false);
    setModalConfig(null);
    setBanTarget(null);
    setModalInputValue("");
  };

  const isRootOrAdmin = currentUser?.role === "root" || currentUser?.role === "admin";

  const roleBadgeVariant = (role: string) => {
    switch (role) {
      case "root":
        return "danger" as const;
      case "admin":
        return "info" as const;
      case "moderator":
        return "warning" as const;
      default:
        return "neutral" as const;
    }
  };

  return (
    <div>
      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-secondary" />
          <Input
            placeholder="Search users..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <select
          value={roleFilter}
          onChange={(e) => {
            setRoleFilter(e.target.value);
            setPage(1);
          }}
          className="h-9 rounded-[6px] border bg-bg-primary text-text-primary text-[14px] px-3"
        >
          <option value="">All Roles</option>
          <option value="root">Root</option>
          <option value="admin">Admin</option>
          <option value="moderator">Moderator</option>
          <option value="user">User</option>
        </select>
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value);
            setPage(1);
          }}
          className="h-9 rounded-[6px] border bg-bg-primary text-text-primary text-[14px] px-3"
        >
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="banned">Banned</option>
        </select>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-[8px] border">
        <table className="w-full text-[14px]">
          <thead>
            <tr className="border-b bg-bg-secondary text-text-secondary">
              <th className="text-left px-4 py-3 font-medium">User</th>
              <th className="text-left px-4 py-3 font-medium">Email</th>
              <th className="text-left px-4 py-3 font-medium">Role</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
              <th className="text-left px-4 py-3 font-medium">Teams</th>
              <th className="text-left px-4 py-3 font-medium">Joined</th>
              <th className="text-right px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {isLoading
              ? Array.from({ length: 8 }).map((_, i) => (
                  <tr key={i} className="border-b">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <Skeleton className="h-8 w-8 rounded-full" />
                        <Skeleton className="h-4 w-24" />
                      </div>
                    </td>
                    <td className="px-4 py-3"><Skeleton className="h-4 w-36" /></td>
                    <td className="px-4 py-3"><Skeleton className="h-6 w-20" /></td>
                    <td className="px-4 py-3"><Skeleton className="h-6 w-16" /></td>
                    <td className="px-4 py-3"><Skeleton className="h-6 w-8" /></td>
                    <td className="px-4 py-3"><Skeleton className="h-4 w-24" /></td>
                    <td className="px-4 py-3"><Skeleton className="h-6 w-16 ml-auto" /></td>
                  </tr>
                ))
              : data?.users.map((user) => {
                  const isSelf = user.id === currentUser?.id;
                  return (
                    <tr key={user.id} className="border-b hover:bg-bg-secondary/50 transition-colors">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <div className="flex items-center justify-center h-8 w-8 rounded-full bg-accent text-white text-[12px] font-medium shrink-0">
                            {user.username[0]?.toUpperCase()}
                          </div>
                          <span className="text-text-primary font-medium truncate">
                            {user.username}
                            {isSelf && (
                              <span className="text-text-tertiary ml-1">(you)</span>
                            )}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-text-secondary truncate">
                        {user.email}
                      </td>
                      <td className="px-4 py-3">
                        {isRootOrAdmin && !isSelf && user.role !== "root" ? (
                          <select
                            value={user.role}
                            onChange={(e) =>
                              handleRoleChange(user, e.target.value as UserRole)
                            }
                            className="h-7 rounded-[4px] border bg-bg-primary text-text-primary text-[13px] px-2"
                          >
                            <option value="user">user</option>
                            <option value="moderator">moderator</option>
                            <option value="admin">admin</option>
                          </select>
                        ) : (
                          <Badge variant={roleBadgeVariant(user.role)}>
                            {user.role}
                          </Badge>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {user.is_banned ? (
                          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[12px] font-medium bg-danger/10 text-danger">
                            <span className="h-1.5 w-1.5 rounded-full bg-danger" />
                            Banned
                          </span>
                        ) : !isSelf ? (
                          <button
                            onClick={() => handleToggleActive(user)}
                            className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[12px] font-medium transition-colors ${
                              user.is_active
                                ? "bg-success/10 text-success hover:bg-success/20"
                                : "bg-warning/10 text-warning hover:bg-warning/20"
                            }`}
                          >
                            <span
                              className={`h-1.5 w-1.5 rounded-full ${
                                user.is_active ? "bg-success" : "bg-warning"
                              }`}
                            />
                            {user.is_active ? "Active" : "Inactive"}
                          </button>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[12px] font-medium bg-success/10 text-success">
                            <span className="h-1.5 w-1.5 rounded-full bg-success" />
                            Active
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => setTeamModalUser(user)}
                          className="inline-flex items-center justify-center min-w-[28px] px-2 py-0.5 rounded-full text-[12px] font-medium bg-[var(--accent-subtle)] text-accent hover:bg-accent/20 transition-colors cursor-pointer"
                        >
                          {user.team_count ?? 0}
                        </button>
                      </td>
                      <td className="px-4 py-3 text-text-secondary whitespace-nowrap">
                        {new Date(user.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {!isSelf && user.role !== "root" && isRootOrAdmin && (
                          user.is_banned ? (
                            <button
                              onClick={() => handleUnban(user)}
                              className="text-[12px] font-medium text-accent hover:text-accent-light transition-colors"
                            >
                              Unban
                            </button>
                          ) : (
                            <button
                              onClick={() => handleBan(user)}
                              className="text-[12px] font-medium text-danger hover:text-[#B33232] transition-colors"
                            >
                              Ban
                            </button>
                          )
                        )}
                      </td>
                    </tr>
                  );
                })}
          </tbody>
        </table>

        {/* Empty state */}
        {!isLoading && data?.users.length === 0 && (
          <div className="text-center py-12 text-text-secondary">
            No users found matching your filters.
          </div>
        )}
      </div>

      {/* Pagination */}
      {data && data.total_pages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <p className="text-[13px] text-text-secondary">
            Page {data.page} of {data.total_pages} ({data.total} users)
          </p>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              <ChevronLeft className="h-4 w-4" />
              Previous
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={page >= data.total_pages}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Team Management Modal */}
      {teamModalUser && (
        <TeamManageModal
          open={!!teamModalUser}
          userId={teamModalUser.id}
          username={teamModalUser.username}
          onClose={() => setTeamModalUser(null)}
          onSuccess={refetch}
        />
      )}

      {/* Confirm Modal */}
      {modalConfig && (
        <ConfirmModal
          open={modalOpen}
          title={modalConfig.title}
          description={modalConfig.description}
          confirmLabel={modalConfig.confirmLabel}
          variant={modalConfig.variant}
          onConfirm={handleConfirm}
          onCancel={handleCancel}
          loading={modalLoading}
          inputLabel={modalConfig.inputLabel}
          inputPlaceholder={modalConfig.inputPlaceholder}
          inputValue={modalInputValue}
          onInputChange={setModalInputValue}
          inputRequired={modalConfig.inputRequired}
        />
      )}
    </div>
  );
}
