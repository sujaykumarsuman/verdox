"use client";

import { useState, useEffect, useCallback } from "react";
import { use } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  KeyRound,
  CheckCircle2,
  XCircle,
  Loader2,
  Eye,
  EyeOff,
  Trash2,
  Globe,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { setPAT, revokePAT } from "@/hooks/use-pat";
import { deleteTeam } from "@/hooks/use-teams";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardBody, CardHeader, CardFooter } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

interface TeamDetail {
  id: string;
  name: string;
  slug: string;
}

interface PATStatus {
  is_configured?: boolean;
  valid?: boolean;
  github_username?: string;
  error?: string;
}

export default function TeamSettingsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id: teamId } = use(params);
  const router = useRouter();

  const [team, setTeam] = useState<TeamDetail | null>(null);
  const [patStatus, setPATStatus] = useState<PATStatus | null>(null);
  const [loadingTeam, setLoadingTeam] = useState(true);
  const [loadingPAT, setLoadingPAT] = useState(true);

  // PAT form state
  const [token, setToken] = useState("");
  const [showToken, setShowToken] = useState(false);
  const [saving, setSaving] = useState(false);
  const [revoking, setRevoking] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deletingTeam, setDeletingTeam] = useState(false);

  const fetchTeam = useCallback(async () => {
    try {
      // We don't have a dedicated GET /teams/:id yet, so use the list
      const teams = await api<TeamDetail[]>("/v1/teams");
      const found = teams.find((t) => t.id === teamId);
      setTeam(found || null);
    } catch {
      setTeam(null);
    } finally {
      setLoadingTeam(false);
    }
  }, [teamId]);

  const fetchPATStatus = useCallback(async () => {
    setLoadingPAT(true);
    try {
      const data = await api<PATStatus>(`/v1/teams/${teamId}/pat/validate`);
      setPATStatus(data);
    } catch {
      setPATStatus(null);
    } finally {
      setLoadingPAT(false);
    }
  }, [teamId]);

  useEffect(() => {
    fetchTeam();
    fetchPATStatus();
  }, [fetchTeam, fetchPATStatus]);

  const handleSavePAT = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token.trim()) return;

    setSaving(true);
    setSaveError(null);
    setSaveSuccess(false);
    try {
      await setPAT(teamId, token.trim());
      setToken("");
      setSaveSuccess(true);
      fetchPATStatus();
      setTimeout(() => setSaveSuccess(false), 3000);
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Failed to save PAT");
    } finally {
      setSaving(false);
    }
  };

  const handleRevokePAT = async () => {
    setRevoking(true);
    try {
      await revokePAT(teamId);
      setPATStatus({ is_configured: false });
    } catch {
      // Silently fail, user can retry
    } finally {
      setRevoking(false);
    }
  };

  const handleDeleteTeam = async () => {
    setDeletingTeam(true);
    try {
      await deleteTeam(teamId);
      router.push("/teams");
    } catch {
      setDeletingTeam(false);
    }
  };

  // Loading
  if (loadingTeam) {
    return (
      <div className="max-w-2xl">
        <Skeleton className="h-5 w-20 mb-4" />
        <Skeleton className="h-8 w-48 mb-8" />
        <Card>
          <CardBody className="space-y-4">
            <Skeleton className="h-6 w-40" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-32" />
          </CardBody>
        </Card>
      </div>
    );
  }

  if (!team) {
    return (
      <div className="max-w-2xl">
        <Link
          href="/teams"
          className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
        >
          <ArrowLeft className="h-4 w-4" />
          Teams
        </Link>
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-8 text-center">
          <p className="text-[16px] font-medium text-text-primary">Team not found</p>
          <p className="text-[14px] text-text-secondary mt-1">
            You may not have access to this team.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl">
      {/* Breadcrumb */}
      <Link
        href="/teams"
        className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        Teams
      </Link>

      {/* Header */}
      <div className="mb-8">
        <h1 className="font-display text-[28px] leading-[36px] tracking-[-0.01em] text-text-primary">
          {team.name}
        </h1>
        <p className="text-[14px] text-text-secondary mt-1">
          Team settings and integrations
        </p>
      </div>

      {/* GitHub PAT Configuration */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2 mb-1">
            <Globe className="h-5 w-5 text-text-primary" />
            <h2 className="text-[18px] font-semibold text-text-primary">
              GitHub Integration
            </h2>
          </div>
          <p className="text-[14px] text-text-secondary">
            Configure a Personal Access Token to allow Verdox to access your
            team&apos;s GitHub repositories.
          </p>
        </CardHeader>

        <CardBody className="space-y-4">
          {/* Current PAT status */}
          {loadingPAT ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin text-text-secondary" />
              <span className="text-[14px] text-text-secondary">Checking PAT status...</span>
            </div>
          ) : patStatus?.is_configured === false && !patStatus?.valid ? (
            <div className="flex items-center gap-2 p-3 rounded-[6px] bg-yellow-500/5 border border-yellow-300/20">
              <KeyRound className="h-4 w-4 text-yellow-600" />
              <span className="text-[14px] text-text-primary">
                No GitHub PAT configured. Add one below to enable repository access.
              </span>
            </div>
          ) : patStatus?.valid ? (
            <div className="flex items-center justify-between p-3 rounded-[6px] bg-green-500/5 border border-green-300/20">
              <div className="flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-green-600" />
                <span className="text-[14px] text-text-primary">
                  PAT active — connected as{" "}
                  <span className="font-semibold">{patStatus.github_username}</span>
                </span>
              </div>
              <Button
                variant="danger"
                size="sm"
                onClick={handleRevokePAT}
                loading={revoking}
              >
                <Trash2 className="h-3.5 w-3.5" />
                Revoke
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2 p-3 rounded-[6px] bg-red-500/5 border border-red-300/20">
              <XCircle className="h-4 w-4 text-red-600" />
              <span className="text-[14px] text-danger">
                PAT is invalid or expired. Please update it below.
              </span>
            </div>
          )}

          {/* Set/Update PAT form */}
          <form onSubmit={handleSavePAT} className="space-y-3">
            <div className="relative">
              <Input
                label={patStatus?.valid ? "Update PAT" : "GitHub Personal Access Token"}
                type={showToken ? "text" : "password"}
                placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                value={token}
                onChange={(e) => {
                  setToken(e.target.value);
                  setSaveError(null);
                  setSaveSuccess(false);
                }}
                error={saveError || undefined}
              />
              <button
                type="button"
                onClick={() => setShowToken(!showToken)}
                className="absolute right-3 top-[38px] text-text-secondary hover:text-text-primary"
              >
                {showToken ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            </div>

            {saveSuccess && (
              <div className="flex items-center gap-2 text-green-600">
                <CheckCircle2 className="h-4 w-4" />
                <span className="text-[13px]">PAT saved and validated successfully!</span>
              </div>
            )}

            <p className="text-[13px] text-text-secondary">
              Create a{" "}
              <a
                href="https://github.com/settings/tokens/new?scopes=repo&description=Verdox"
                target="_blank"
                rel="noopener noreferrer"
                className="text-accent hover:underline"
              >
                fine-grained or classic token
              </a>{" "}
              with <code className="bg-bg-tertiary px-1 rounded text-[12px]">repo</code> scope.
              The token is encrypted before storage.
            </p>

            <Button type="submit" loading={saving} disabled={!token.trim()}>
              <KeyRound className="h-4 w-4" />
              {patStatus?.valid ? "Update PAT" : "Save PAT"}
            </Button>
          </form>
        </CardBody>

        <CardFooter>
          <p className="text-[12px] text-text-secondary">
            Tokens are encrypted with AES-256-GCM and can only be used by this team.
            Only team admins can manage this setting.
          </p>
        </CardFooter>
      </Card>

      {/* Danger Zone */}
      <Card className="mt-8 border-danger/30">
        <CardBody>
          <h2 className="text-[16px] font-semibold text-danger mb-2">Danger Zone</h2>
          <p className="text-[14px] text-text-secondary mb-4">
            Permanently delete this team and remove all associated repository links.
          </p>
          {!showDeleteConfirm ? (
            <Button variant="danger" size="sm" onClick={() => setShowDeleteConfirm(true)}>
              <Trash2 className="h-4 w-4" />
              Delete Team
            </Button>
          ) : (
            <div className="flex items-center gap-3">
              <span className="text-[14px] text-danger">This cannot be undone. Are you sure?</span>
              <Button variant="danger" size="sm" onClick={handleDeleteTeam} loading={deletingTeam}>
                Yes, delete team
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setShowDeleteConfirm(false)}>
                Cancel
              </Button>
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
