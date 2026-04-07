"use client";

import { Search } from "lucide-react";

type StatusFilter = "all" | "failed" | "passed" | "skipped";

interface HierarchyFiltersProps {
  statusFilter: StatusFilter;
  onStatusFilterChange: (filter: StatusFilter) => void;
  search: string;
  onSearchChange: (search: string) => void;
}

const filterOptions: { value: StatusFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "failed", label: "Failed" },
  { value: "passed", label: "Passed" },
  { value: "skipped", label: "Skipped" },
];

export function HierarchyFilters({
  statusFilter,
  onStatusFilterChange,
  search,
  onSearchChange,
}: HierarchyFiltersProps) {
  return (
    <div className="flex items-center gap-3 mb-4">
      <div className="flex rounded-[6px] border border-[var(--border)] overflow-hidden">
        {filterOptions.map((opt) => (
          <button
            key={opt.value}
            onClick={() => onStatusFilterChange(opt.value)}
            className={`px-3 py-1.5 text-[13px] transition-colors ${
              statusFilter === opt.value
                ? "bg-[var(--accent)] text-white"
                : "bg-bg-secondary text-text-secondary hover:text-text-primary hover:bg-bg-tertiary"
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>
      <div className="relative flex-1 max-w-xs">
        <Search
          size={14}
          className="absolute left-3 top-1/2 -translate-y-1/2 text-text-secondary"
        />
        <input
          type="text"
          placeholder="Search jobs or cases..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className="w-full pl-9 pr-3 py-1.5 text-[13px] rounded-[6px] border border-[var(--border)] bg-bg-primary text-text-primary placeholder:text-text-secondary/50 focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
        />
      </div>
    </div>
  );
}
