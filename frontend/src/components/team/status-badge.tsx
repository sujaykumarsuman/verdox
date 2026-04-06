"use client";

import { Badge } from "@/components/ui/badge";
import type { MemberStatus } from "@/types/team";

const statusConfig: Record<MemberStatus, { variant: "success" | "warning" | "danger"; label: string }> = {
  approved: { variant: "success", label: "Approved" },
  pending: { variant: "warning", label: "Pending" },
  rejected: { variant: "danger", label: "Rejected" },
};

export function StatusBadge({ status }: { status: MemberStatus }) {
  const config = statusConfig[status] || statusConfig.pending;
  return <Badge variant={config.variant}>{config.label}</Badge>;
}
