"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Bell, LogOut, Settings, Shield, User } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useUnreadCount } from "@/hooks/use-notifications";
import { useSSE } from "@/hooks/use-sse";
import { api } from "@/lib/api";
import { ThemeToggle } from "./theme-toggle";
import { NotificationList } from "@/components/notification/notification-list";
import type { Notification, NotificationListResponse } from "@/types/notification";

export function TopBar() {
  const { user, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const [bellOpen, setBellOpen] = useState(false);
  const [previewNotifs, setPreviewNotifs] = useState<Notification[]>([]);
  const [previewVersion, setPreviewVersion] = useState(0);
  const menuRef = useRef<HTMLDivElement>(null);
  const bellRef = useRef<HTMLDivElement>(null);
  const router = useRouter();
  const { count: unreadCount, refetch: refetchUnread } = useUnreadCount();

  // SSE: refresh unread count + dropdown content on new notifications
  useSSE({
    enabled: !!user,
    onNotificationNew: () => {
      refetchUnread();
      setPreviewVersion((v) => v + 1);
    },
    onBanReviewRequested: () => {
      refetchUnread();
      setPreviewVersion((v) => v + 1);
    },
  });

  // Fetch preview notifications when bell dropdown opens or SSE fires
  useEffect(() => {
    if (!bellOpen && previewVersion === 0) return;
    const fetchPreview = async () => {
      try {
        const data = await api<NotificationListResponse>("/v1/notifications?page=1&per_page=5");
        setPreviewNotifs(data.notifications || []);
      } catch {
        // Ignore
      }
    };
    fetchPreview();
  }, [bellOpen, previewVersion]);

  // Close menus on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
      if (bellRef.current && !bellRef.current.contains(e.target as Node)) {
        setBellOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const handleLogout = async () => {
    await logout();
    router.push("/login");
  };

  return (
    <header className="h-14 bg-bg-primary border-b flex items-center justify-between px-6 shrink-0">
      {/* Breadcrumb placeholder */}
      <div className="text-[14px] text-text-secondary" />

      {/* Right side */}
      <div className="flex items-center gap-2">
        <ThemeToggle />

        {/* Notification bell */}
        <div className="relative" ref={bellRef}>
          <button
            onClick={() => setBellOpen(!bellOpen)}
            className="relative inline-flex items-center justify-center h-9 w-9 rounded-[6px] text-text-secondary hover:bg-bg-tertiary transition-colors duration-150"
            aria-label="Notifications"
          >
            <Bell className="h-5 w-5" />
            {unreadCount > 0 && (
              <span className="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full bg-danger text-white text-[10px] font-bold">
                {unreadCount > 99 ? "99+" : unreadCount}
              </span>
            )}
          </button>

          {bellOpen && (
            <div className="absolute right-0 top-11 w-80 rounded-[8px] border bg-bg-primary shadow-[var(--shadow-lg)] z-50 overflow-hidden">
              <div className="flex items-center justify-between px-4 py-2.5 border-b">
                <span className="text-[14px] font-medium text-text-primary">Notifications</span>
                {unreadCount > 0 && (
                  <span className="text-[12px] text-accent font-medium">{unreadCount} unread</span>
                )}
              </div>
              <div className="max-h-[320px] overflow-y-auto">
                <NotificationList
                  notifications={previewNotifs}
                  onSelect={(n) => {
                    setBellOpen(false);
                    router.push(`/notifications`);
                  }}
                  compact
                />
              </div>
              <Link
                href="/notifications"
                onClick={() => setBellOpen(false)}
                className="block text-center px-4 py-2.5 border-t text-[13px] text-accent hover:bg-bg-tertiary transition-colors"
              >
                View all notifications
              </Link>
            </div>
          )}
        </div>

        {/* User menu */}
        <div className="relative" ref={menuRef}>
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="inline-flex items-center justify-center h-9 w-9 rounded-full bg-accent text-white text-[14px] font-medium hover:bg-accent-light transition-colors duration-200"
          >
            {user?.username?.[0]?.toUpperCase() || <User className="h-4 w-4" />}
          </button>

          {menuOpen && (
            <div className="absolute right-0 top-11 w-48 rounded-[8px] border bg-bg-primary shadow-[var(--shadow-lg)] py-1 z-50">
              <div className="px-3 py-2 border-b">
                <p className="text-[14px] font-medium text-text-primary truncate">
                  {user?.username}
                </p>
                <p className="text-[12px] text-text-secondary truncate">
                  {user?.email}
                </p>
              </div>
              {(user?.role === "root" || user?.role === "admin" || user?.role === "moderator") && (
                <Link
                  href="/admin"
                  onClick={() => setMenuOpen(false)}
                  className="flex items-center gap-2 px-3 py-2 text-[14px] text-text-secondary hover:bg-bg-tertiary"
                >
                  <Shield className="h-4 w-4" />
                  {user?.role === "moderator" ? "Mod Panel" : "Admin Panel"}
                </Link>
              )}
              <Link
                href="/settings"
                onClick={() => setMenuOpen(false)}
                className="flex items-center gap-2 px-3 py-2 text-[14px] text-text-secondary hover:bg-bg-tertiary"
              >
                <Settings className="h-4 w-4" />
                Settings
              </Link>
              <button
                onClick={handleLogout}
                className="flex items-center gap-2 px-3 py-2 text-[14px] text-danger hover:bg-bg-tertiary w-full"
              >
                <LogOut className="h-4 w-4" />
                Sign Out
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
