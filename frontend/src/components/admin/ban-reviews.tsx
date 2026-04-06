"use client";

import { useState, useCallback } from "react";
import { ShieldAlert, CheckCircle2, XCircle } from "lucide-react";
import { toast } from "sonner";
import { useBanReviews, reviewBan } from "@/hooks/use-admin";
import { Card, CardBody } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmModal } from "@/components/ui/confirm-modal";

export function BanReviews() {
  const { data, isLoading, refetch } = useBanReviews();

  const [modalOpen, setModalOpen] = useState(false);
  const [modalConfig, setModalConfig] = useState<{
    title: string;
    description: string;
    confirmLabel: string;
    variant: "danger" | "default";
    action: () => Promise<void>;
  } | null>(null);
  const [modalLoading, setModalLoading] = useState(false);

  const handleApprove = useCallback(
    (reviewId: string, username: string) => {
      setModalConfig({
        title: "Approve Ban Review",
        description: `Approve the review for ${username}? This will unban the user and allow them to sign in again.`,
        confirmLabel: "Approve & Unban",
        variant: "default",
        action: async () => {
          await reviewBan(reviewId, "approved");
          toast.success(`${username} has been unbanned`);
          refetch();
        },
      });
      setModalOpen(true);
    },
    [refetch]
  );

  const handleDeny = useCallback(
    (reviewId: string, username: string) => {
      setModalConfig({
        title: "Deny Ban Review",
        description: `Deny the review for ${username}? Their ban will remain in effect.`,
        confirmLabel: "Deny",
        variant: "danger",
        action: async () => {
          await reviewBan(reviewId, "denied");
          toast.success("Ban review denied");
          refetch();
        },
      });
      setModalOpen(true);
    },
    [refetch]
  );

  const handleConfirm = async () => {
    if (!modalConfig) return;
    setModalLoading(true);
    try {
      await modalConfig.action();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed");
    } finally {
      setModalLoading(false);
      setModalOpen(false);
      setModalConfig(null);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-24 w-full rounded-[8px]" />
      </div>
    );
  }

  if (!data || data.count === 0) return null;

  return (
    <div>
      <div className="flex items-center gap-2 mb-4">
        <h2 className="text-[18px] font-semibold text-text-primary">
          Ban Reviews
        </h2>
        <Badge variant="warning">{data.count} pending</Badge>
      </div>

      <div className="space-y-3">
        {data.reviews.map((review) => (
          <Card key={review.id}>
            <CardBody>
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-2">
                    <ShieldAlert className="h-4 w-4 text-danger shrink-0" />
                    <span className="text-[14px] font-semibold text-text-primary">
                      {review.username}
                    </span>
                    <span className="text-[13px] text-text-secondary">
                      {review.email}
                    </span>
                  </div>

                  <div className="space-y-2 text-[13px]">
                    <div>
                      <span className="text-text-secondary font-medium">Ban reason: </span>
                      <span className="text-text-primary">{review.ban_reason}</span>
                    </div>
                    <div className="p-2 rounded bg-bg-secondary border">
                      <span className="text-text-secondary font-medium">User&apos;s clarification: </span>
                      <span className="text-text-primary">{review.clarification}</span>
                    </div>
                    <p className="text-[12px] text-text-tertiary">
                      Submitted {new Date(review.created_at).toLocaleDateString()} at{" "}
                      {new Date(review.created_at).toLocaleTimeString()}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-2 shrink-0">
                  <Button
                    size="sm"
                    onClick={() => handleApprove(review.id, review.username)}
                  >
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    Unban
                  </Button>
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={() => handleDeny(review.id, review.username)}
                  >
                    <XCircle className="h-3.5 w-3.5" />
                    Deny
                  </Button>
                </div>
              </div>
            </CardBody>
          </Card>
        ))}
      </div>

      {modalConfig && (
        <ConfirmModal
          open={modalOpen}
          title={modalConfig.title}
          description={modalConfig.description}
          confirmLabel={modalConfig.confirmLabel}
          variant={modalConfig.variant}
          onConfirm={handleConfirm}
          onCancel={() => { setModalOpen(false); setModalConfig(null); }}
          loading={modalLoading}
        />
      )}
    </div>
  );
}
