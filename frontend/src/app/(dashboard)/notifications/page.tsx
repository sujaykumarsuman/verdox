"use client";

import { useState, useCallback } from "react";
import { Bell, CheckCheck } from "lucide-react";
import { toast } from "sonner";
import { useNotifications, useMarkRead, useMarkAllRead, useUnreadCount } from "@/hooks/use-notifications";
import { useSSE } from "@/hooks/use-sse";
import { useAuth } from "@/hooks/use-auth";
import { NotificationList } from "@/components/notification/notification-list";
import { NotificationDetail } from "@/components/notification/notification-detail";
import type { Notification } from "@/types/notification";

export default function NotificationsPage() {
  const { user } = useAuth();
  const [page, setPage] = useState(1);
  const { data, isLoading, refetch } = useNotifications(page);
  const markRead = useMarkRead();
  const markAllRead = useMarkAllRead();
  const { refetch: refetchUnread } = useUnreadCount();
  const [selected, setSelected] = useState<Notification | null>(null);

  // SSE: auto-refetch the list when new notifications arrive
  useSSE({
    enabled: !!user,
    onNotificationNew: () => {
      refetch();
      refetchUnread();
    },
    onBanReviewRequested: () => {
      refetch();
      refetchUnread();
    },
  });

  const handleSelect = useCallback((notification: Notification) => {
    setSelected(notification);
  }, []);

  const handleMarkRead = useCallback(async (id: string) => {
    try {
      await markRead(id);
      refetch();
      refetchUnread();
    } catch {
      // Ignore
    }
  }, [markRead, refetch, refetchUnread]);

  const handleMarkAllRead = async () => {
    try {
      await markAllRead();
      toast.success("All notifications marked as read");
      refetch();
      refetchUnread();
    } catch {
      toast.error("Failed to mark all as read");
    }
  };

  const totalPages = data ? Math.ceil(data.total / data.per_page) : 1;

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b">
        <div className="flex items-center gap-2">
          <Bell className="h-5 w-5 text-accent" />
          <h1 className="text-[20px] font-display font-semibold text-text-primary">Notifications</h1>
          {data && data.total > 0 && (
            <span className="text-[13px] text-text-secondary ml-1">({data.total})</span>
          )}
        </div>
        <button
          onClick={handleMarkAllRead}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-[6px] text-[13px] text-text-secondary hover:bg-bg-tertiary transition-colors"
        >
          <CheckCheck className="h-4 w-4" />
          Mark all read
        </button>
      </div>

      {/* Content */}
      <div className="flex flex-1 overflow-hidden">
        {/* List */}
        <div className="w-[400px] border-r overflow-y-auto shrink-0">
          {isLoading ? (
            <div className="p-4 space-y-3">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="h-16 bg-bg-tertiary rounded-[6px] animate-pulse" />
              ))}
            </div>
          ) : data && data.notifications && data.notifications.length > 0 ? (
            <>
              <NotificationList
                notifications={data.notifications}
                selectedId={selected?.id}
                onSelect={handleSelect}
              />
              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-center gap-2 p-3 border-t">
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page <= 1}
                    className="px-3 py-1 rounded-[6px] text-[13px] text-text-secondary hover:bg-bg-tertiary disabled:opacity-40 transition-colors"
                  >
                    Previous
                  </button>
                  <span className="text-[13px] text-text-secondary">
                    {page} / {totalPages}
                  </span>
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page >= totalPages}
                    className="px-3 py-1 rounded-[6px] text-[13px] text-text-secondary hover:bg-bg-tertiary disabled:opacity-40 transition-colors"
                  >
                    Next
                  </button>
                </div>
              )}
            </>
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-text-secondary text-[14px]">
              <Bell className="h-8 w-8 mb-2 opacity-40" />
              <p>No notifications</p>
            </div>
          )}
        </div>

        {/* Detail */}
        <div className="flex-1 overflow-y-auto">
          {selected ? (
            <NotificationDetail
              notification={selected}
              onMarkRead={handleMarkRead}
              onActionComplete={() => {
                refetch();
                refetchUnread();
                setSelected(null);
              }}
            />
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-text-secondary text-[14px]">
              <Bell className="h-10 w-10 mb-3 opacity-30" />
              <p>Select a notification to view details</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
