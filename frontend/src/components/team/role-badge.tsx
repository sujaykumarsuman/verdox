"use client";

import { Badge } from "@/components/ui/badge";
import type { TeamRole } from "@/types/team";

const roleConfig: Record<TeamRole, { variant: "info" | "warning" | "neutral"; label: string }> = {
  admin: { variant: "info", label: "Admin" },
  maintainer: { variant: "warning", label: "Maintainer" },
  viewer: { variant: "neutral", label: "Viewer" },
};

export function RoleBadge({ role }: { role: TeamRole }) {
  const config = roleConfig[role] || roleConfig.viewer;
  return <Badge variant={config.variant}>{config.label}</Badge>;
}
