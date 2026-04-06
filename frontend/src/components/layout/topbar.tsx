"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Bell, LogOut, Settings, Shield, User } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { api } from "@/lib/api";
import { ThemeToggle } from "./theme-toggle";

export function TopBar() {
  const { user, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const router = useRouter();
  const [pendingReviews, setPendingReviews] = useState(0);

  const isAdminUser = user?.role === "root" || user?.role === "admin";

  // Poll pending ban reviews for admin users
  const fetchPendingReviews = useCallback(async () => {
    if (!isAdminUser) return;
    try {
      const data = await api<{ count: number }>("/v1/admin/ban-reviews");
      setPendingReviews(data.count);
    } catch {
      // Ignore errors
    }
  }, [isAdminUser]);

  useEffect(() => {
    fetchPendingReviews();
    if (!isAdminUser) return;
    const interval = setInterval(fetchPendingReviews, 30000); // poll every 30s
    return () => clearInterval(interval);
  }, [fetchPendingReviews, isAdminUser]);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
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
        <button
          onClick={() => { if (isAdminUser) router.push("/admin"); }}
          className="relative inline-flex items-center justify-center h-9 w-9 rounded-[6px] text-text-secondary hover:bg-bg-tertiary transition-colors duration-150"
          aria-label="Notifications"
        >
          <Bell className="h-5 w-5" />
          {pendingReviews > 0 && (
            <span className="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full bg-danger text-white text-[10px] font-bold">
              {pendingReviews}
            </span>
          )}
        </button>

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
