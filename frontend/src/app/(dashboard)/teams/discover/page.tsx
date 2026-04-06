"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowLeft, Users, GitBranch, Loader2, Search, CheckCircle2 } from "lucide-react";
import { useDiscoverableTeams, submitJoinRequest } from "@/hooks/use-teams";
import { Button } from "@/components/ui/button";
import { Card, CardBody } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

export default function TeamDiscoveryPage() {
  const { teams, isLoading, error, refetch } = useDiscoverableTeams();
  const [loadingId, setLoadingId] = useState<string | null>(null);
  const [successId, setSuccessId] = useState<string | null>(null);

  const handleJoinRequest = async (teamId: string) => {
    setLoadingId(teamId);
    try {
      await submitJoinRequest(teamId);
      setSuccessId(teamId);
      refetch();
    } catch {
      // error handled by API wrapper
    } finally {
      setLoadingId(null);
    }
  };

  return (
    <div className="max-w-4xl">
      <Link
        href="/teams"
        className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        My Teams
      </Link>

      <div className="mb-8">
        <h1 className="font-display text-[28px] leading-[36px] tracking-[-0.01em] text-text-primary">
          Discover Teams
        </h1>
        <p className="text-[14px] text-text-secondary mt-1">
          Browse teams and request to join.
        </p>
      </div>

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i}>
              <CardBody>
                <Skeleton className="h-6 w-40 mb-2" />
                <Skeleton className="h-4 w-24 mb-4" />
                <Skeleton className="h-8 w-28" />
              </CardBody>
            </Card>
          ))}
        </div>
      ) : error ? (
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-8 text-center">
          <p className="text-[14px] text-danger">{error}</p>
        </div>
      ) : teams.length === 0 ? (
        <div className="rounded-[8px] border border-border bg-bg-secondary/30 p-12 text-center">
          <Search className="h-10 w-10 text-text-secondary mx-auto mb-3" />
          <p className="text-[16px] font-medium text-text-primary">No teams available</p>
          <p className="text-[14px] text-text-secondary mt-1">
            There are no discoverable teams at the moment.
          </p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {teams.map((t) => (
            <Card key={t.id}>
              <CardBody>
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="text-[16px] font-semibold text-text-primary">{t.name}</h3>
                    <p className="text-[13px] text-text-secondary">{t.slug}</p>
                  </div>
                  {t.user_status === "approved" && (
                    <Badge variant="success">Member</Badge>
                  )}
                  {t.user_status === "pending" && (
                    <Badge variant="warning">Pending</Badge>
                  )}
                </div>

                <div className="flex items-center gap-4 mt-3 text-[13px] text-text-secondary">
                  <span className="flex items-center gap-1">
                    <Users className="h-3.5 w-3.5" />
                    {t.member_count} members
                  </span>
                  <span className="flex items-center gap-1">
                    <GitBranch className="h-3.5 w-3.5" />
                    {t.repo_count} repos
                  </span>
                </div>

                <div className="mt-4">
                  {t.user_status === "approved" ? (
                    <Link href={`/teams/${t.id}`}>
                      <Button variant="secondary" size="sm">
                        View Team
                      </Button>
                    </Link>
                  ) : t.user_status === "pending" || successId === t.id ? (
                    <div className="flex items-center gap-1.5 text-[13px] text-text-secondary">
                      <CheckCircle2 className="h-4 w-4 text-yellow-600" />
                      Request pending
                    </div>
                  ) : (
                    <Button
                      size="sm"
                      onClick={() => handleJoinRequest(t.id)}
                      loading={loadingId === t.id}
                    >
                      Request to Join
                    </Button>
                  )}
                </div>
              </CardBody>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
