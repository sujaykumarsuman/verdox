"use client";

import { Shield } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useAdminStats } from "@/hooks/use-admin";
import { StatsCards } from "@/components/admin/stats-cards";
import { BanReviews } from "@/components/admin/ban-reviews";
import { UserTable } from "@/components/admin/user-table";

export default function AdminPage() {
  const { user } = useAuth();
  const { stats, isLoading: statsLoading } = useAdminStats();

  const isAdmin = user?.role === "root" || user?.role === "admin" || user?.role === "moderator";

  if (!isAdmin) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center">
        <Shield className="h-12 w-12 text-text-secondary mb-4" />
        <h2 className="text-[18px] font-semibold text-text-primary mb-1">
          Access Denied
        </h2>
        <p className="text-[14px] text-text-secondary">
          You don&apos;t have permission to view the admin panel.
        </p>
      </div>
    );
  }

  const isMod = user?.role === "moderator";

  return (
    <div>
      <div className="mb-6">
        <h1 className="font-display text-[30px] leading-[38px] tracking-[-0.01em] text-text-primary mb-1">
          {isMod ? "Mod Panel" : "Admin Panel"}
        </h1>
        <p className="text-[14px] text-text-secondary">
          {isMod ? "View users and system health." : "Manage users and monitor system health."}
        </p>
      </div>

      <div className="mb-8">
        <StatsCards stats={stats} isLoading={statsLoading} />
      </div>

      {/* Ban reviews — visible to root and admin only */}
      {!isMod && (
        <div className="mb-8">
          <BanReviews />
        </div>
      )}

      <div>
        <h2 className="text-[18px] font-semibold text-text-primary mb-4">
          Users
        </h2>
        <UserTable />
      </div>
    </div>
  );
}
