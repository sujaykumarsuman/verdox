"use client";

import { forwardRef, type ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md" | "lg";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
}

const variantStyles: Record<ButtonVariant, string> = {
  primary:
    "bg-accent text-white hover:bg-accent-light active:bg-accent-dark disabled:bg-disabled-bg disabled:text-disabled",
  secondary:
    "bg-transparent text-accent border border-accent hover:bg-accent-subtle active:bg-accent-subtle disabled:border-disabled disabled:text-disabled",
  ghost:
    "bg-transparent text-text-primary hover:bg-bg-secondary active:bg-bg-tertiary disabled:text-disabled",
  danger:
    "bg-danger text-white hover:bg-[#B33232] active:bg-[#9E2B2B] disabled:bg-disabled-bg disabled:text-disabled",
};

const sizeStyles: Record<ButtonSize, string> = {
  sm: "h-7 px-3 text-[14px]",
  md: "h-9 px-4 text-[14px]",
  lg: "h-11 px-6 text-[16px]",
};

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "primary", size = "md", loading, disabled, children, ...props }, ref) => {
    return (
      <button
        ref={ref}
        className={cn(
          "inline-flex items-center justify-center gap-2 rounded-[6px] font-medium transition-colors duration-200 cursor-pointer",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
          variantStyles[variant],
          sizeStyles[size],
          (disabled || loading) && "pointer-events-none",
          className
        )}
        disabled={disabled || loading}
        {...props}
      >
        {loading && <Loader2 className="h-4 w-4 animate-spin" />}
        {children}
      </button>
    );
  }
);

Button.displayName = "Button";

export { Button };
export type { ButtonProps };
