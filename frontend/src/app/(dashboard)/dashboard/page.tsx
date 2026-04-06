"use client";

import { useAuth } from "@/hooks/use-auth";

export default function DashboardPage() {
  const { user } = useAuth();

  return (
    <div className="max-w-3xl">
      <h1 className="font-display text-[30px] leading-[38px] tracking-[-0.01em] text-text-primary mb-2">
        Welcome to Verdox
      </h1>
      <p className="text-[16px] text-text-secondary mb-8">
        {user ? `Hello, ${user.username}. ` : ""}
        Your dashboard will show repository cards and test runs here.
      </p>
      <div className="rounded-[8px] border border-dashed bg-bg-secondary p-12 text-center">
        <p className="text-[14px] text-text-secondary">
          Repository management and test suite features coming in Phase 2.
        </p>
      </div>
    </div>
  );
}
