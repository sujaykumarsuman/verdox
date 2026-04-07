"use client";

import { useEffect } from "react";
import { Bell, Mail, ShieldAlert, TestTube, Users, UserPlus, CheckCircle, XCircle } from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import type { Notification } from "@/types/notification";

const typeIcons: Record<string, typeof Bell> = {
  system: Bell,
  admin_message: Mail,
  ban_review: ShieldAlert,
  test_complete: TestTube,
  team_invite: Users,
  team_join_request: UserPlus,
};

interface NotificationDetailProps {
  notification: Notification;
  onMarkRead: (id: string) => void;
  onActionComplete?: () => void;
}

export function NotificationDetail({ notification, onMarkRead, onActionComplete }: NotificationDetailProps) {
  const Icon = typeIcons[notification.type] || Bell;

  // Mark as read on mount
  useEffect(() => {
    if (!notification.is_read) {
      onMarkRead(notification.id);
    }
  }, [notification.id, notification.is_read, onMarkRead]);

  const handleBanReviewAction = async (action: "approved" | "denied") => {
    const reviewId = notification.action_payload?.review_id as string;
    if (!reviewId) return;

    try {
      await api(`/v1/admin/ban-reviews/${reviewId}`, {
        method: "PUT",
        body: JSON.stringify({ status: action }),
      });
      toast.success(action === "approved" ? "Ban review approved — user unbanned" : "Ban review denied");
      onActionComplete?.();
    } catch {
      toast.error("Failed to process ban review");
    }
  };

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-start gap-3 mb-4">
        <div className="shrink-0 flex items-center justify-center h-10 w-10 rounded-full bg-accent-subtle text-accent">
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <h2 className="text-[18px] font-semibold text-text-primary">{notification.subject}</h2>
          <div className="flex items-center gap-2 text-[13px] text-text-secondary mt-1">
            {notification.sender_username && <span>From {notification.sender_username}</span>}
            <span>{new Date(notification.created_at).toLocaleString()}</span>
          </div>
        </div>
      </div>

      {/* Body */}
      <div className="text-[14px] text-text-primary leading-relaxed whitespace-pre-wrap border-t pt-4">
        {notification.body || "No content."}
      </div>

      {/* Action Buttons */}
      {notification.action_type === "ban_review_decision" && notification.action_payload && (
        <div className="flex items-center gap-3 mt-6 pt-4 border-t">
          <button
            onClick={() => handleBanReviewAction("approved")}
            className="flex items-center gap-2 px-4 py-2 rounded-[6px] bg-success text-white text-[14px] font-medium hover:opacity-90 transition-opacity"
          >
            <CheckCircle className="h-4 w-4" />
            Approve &amp; Unban
          </button>
          <button
            onClick={() => handleBanReviewAction("denied")}
            className="flex items-center gap-2 px-4 py-2 rounded-[6px] bg-danger text-white text-[14px] font-medium hover:opacity-90 transition-opacity"
          >
            <XCircle className="h-4 w-4" />
            Deny
          </button>
          {notification.action_payload.username ? (
            <span className="text-[13px] text-text-secondary ml-2">
              User: {String(notification.action_payload.username)}
            </span>
          ) : null}
        </div>
      )}
    </div>
  );
}
