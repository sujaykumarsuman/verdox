"use client";

import { Bell, Mail, ShieldAlert, TestTube, Users, UserPlus } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Notification } from "@/types/notification";

const typeIcons: Record<string, typeof Bell> = {
  system: Bell,
  admin_message: Mail,
  ban_review: ShieldAlert,
  test_complete: TestTube,
  team_invite: Users,
  team_join_request: UserPlus,
};

interface NotificationListProps {
  notifications: Notification[];
  selectedId?: string;
  onSelect: (notification: Notification) => void;
  compact?: boolean;
}

export function NotificationList({ notifications, selectedId, onSelect, compact }: NotificationListProps) {
  if (notifications.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-text-secondary text-[14px]">
        <Bell className="h-8 w-8 mb-2 opacity-40" />
        <p>No notifications</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      {notifications.map((n) => {
        const Icon = typeIcons[n.type] || Bell;
        const isSelected = selectedId === n.id;
        return (
          <button
            key={n.id}
            onClick={() => onSelect(n)}
            className={cn(
              "flex items-start gap-3 px-4 py-3 text-left border-b transition-colors duration-150 hover:bg-bg-tertiary",
              !n.is_read && "bg-accent-subtle/30",
              isSelected && "bg-bg-tertiary",
              compact && "px-3 py-2"
            )}
          >
            <div className={cn(
              "shrink-0 mt-0.5 flex items-center justify-center h-8 w-8 rounded-full",
              !n.is_read ? "bg-accent text-white" : "bg-bg-tertiary text-text-secondary"
            )}>
              <Icon className="h-4 w-4" />
            </div>
            <div className="min-w-0 flex-1">
              <p className={cn(
                "text-[14px] truncate",
                !n.is_read ? "font-medium text-text-primary" : "text-text-secondary"
              )}>
                {n.subject}
              </p>
              {!compact && n.sender_username && (
                <p className="text-[12px] text-text-secondary mt-0.5">
                  From {n.sender_username}
                </p>
              )}
              <p className="text-[12px] text-text-secondary mt-0.5">
                {formatRelativeTime(n.created_at)}
              </p>
            </div>
            {!n.is_read && (
              <span className="shrink-0 mt-2 h-2 w-2 rounded-full bg-accent" />
            )}
          </button>
        );
      })}
    </div>
  );
}

function formatRelativeTime(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);

  if (diffMin < 1) return "Just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHrs = Math.floor(diffMin / 60);
  if (diffHrs < 24) return `${diffHrs}h ago`;
  const diffDays = Math.floor(diffHrs / 24);
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}
