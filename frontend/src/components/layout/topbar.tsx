"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { LogOut, Settings, User } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { ThemeToggle } from "./theme-toggle";

export function TopBar() {
  const { user, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const router = useRouter();

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
