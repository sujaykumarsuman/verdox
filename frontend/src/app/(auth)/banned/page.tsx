"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { ShieldAlert, CheckCircle2, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost/api";

interface BanInfo {
  reason: string;
  hasPendingReview: boolean;
  reviewsRemaining: number;
  login: string;
  password: string;
  email?: string;
  username?: string;
  ban_reason?: string; // from SSE forceBan path
}

export default function BannedPage() {
  const [banInfo, setBanInfo] = useState<BanInfo | null>(null);
  const [clarification, setClarification] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Clear stale auth cookies by calling logout (raw fetch, ignore errors)
    fetch(`${API_BASE}/v1/auth/logout`, {
      method: "POST",
      credentials: "include",
    }).catch(() => {});

    // Read ban info from sessionStorage
    try {
      const raw = sessionStorage.getItem("verdox_ban_info");
      if (raw) {
        const parsed = JSON.parse(raw);
        // Normalize: SSE path stores ban_reason, login path stores reason
        if (parsed.ban_reason && !parsed.reason) {
          parsed.reason = parsed.ban_reason;
        }
        setBanInfo(parsed);
      }
    } catch {
      // No ban info available
    }
  }, []);

  const handleBackToSignIn = () => {
    // Clear all auth state and ban info before navigating
    sessionStorage.removeItem("verdox_ban_info");
    // Force-clear cookies by calling logout
    fetch(`${API_BASE}/v1/auth/logout`, {
      method: "POST",
      credentials: "include",
    }).catch(() => {});
  };

  const handleSubmitReview = async () => {
    if (!banInfo || !clarification.trim()) return;
    setSubmitting(true);
    setError(null);

    try {
      const res = await fetch(`${API_BASE}/v1/auth/ban-review`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          login: banInfo.login,
          password: banInfo.password,
          clarification: clarification.trim(),
        }),
      });

      if (res.ok) {
        setSubmitted(true);
        setClarification("");
        // Update remaining count
        setBanInfo((prev) =>
          prev
            ? { ...prev, hasPendingReview: true, reviewsRemaining: prev.reviewsRemaining - 1 }
            : prev
        );
        // Update sessionStorage
        const updated = { ...banInfo, hasPendingReview: true, reviewsRemaining: banInfo.reviewsRemaining - 1 };
        sessionStorage.setItem("verdox_ban_info", JSON.stringify(updated));
      } else {
        const body = await res.json();
        const code = body.error?.code;
        if (code === "REVIEW_PENDING") {
          setBanInfo((prev) => prev ? { ...prev, hasPendingReview: true } : prev);
          setError("You already have a pending review request.");
        } else if (code === "REVIEW_LIMIT_REACHED") {
          setBanInfo((prev) => prev ? { ...prev, reviewsRemaining: 0 } : prev);
          setError("You have used all 3 review attempts.");
        } else if (code === "UNAUTHORIZED") {
          setError("Invalid credentials. Please go back to login and try again.");
        } else {
          setError(body.error?.message || "Failed to submit review request.");
        }
      }
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setSubmitting(false);
    }
  };

  // No ban info — generic message
  if (!banInfo) {
    return (
      <div className="text-center">
        <ShieldAlert className="h-12 w-12 text-danger mx-auto mb-4" />
        <h2 className="font-display text-[24px] text-text-primary mb-2">
          Account Banned
        </h2>
        <p className="text-[14px] text-text-secondary mb-6">
          Your account has been restricted. If you believe this is an error, please contact an administrator.
        </p>
        <Link
          href="/login"
          onClick={() => {
            sessionStorage.removeItem("verdox_ban_info");
            fetch(`${API_BASE}/v1/auth/logout`, { method: "POST", credentials: "include" }).catch(() => {});
          }}
          className="text-[14px] text-accent hover:text-accent-light font-medium"
        >
          Go to Sign In
        </Link>
      </div>
    );
  }

  return (
    <div>
      <div className="text-center mb-6">
        <ShieldAlert className="h-10 w-10 text-danger mx-auto mb-3" />
        <h2 className="font-display text-[24px] text-text-primary mb-1">
          Account Banned
        </h2>
        <p className="text-[14px] text-text-secondary">
          Your account has been suspended by an administrator.
        </p>
        {(banInfo.email || banInfo.username || banInfo.login) && (
          <p className="text-[13px] text-text-secondary mt-1">
            Account: {banInfo.email || banInfo.username || banInfo.login}
          </p>
        )}
      </div>

      {/* Ban reason */}
      {banInfo.reason && (
        <div className="rounded-[8px] border bg-bg-primary p-4 mb-4">
          <p className="text-[12px] text-text-secondary font-medium uppercase tracking-wide mb-1">
            Reason
          </p>
          <p className="text-[14px] text-text-primary">{banInfo.reason}</p>
        </div>
      )}

      {/* Pending review notice */}
      {banInfo.hasPendingReview && !submitted && (
        <div className="rounded-[8px] border border-warning/30 bg-warning/5 p-3 mb-4 flex items-start gap-2">
          <AlertTriangle className="h-4 w-4 text-warning shrink-0 mt-0.5" />
          <p className="text-[13px] text-text-secondary">
            Your review request is pending. An administrator will review your case.
          </p>
        </div>
      )}

      {/* Review submitted confirmation */}
      {submitted && (
        <div className="rounded-[8px] border border-success/30 bg-success/5 p-3 mb-4 flex items-start gap-2">
          <CheckCircle2 className="h-4 w-4 text-success shrink-0 mt-0.5" />
          <p className="text-[13px] text-text-secondary">
            Review request submitted. An administrator will review your case.
          </p>
        </div>
      )}

      {/* Review request form */}
      {!banInfo.hasPendingReview && banInfo.reviewsRemaining > 0 && !submitted && (
        <div className="mb-4">
          <div className="flex items-center justify-between mb-2">
            <p className="text-[13px] font-medium text-text-primary">
              Request a Review
            </p>
            <p className="text-[11px] text-text-tertiary">
              {banInfo.reviewsRemaining} attempt{banInfo.reviewsRemaining !== 1 ? "s" : ""} remaining
            </p>
          </div>
          <textarea
            value={clarification}
            onChange={(e) => setClarification(e.target.value)}
            placeholder="Explain why you believe this ban should be reviewed..."
            rows={4}
            className="w-full rounded-[4px] border bg-bg-primary px-3 py-2.5 text-[14px] text-text-primary placeholder:text-text-secondary focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20 resize-none mb-2"
          />
          {clarification.trim().length > 0 && clarification.trim().length < 10 && (
            <p className="text-[11px] text-text-tertiary mb-2">
              Minimum 10 characters required
            </p>
          )}
          <Button
            onClick={handleSubmitReview}
            loading={submitting}
            disabled={!clarification.trim() || clarification.trim().length < 10}
            className="w-full"
          >
            Submit Review Request
          </Button>
        </div>
      )}

      {/* No more attempts */}
      {!banInfo.hasPendingReview && banInfo.reviewsRemaining <= 0 && !submitted && (
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-3 mb-4">
          <p className="text-[13px] text-danger font-medium">
            You have used all 3 review attempts. Please contact an administrator directly.
          </p>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-3 mb-4">
          <p className="text-[13px] text-danger">{error}</p>
        </div>
      )}

      <div className="text-center pt-2">
        <Link
          href="/login"
          onClick={handleBackToSignIn}
          className="text-[14px] text-accent hover:text-accent-light font-medium"
        >
          Back to Sign In
        </Link>
      </div>
    </div>
  );
}
