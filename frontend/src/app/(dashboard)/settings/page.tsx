"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { api } from "@/lib/api";
import { Card, CardBody } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export default function SettingsPage() {
  const { user, isLoading: authLoading, refreshUser } = useAuth();

  // Profile form state
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [profileLoading, setProfileLoading] = useState(false);
  const [profileInit, setProfileInit] = useState(false);

  // Password form state
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordLoading, setPasswordLoading] = useState(false);

  // Initialize form with user data once loaded
  if (user && !profileInit) {
    setUsername(user.username);
    setEmail(user.email);
    setAvatarUrl(user.avatar_url || "");
    setProfileInit(true);
  }

  const handleProfileSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setProfileLoading(true);
    try {
      await api("/v1/users/me", {
        method: "PUT",
        body: JSON.stringify({
          username: username !== user?.username ? username : undefined,
          email: email !== user?.email ? email : undefined,
          avatar_url: avatarUrl || undefined,
        }),
      });
      await refreshUser();
      toast.success("Profile updated successfully");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update profile");
    } finally {
      setProfileLoading(false);
    }
  };

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (newPassword !== confirmPassword) {
      toast.error("New passwords do not match");
      return;
    }

    if (newPassword.length < 8) {
      toast.error("Password must be at least 8 characters");
      return;
    }

    setPasswordLoading(true);
    try {
      await api("/v1/users/me/password", {
        method: "PUT",
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.success("Password changed successfully");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to change password");
    } finally {
      setPasswordLoading(false);
    }
  };

  if (authLoading) {
    return (
      <div className="max-w-2xl mx-auto">
        <Skeleton className="h-9 w-32 mb-2" />
        <Skeleton className="h-5 w-64 mb-8" />
        <div className="space-y-6">
          <div className="rounded-[8px] border bg-bg-secondary p-6 space-y-4">
            <Skeleton className="h-5 w-24" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-6">
        <h1 className="font-display text-[30px] leading-[38px] tracking-[-0.01em] text-text-primary mb-1">
          Settings
        </h1>
        <p className="text-[14px] text-text-secondary">
          Manage your profile and account settings.
        </p>
      </div>

      {/* Profile Section */}
      <Card className="mb-6">
        <CardBody>
          <h2 className="text-[16px] font-semibold text-text-primary mb-4">
            Profile
          </h2>
          <form onSubmit={handleProfileSubmit} className="space-y-4">
            <Input
              label="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Your username"
            />
            <Input
              label="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="your@email.com"
            />
            <Input
              label="Avatar URL"
              value={avatarUrl}
              onChange={(e) => setAvatarUrl(e.target.value)}
              placeholder="https://example.com/avatar.png"
            />
            <div className="flex justify-end pt-2">
              <Button type="submit" disabled={profileLoading}>
                {profileLoading && <Loader2 className="h-4 w-4 animate-spin" />}
                Save Profile
              </Button>
            </div>
          </form>
        </CardBody>
      </Card>

      {/* Password Section */}
      <Card>
        <CardBody>
          <h2 className="text-[16px] font-semibold text-text-primary mb-4">
            Change Password
          </h2>
          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <Input
              label="Current Password"
              type="password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              placeholder="Enter current password"
            />
            <Input
              label="New Password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="Enter new password"
            />
            <Input
              label="Confirm New Password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="Confirm new password"
              error={
                confirmPassword && newPassword !== confirmPassword
                  ? "Passwords do not match"
                  : undefined
              }
            />
            <div className="flex justify-end pt-2">
              <Button
                type="submit"
                disabled={passwordLoading || !currentPassword || !newPassword || !confirmPassword}
              >
                {passwordLoading && <Loader2 className="h-4 w-4 animate-spin" />}
                Change Password
              </Button>
            </div>
          </form>
        </CardBody>
      </Card>
    </div>
  );
}
