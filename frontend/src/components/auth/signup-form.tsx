"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiRequestError } from "@/lib/api";

export function SignupForm() {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const { signup } = useAuth();
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors({});

    // Client-side validation
    const newErrors: Record<string, string> = {};
    if (!username) newErrors.username = "Username is required";
    else if (username.length < 3) newErrors.username = "Must be at least 3 characters";
    else if (!/^[a-zA-Z0-9_]+$/.test(username)) newErrors.username = "Only letters, numbers, and underscores";

    if (!email) newErrors.email = "Email is required";
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) newErrors.email = "Must be a valid email";

    if (!password) newErrors.password = "Password is required";
    else if (password.length < 8) newErrors.password = "Must be at least 8 characters";
    else if (!/[A-Z]/.test(password)) newErrors.password = "Must include an uppercase letter";
    else if (!/[a-z]/.test(password)) newErrors.password = "Must include a lowercase letter";
    else if (!/[0-9]/.test(password)) newErrors.password = "Must include a digit";

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setLoading(true);
    try {
      await signup(username, email, password);
      router.push("/dashboard");
    } catch (err) {
      if (err instanceof ApiRequestError) {
        if (err.code === "CONFLICT") {
          toast.error(err.message);
        } else if (err.details) {
          setErrors(err.details);
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
        Create Account
      </h2>

      <Input
        label="Username"
        type="text"
        value={username}
        onChange={(e) => setUsername(e.target.value)}
        error={errors.username}
        placeholder="johndoe"
        disabled={loading}
        autoComplete="username"
      />

      <Input
        label="Email"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        error={errors.email}
        placeholder="you@example.com"
        disabled={loading}
        autoComplete="email"
      />

      <Input
        label="Password"
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        error={errors.password}
        placeholder="Min 8 chars, 1 upper, 1 lower, 1 digit"
        disabled={loading}
        autoComplete="new-password"
      />

      <Button type="submit" loading={loading} className="w-full">
        {loading ? "Creating account..." : "Sign Up"}
      </Button>

      <p className="text-[14px] text-text-secondary text-center">
        Already have an account?{" "}
        <Link href="/login" className="text-accent hover:text-accent-light font-medium">
          Sign In
        </Link>
      </p>
    </form>
  );
}
