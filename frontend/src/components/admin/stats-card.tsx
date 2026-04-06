import type { LucideIcon } from "lucide-react";
import { Card } from "@/components/ui/card";

interface StatsCardProps {
  icon: LucideIcon;
  label: string;
  value: string | number;
  subValue?: string;
}

export function StatsCard({ icon: Icon, label, value, subValue }: StatsCardProps) {
  return (
    <Card className="p-4">
      <div className="flex items-center gap-3">
        <div className="flex items-center justify-center h-10 w-10 rounded-[8px] bg-accent-subtle">
          <Icon className="h-5 w-5 text-accent" />
        </div>
        <div className="min-w-0">
          <p className="text-[22px] font-semibold text-text-primary leading-tight">
            {value}
          </p>
          <p className="text-[12px] text-text-secondary truncate">{label}</p>
          {subValue && (
            <p className="text-[11px] text-text-tertiary">{subValue}</p>
          )}
        </div>
      </div>
    </Card>
  );
}
