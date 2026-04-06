"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { LayoutDashboard, Users, Settings, Shield, PanelLeftClose, PanelLeftOpen, Search } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/hooks/use-auth";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/teams", label: "Teams", icon: Users },
  { href: "/teams/discover", label: "Discover", icon: Search },
  { href: "/settings", label: "Settings", icon: Settings },
];

const adminItem = { href: "/admin", label: "Admin", icon: Shield };

export function Sidebar() {
  const [collapsed, setCollapsed] = useState(false);
  const pathname = usePathname();
  const { user } = useAuth();

  useEffect(() => {
    const saved = localStorage.getItem("verdox-sidebar-collapsed");
    if (saved === "true") setCollapsed(true);
  }, []);

  const toggle = () => {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem("verdox-sidebar-collapsed", String(next));
  };

  const isAdmin = user?.role === "root" || user?.role === "moderator";
  const items = isAdmin ? [...navItems, adminItem] : navItems;

  return (
    <aside
      className={cn(
        "h-screen bg-bg-secondary border-r flex flex-col transition-all duration-300",
        collapsed ? "w-16" : "w-[260px]"
      )}
    >
      {/* Logo */}
      <div className="h-14 flex items-center px-4 border-b">
        <Link href="/dashboard" className="overflow-hidden">
          <span className="font-display text-[20px] text-accent whitespace-nowrap">
            {collapsed ? "V" : "Verdox"}
          </span>
        </Link>
      </div>

      {/* Nav */}
      <nav className="flex-1 py-4 px-3 flex flex-col gap-1">
        {items.map((item) => {
          // Match exact path or prefix, but prefer the longest matching nav item
          // so /teams/discover highlights "Discover" not "Teams"
          const matchesThis = pathname === item.href || pathname.startsWith(item.href + "/");
          const moreSpecificMatch = matchesThis && items.some(
            (other) => other.href !== item.href && other.href.startsWith(item.href + "/") &&
              (pathname === other.href || pathname.startsWith(other.href + "/"))
          );
          const isActive = matchesThis && !moreSpecificMatch;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-[6px] text-[14px] font-medium transition-colors duration-200",
                isActive
                  ? "bg-accent-subtle text-accent"
                  : "text-text-secondary hover:bg-bg-tertiary"
              )}
            >
              <item.icon className="h-5 w-5 shrink-0" />
              {!collapsed && <span>{item.label}</span>}
            </Link>
          );
        })}
      </nav>

      {/* Collapse toggle */}
      <div className="p-3 border-t">
        <button
          onClick={toggle}
          className="flex items-center gap-3 px-3 py-2 rounded-[6px] text-[14px] text-text-secondary hover:bg-bg-tertiary w-full transition-colors duration-200"
        >
          {collapsed ? (
            <PanelLeftOpen className="h-5 w-5 shrink-0" />
          ) : (
            <>
              <PanelLeftClose className="h-5 w-5 shrink-0" />
              <span>Collapse</span>
            </>
          )}
        </button>
      </div>
    </aside>
  );
}
