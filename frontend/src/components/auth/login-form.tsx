"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Eye, EyeOff } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiRequestError } from "@/lib/api";

export function LoginForm() {
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const { login: authLogin } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors({});

    const newErrors: Record<string, string> = {};
    if (!login) newErrors.login = "Email or username is required";
    if (!password) newErrors.password = "Password is required";
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setLoading(true);
    try {
      await authLogin(login, password);
      const returnTo = searchParams.get("returnTo") || "/dashboard";
      router.push(returnTo);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        if (err.code === "ACCOUNT_BANNED") {
          // Store ban info + credentials for the /banned page
          try {
            sessionStorage.setItem("verdox_ban_info", JSON.stringify({
              reason: (err.details?.ban_reason as string) || "",
              hasPendingReview: (err.details?.has_pending_review as boolean) || false,
              reviewsRemaining: (err.details?.reviews_remaining as number) ?? 0,
              login,
              password,
              email: login, // login could be email or username
            }));
          } catch {}
          router.push("/banned");
          return;
        }
        if (err.code === "ACCOUNT_DEACTIVATED") {
          toast.error(err.message);
        } else if (err.details) {
          setErrors(err.details as Record<string, string>);
        } else {
          toast.error(err.message);
        }
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
        Sign In
      </h2>

      <Input
        label="Email or Username"
        type="text"
        value={login}
        onChange={(e) => setLogin(e.target.value)}
        error={errors.login}
        placeholder="you@example.com"
        disabled={loading}
        autoComplete="username"
      />

      <div className="relative">
        <Input
          label="Password"
          type={showPassword ? "text" : "password"}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          error={errors.password}
          placeholder="Enter your password"
          disabled={loading}
          autoComplete="current-password"
        />
        <button
          type="button"
          onClick={() => setShowPassword(!showPassword)}
          className="absolute right-3 top-[38px] text-text-secondary hover:text-text-primary"
          tabIndex={-1}
        >
          {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </button>
      </div>

      <div className="flex justify-end">
        <Link
          href="/forgot-password"
          className="text-[14px] text-accent hover:text-accent-light"
        >
          Forgot password?
        </Link>
      </div>

      <Button type="submit" loading={loading} className="w-full">
        {loading ? "Signing in..." : "Sign In"}
      </Button>

      <p className="text-[14px] text-text-secondary text-center">
        Don&apos;t have an account?{" "}
        <Link href="/signup" className="text-accent hover:text-accent-light font-medium">
          Sign Up
        </Link>
      </p>
    </form>
  );
}
