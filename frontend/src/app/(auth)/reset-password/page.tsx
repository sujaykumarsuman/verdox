"use client";

import { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { api, ApiRequestError } from "@/lib/api";

function ResetPasswordForm() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const router = useRouter();

  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  if (!token) {
    return (
      <div className="flex flex-col gap-4 text-center">
        <h2 className="font-display text-[24px] leading-[32px] tracking-[-0.01em] text-text-primary">
          Invalid Reset Link
        </h2>
        <p className="text-[14px] text-text-secondary">
          The reset link is invalid or missing. Please request a new one.
        </p>
        <Link
          href="/forgot-password"
          className="text-[14px] text-accent hover:text-accent-light font-medium"
        >
          Request new reset link
        </Link>
      </div>
    );
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors({});

    const newErrors: Record<string, string> = {};
    if (!newPassword) newErrors.newPassword = "New password is required";
    else if (newPassword.length < 8) newErrors.newPassword = "Must be at least 8 characters";
    else if (!/[A-Z]/.test(newPassword)) newErrors.newPassword = "Must include an uppercase letter";
    else if (!/[a-z]/.test(newPassword)) newErrors.newPassword = "Must include a lowercase letter";
    else if (!/[0-9]/.test(newPassword)) newErrors.newPassword = "Must include a digit";

    if (!confirmPassword) newErrors.confirmPassword = "Please confirm your password";
    else if (newPassword !== confirmPassword) newErrors.confirmPassword = "Passwords do not match";

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setLoading(true);
    try {
      await api("/v1/auth/reset-password", {
        method: "POST",
        body: JSON.stringify({ token, new_password: newPassword }),
      });
      toast.success("Password reset successfully. Redirecting to login...");
      setTimeout(() => router.push("/login"), 2000);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        toast.error(err.message);
      } else {
        toast.error("An unexpected error occurred");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <h2 className="font-display text-[24px] leading-[32px] tracking-[-0.01em] text-text-primary text-center mb-2">
        Set New Password
      </h2>

      <Input
        label="New Password"
        type="password"
        value={newPassword}
        onChange={(e) => setNewPassword(e.target.value)}
        error={errors.newPassword}
        placeholder="Min 8 chars, 1 upper, 1 lower, 1 digit"
        disabled={loading}
        autoComplete="new-password"
      />

      <Input
        label="Confirm Password"
        type="password"
        value={confirmPassword}
        onChange={(e) => setConfirmPassword(e.target.value)}
        error={errors.confirmPassword}
        placeholder="Re-enter your password"
        disabled={loading}
        autoComplete="new-password"
      />

      <Button type="submit" loading={loading} className="w-full">
        {loading ? "Resetting..." : "Reset Password"}
      </Button>
    </form>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense>
      <ResetPasswordForm />
    </Suspense>
  );
}
