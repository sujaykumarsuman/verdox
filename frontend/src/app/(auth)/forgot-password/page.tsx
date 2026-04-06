"use client";

import { useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!email) {
      setError("Email is required");
      return;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setError("Must be a valid email");
      return;
    }

    setLoading(true);
    try {
      await api("/v1/auth/forgot-password", {
        method: "POST",
        body: JSON.stringify({ email }),
      });
      setSubmitted(true);
    } catch {
      toast.error("An unexpected error occurred");
    } finally {
      setLoading(false);
    }
  };

  if (submitted) {
    return (
      <div className="flex flex-col gap-4 text-center">
        <h2 className="font-display text-[24px] leading-[32px] tracking-[-0.01em] text-text-primary">
          Check Your Email
        </h2>
        <p className="text-[14px] text-text-secondary">
          If an account with that email exists, a password reset link has been sent.
        </p>
        <Link
          href="/login"
          className="text-[14px] text-accent hover:text-accent-light font-medium"
        >
          Back to login
        </Link>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <h2 className="font-display text-[24px] leading-[32px] tracking-[-0.01em] text-text-primary text-center mb-2">
        Reset Password
      </h2>
      <p className="text-[14px] text-text-secondary text-center mb-2">
        Enter your email and we&apos;ll send you a reset link.
      </p>

      <Input
        label="Email"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        error={error}
        placeholder="you@example.com"
        disabled={loading}
      />

      <Button type="submit" loading={loading} className="w-full">
        {loading ? "Sending..." : "Send Reset Link"}
      </Button>

      <Link
        href="/login"
        className="text-[14px] text-accent hover:text-accent-light font-medium text-center"
      >
        Back to login
      </Link>
    </form>
  );
}
