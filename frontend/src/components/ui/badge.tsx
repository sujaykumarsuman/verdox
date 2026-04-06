"use client";

import { cn } from "@/lib/utils";

type BadgeVariant = "success" | "danger" | "warning" | "info" | "neutral";

interface BadgeProps {
  variant?: BadgeVariant;
  children: React.ReactNode;
  className?: string;
}

const variantStyles: Record<BadgeVariant, string> = {
  success: "bg-[#E6F4EC] text-[var(--success)]",
  danger: "bg-[#FDEAEA] text-[var(--danger)]",
  warning: "bg-[#FEF3D9] text-[var(--warning)]",
  info: "bg-[var(--accent-subtle)] text-accent",
  neutral: "bg-bg-tertiary text-text-secondary",
};

export function Badge({ variant = "neutral", children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center px-2 py-0.5 rounded-full text-[12px] font-medium",
        variantStyles[variant],
        className
      )}
    >
      {children}
    </span>
  );
}
