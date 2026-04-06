"use client";

import { useState } from "react";
import { use } from "react";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { useJoinRequests } from "@/hooks/use-teams";
import { Card, CardBody, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { JoinRequestList } from "@/components/team/join-request-list";

export default function JoinRequestsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id: teamId } = use(params);
  const [statusFilter, setStatusFilter] = useState<string>("pending");
  const { requests, isLoading, error, refetch } = useJoinRequests(teamId, statusFilter);

  const filters = [
    { value: "pending", label: "Pending" },
    { value: "approved", label: "Approved" },
    { value: "rejected", label: "Rejected" },
    { value: "all", label: "All" },
  ];

  return (
    <div className="max-w-4xl">
      <Link
        href={`/teams/${teamId}`}
        className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Team
      </Link>

      <div className="mb-8">
        <h1 className="font-display text-[28px] leading-[36px] tracking-[-0.01em] text-text-primary">
          Join Requests
        </h1>
        <p className="text-[14px] text-text-secondary mt-1">
          Review requests from users who want to join this team.
        </p>
      </div>

      {/* Filters */}
      <div className="flex gap-2 mb-6">
        {filters.map((f) => (
          <button
            key={f.value}
            onClick={() => setStatusFilter(f.value)}
            className={`px-3 py-1.5 text-[13px] rounded-[4px] border transition-colors ${
              statusFilter === f.value
                ? "border-accent bg-accent/10 text-accent font-medium"
                : "border-border text-text-secondary hover:bg-bg-secondary"
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {isLoading ? (
        <Card>
          <CardBody>
            <Skeleton className="h-8 w-full mb-3" />
            <Skeleton className="h-8 w-full mb-3" />
            <Skeleton className="h-8 w-full" />
          </CardBody>
        </Card>
      ) : error ? (
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-8 text-center">
          <p className="text-[14px] text-danger">{error}</p>
        </div>
      ) : (
        <Card>
          <CardHeader>
            <h2 className="text-[16px] font-semibold text-text-primary">
              {statusFilter === "all" ? "All" : statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1)} Requests
              <span className="ml-2 text-[14px] font-normal text-text-secondary">
                ({requests.length})
              </span>
            </h2>
          </CardHeader>
          <CardBody>
            <JoinRequestList teamId={teamId} requests={requests} onRefresh={refetch} />
          </CardBody>
        </Card>
      )}
    </div>
  );
}
