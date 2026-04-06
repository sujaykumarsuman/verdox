import type { ReactNode } from "react";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-primary px-4">
      <div className="w-full max-w-[420px]">
        {/* Logo */}
        <div className="flex justify-center mb-8">
          <span className="font-display text-[30px] tracking-[-0.01em] text-accent">
            Verdox
          </span>
        </div>

        {/* Card */}
        <div className="rounded-xl border bg-bg-secondary p-8 shadow-[var(--shadow-card)]">
          {children}
        </div>
      </div>
    </div>
  );
}
