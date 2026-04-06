"use client";

import { Users, UserCheck, GitFork, UsersRound, FlaskConical, TrendingUp, CalendarCheck, TestTubes } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { StatsCard } from "./stats-card";
import type { AdminStats } from "@/types/admin";

interface StatsCardsProps {
  stats: AdminStats | null;
  isLoading: boolean;
}

export function StatsCards({ stats, isLoading }: StatsCardsProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="rounded-[8px] border bg-bg-secondary p-4">
            <div className="flex items-center gap-3">
              <Skeleton className="h-10 w-10 rounded-[8px]" />
              <div className="space-y-1.5">
                <Skeleton className="h-6 w-12" />
                <Skeleton className="h-3 w-20" />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (!stats) return null;

  const cards = [
    { icon: Users, label: "Total Users", value: stats.total_users },
    { icon: UserCheck, label: "Active Users", value: stats.active_users },
    { icon: GitFork, label: "Repositories", value: stats.total_repos },
    { icon: UsersRound, label: "Teams", value: stats.total_teams },
    { icon: TestTubes, label: "Test Suites", value: stats.total_suites },
    { icon: FlaskConical, label: "Total Runs", value: stats.total_test_runs },
    { icon: CalendarCheck, label: "Runs Today", value: stats.runs_today },
    { icon: TrendingUp, label: "Pass Rate (7d)", value: `${stats.pass_rate_7d}%` },
  ];

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      {cards.map((card) => (
        <StatsCard key={card.label} {...card} />
      ))}
    </div>
  );
}
