"use client";

import { useEffect, useRef } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "./button";

interface ConfirmModalProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  variant?: "danger" | "default";
  onConfirm: () => void;
  onCancel: () => void;
  loading?: boolean;
  inputLabel?: string;
  inputPlaceholder?: string;
  inputValue?: string;
  onInputChange?: (value: string) => void;
  inputRequired?: boolean;
}

export function ConfirmModal({
  open,
  title,
  description,
  confirmLabel = "Confirm",
  variant = "default",
  onConfirm,
  onCancel,
  loading = false,
  inputLabel,
  inputPlaceholder,
  inputValue,
  onInputChange,
  inputRequired = false,
}: ConfirmModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, onCancel]);

  if (!open) return null;

  const confirmDisabled = loading || (inputRequired && !inputValue?.trim());

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="fixed inset-0 bg-black/50"
        onClick={onCancel}
      />
      <div
        ref={dialogRef}
        className="relative z-50 w-full max-w-md mx-4 rounded-[12px] border bg-bg-primary shadow-[var(--shadow-lg)] p-6"
      >
        <h3 className="text-[16px] font-semibold text-text-primary mb-2">
          {title}
        </h3>
        <p className="text-[14px] text-text-secondary mb-4">{description}</p>

        {inputLabel && onInputChange && (
          <div className="mb-4">
            <label className="block text-[14px] font-medium text-text-primary mb-1.5">
              {inputLabel}
            </label>
            <textarea
              value={inputValue || ""}
              onChange={(e) => onInputChange(e.target.value)}
              placeholder={inputPlaceholder}
              rows={3}
              className="w-full rounded-[4px] border bg-bg-primary px-3 py-2.5 text-[14px] text-text-primary placeholder:text-text-secondary focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20 resize-none"
            />
          </div>
        )}

        <div className="flex justify-end gap-3">
          <Button
            variant="secondary"
            onClick={onCancel}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            variant={variant === "danger" ? "danger" : "primary"}
            onClick={onConfirm}
            disabled={confirmDisabled}
          >
            {loading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
