"use client";

import { Settings } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";

export default function SettingsPage() {
  const { user } = useAuth();

  return (
    <div className="max-w-3xl">
      <h1 className="font-display text-[30px] leading-[38px] tracking-[-0.01em] text-text-primary mb-2">
        Settings
      </h1>
      <p className="text-[14px] text-text-secondary mb-8">
        {user ? `Signed in as ${user.username} (${user.email})` : "Manage your account settings."}
      </p>
      <div className="rounded-[8px] border border-dashed bg-bg-secondary p-12 text-center">
        <Settings className="h-12 w-12 text-text-secondary mx-auto mb-4" />
        <h3 className="text-[16px] font-medium text-text-primary mb-1">
          Coming soon
        </h3>
        <p className="text-[14px] text-text-secondary">
          Profile and account settings will be available in a future update.
        </p>
      </div>
    </div>
  );
}
