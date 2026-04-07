"use client";

import { GitBranch } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Branch } from "@/types/repository";

interface BranchSelectorProps {
  branches: Branch[];
  selected: string;
  onSelect: (name: string) => void;
  isLoading?: boolean;
  defaultBranch?: string;
}

export function BranchSelector({
  branches,
  selected,
  onSelect,
  isLoading,
  defaultBranch,
}: BranchSelectorProps) {
  if (isLoading) {
    return (
      <div className="h-9 w-48 bg-bg-tertiary rounded-[4px] animate-pulse" />
    );
  }

  return (
    <div className="relative inline-flex items-center">
      <GitBranch className="absolute left-3 h-4 w-4 text-text-secondary pointer-events-none" />
      <select
        value={selected}
        onChange={(e) => onSelect(e.target.value)}
        className={cn(
          "appearance-none rounded-[4px] border bg-bg-primary pl-9 pr-8 py-2 text-[14px] text-text-primary",
          "transition-colors duration-200 cursor-pointer max-w-[220px] truncate",
          "focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20"
        )}
      >
        {branches.map((branch) => (
          <option key={branch.name} value={branch.name}>
            {branch.name}
            {branch.name === defaultBranch ? " (default)" : ""}
          </option>
        ))}
      </select>
      <svg
        className="absolute right-2.5 h-4 w-4 text-text-secondary pointer-events-none"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
      </svg>
    </div>
  );
}
