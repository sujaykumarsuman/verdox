"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, Mail, Send, Users, Filter, UserCheck } from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface UserEntry {
  id: string;
  username: string;
  email: string;
  role: string;
  is_active: boolean;
}

export default function AdminMailPage() {
  const { user } = useAuth();
  const router = useRouter();
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [recipientType, setRecipientType] = useState<"all" | "filtered" | "selected">("all");
  const [roleFilter, setRoleFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("active");
  const [search, setSearch] = useState("");
  const [availableUsers, setAvailableUsers] = useState<UserEntry[]>([]);
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [recipientCount, setRecipientCount] = useState<number | null>(null);

  const isAdmin = user?.role === "root" || user?.role === "admin";

  // Fetch users for selection mode
  const fetchUsers = useCallback(async () => {
    if (recipientType !== "selected") return;
    try {
      const data = await api<{ users: UserEntry[]; total: number }>(
        `/v1/admin/users?per_page=100&search=${encodeURIComponent(search)}`
      );
      setAvailableUsers(data.users);
    } catch {
      // Ignore
    }
  }, [recipientType, search]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  // Estimate recipient count
  useEffect(() => {
    if (recipientType === "selected") {
      setRecipientCount(selectedUserIds.length);
    } else {
      // Fetch count estimate
      const fetchCount = async () => {
        try {
          const params = new URLSearchParams();
          if (recipientType === "filtered") {
            if (roleFilter) params.set("role", roleFilter);
            if (statusFilter) params.set("status", statusFilter);
          }
          const data = await api<{ total: number }>(`/v1/admin/users?per_page=1&${params}`);
          setRecipientCount(data.total);
        } catch {
          setRecipientCount(null);
        }
      };
      fetchCount();
    }
  }, [recipientType, roleFilter, statusFilter, selectedUserIds]);

  const handleSend = async () => {
    if (!subject.trim() || !body.trim()) return;

    setLoading(true);
    try {
      const recipients: Record<string, unknown> = { type: recipientType };
      if (recipientType === "filtered") {
        if (roleFilter) recipients.role = roleFilter;
        if (statusFilter) recipients.status = statusFilter;
      }
      if (recipientType === "selected") {
        recipients.user_ids = selectedUserIds;
      }

      const result = await api<{ sent: number; failed: number; recipients: number }>(
        "/v1/admin/mail",
        {
          method: "POST",
          body: JSON.stringify({ subject: subject.trim(), body: body.trim(), recipients }),
        }
      );

      toast.success(`Mail sent to ${result.sent} recipient(s)`);
      setSubject("");
      setBody("");
      setSelectedUserIds([]);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to send mail");
    } finally {
      setLoading(false);
    }
  };

  const toggleUser = (id: string) => {
    setSelectedUserIds((prev) =>
      prev.includes(id) ? prev.filter((uid) => uid !== id) : [...prev, id]
    );
  };

  if (!isAdmin) {
    return (
      <div className="flex items-center justify-center h-full text-text-secondary">
        Access denied
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto py-8 px-6">
      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        <button
          onClick={() => router.push("/admin")}
          className="text-text-secondary hover:text-text-primary"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <Mail className="h-5 w-5 text-accent" />
        <h1 className="text-[20px] font-display font-semibold text-text-primary">Send Mail</h1>
      </div>

      <div className="space-y-6">
        {/* Recipients */}
        <div>
          <label className="block text-sm font-medium text-text-primary mb-2">Recipients</label>
          <div className="grid grid-cols-3 gap-2">
            <button
              onClick={() => setRecipientType("all")}
              className={`flex items-center gap-2 px-3 py-2 rounded-[6px] border text-[13px] transition-colors ${
                recipientType === "all"
                  ? "border-accent bg-accent-subtle text-accent"
                  : "border-[var(--border)] text-text-secondary hover:border-accent/50"
              }`}
            >
              <Users className="h-4 w-4" />
              All Users
            </button>
            <button
              onClick={() => setRecipientType("filtered")}
              className={`flex items-center gap-2 px-3 py-2 rounded-[6px] border text-[13px] transition-colors ${
                recipientType === "filtered"
                  ? "border-accent bg-accent-subtle text-accent"
                  : "border-[var(--border)] text-text-secondary hover:border-accent/50"
              }`}
            >
              <Filter className="h-4 w-4" />
              Filtered
            </button>
            <button
              onClick={() => setRecipientType("selected")}
              className={`flex items-center gap-2 px-3 py-2 rounded-[6px] border text-[13px] transition-colors ${
                recipientType === "selected"
                  ? "border-accent bg-accent-subtle text-accent"
                  : "border-[var(--border)] text-text-secondary hover:border-accent/50"
              }`}
            >
              <UserCheck className="h-4 w-4" />
              Select
            </button>
          </div>

          {/* Filters */}
          {recipientType === "filtered" && (
            <div className="grid grid-cols-2 gap-3 mt-3">
              <div>
                <label className="block text-[12px] text-text-secondary mb-1">Role</label>
                <select
                  value={roleFilter}
                  onChange={(e) => setRoleFilter(e.target.value)}
                  className="w-full h-9 rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 text-sm text-text-primary"
                >
                  <option value="">All roles</option>
                  <option value="user">User</option>
                  <option value="moderator">Moderator</option>
                  <option value="admin">Admin</option>
                  <option value="root">Root</option>
                </select>
              </div>
              <div>
                <label className="block text-[12px] text-text-secondary mb-1">Status</label>
                <select
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                  className="w-full h-9 rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 text-sm text-text-primary"
                >
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                  <option value="banned">Banned</option>
                </select>
              </div>
            </div>
          )}

          {/* User selection */}
          {recipientType === "selected" && (
            <div className="mt-3">
              <Input
                placeholder="Search users..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
              <div className="mt-2 max-h-40 overflow-y-auto border rounded-[6px]">
                {availableUsers.map((u) => (
                  <label
                    key={u.id}
                    className="flex items-center gap-3 px-3 py-2 hover:bg-bg-tertiary cursor-pointer text-[13px]"
                  >
                    <input
                      type="checkbox"
                      checked={selectedUserIds.includes(u.id)}
                      onChange={() => toggleUser(u.id)}
                      className="rounded"
                    />
                    <span className="text-text-primary">{u.username}</span>
                    <span className="text-text-secondary">{u.email}</span>
                  </label>
                ))}
                {availableUsers.length === 0 && (
                  <p className="px-3 py-2 text-[13px] text-text-secondary">No users found</p>
                )}
              </div>
            </div>
          )}

          {recipientCount !== null && (
            <p className="mt-2 text-[12px] text-text-secondary">
              {recipientCount} recipient{recipientCount !== 1 ? "s" : ""} will receive this message
            </p>
          )}
        </div>

        {/* Subject */}
        <Input
          label="Subject"
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          placeholder="Enter subject..."
          required
        />

        {/* Body */}
        <div>
          <label className="block text-sm font-medium text-text-primary mb-1">Message</label>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            className="w-full min-h-[200px] rounded-[6px] border border-[var(--border)] bg-bg-primary px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/30"
            placeholder="Write your message..."
            required
          />
        </div>

        {/* Send */}
        <div className="flex items-center justify-between pt-2">
          <p className="text-[12px] text-text-secondary">
            Messages will appear as push notifications in each recipient&apos;s bell and notifications page in real-time.
          </p>
          <Button
            onClick={handleSend}
            loading={loading}
            disabled={!subject.trim() || !body.trim() || (recipientType === "selected" && selectedUserIds.length === 0)}
          >
            <Send className="h-4 w-4 mr-2" />
            Send
          </Button>
        </div>
      </div>
    </div>
  );
}
