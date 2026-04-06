"use client";

import { forwardRef, type InputHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  icon?: ReactNode;
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, icon, id, ...props }, ref) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, "-");

    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label
            htmlFor={inputId}
            className="text-[14px] font-medium text-text-primary"
          >
            {label}
          </label>
        )}
        <div className="relative">
          {icon && (
            <div className="absolute left-3 top-1/2 -translate-y-1/2 text-text-secondary">
              {icon}
            </div>
          )}
          <input
            ref={ref}
            id={inputId}
            className={cn(
              "w-full rounded-[4px] border bg-bg-primary px-3 py-2.5 text-[16px] text-text-primary placeholder:text-text-secondary",
              "transition-colors duration-200",
              "focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20",
              "disabled:bg-disabled-bg disabled:text-disabled disabled:cursor-not-allowed",
              icon && "pl-10",
              error && "border-danger focus:border-danger focus:ring-danger/20",
              className
            )}
            {...props}
          />
        </div>
        {error && (
          <p className="text-[12px] text-danger">{error}</p>
        )}
      </div>
    );
  }
);

Input.displayName = "Input";

export { Input };
export type { InputProps };
