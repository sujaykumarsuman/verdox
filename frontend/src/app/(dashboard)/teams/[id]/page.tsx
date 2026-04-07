"use client";

import { useState, useCallback } from "react";
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
  Users,
  GitBranch,
  Settings,
  UserPlus,
  Plus,
  ClipboardList,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useTeamDetail, deleteTeam, useJoinRequests } from "@/hooks/use-teams";
import { setPAT, revokePAT } from "@/hooks/use-pat";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardBody, CardHeader, CardFooter } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { MemberList } from "@/components/team/member-list";
import { RepoList } from "@/components/team/repo-list";
import { InviteMemberDialog } from "@/components/team/invite-member-dialog";
import { AssignRepoDialog } from "@/components/team/assign-repo-dialog";

interface PATStatus {
  is_configured?: boolean;
  valid?: boolean;
  github_username?: string;
  error?: string;
}

type Tab = "members" | "repositories" | "settings";

export default function TeamDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id: teamId } = use(params);
  const router = useRouter();
  const { team, isLoading, error, refetch } = useTeamDetail(teamId);

  const [activeTab, setActiveTab] = useState<Tab>("members");
  const [showInviteDialog, setShowInviteDialog] = useState(false);
  const [showAssignDialog, setShowAssignDialog] = useState(false);

  // PAT state
  const [patStatus, setPATStatus] = useState<PATStatus | null>(null);
  const [loadingPAT, setLoadingPAT] = useState(false);
  const [token, setToken] = useState("");
  const [showToken, setShowToken] = useState(false);
  const [saving, setSaving] = useState(false);
  const [revoking, setRevoking] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deletingTeam, setDeletingTeam] = useState(false);

  // Get current user ID from auth context
  const [currentUserId, setCurrentUserId] = useState<string>("");
  const fetchCurrentUser = useCallback(async () => {
    try {
      const me = await api<{ id: string }>("/v1/auth/me");
      setCurrentUserId(me.id);
    } catch {
      // ignore
    }
  }, []);
  // Fetch on mount
  useState(() => {
    fetchCurrentUser();
  });

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

  // Fetch PAT when settings tab is selected
  const handleTabChange = (tab: Tab) => {
    setActiveTab(tab);
    if (tab === "settings" && patStatus === null) {
      fetchPATStatus();
    }
  };

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
      // retry
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

  const myRole = team?.my_role;
  const isAdmin = myRole === "admin";
  const isAdminOrMaintainer = isAdmin || myRole === "maintainer";

  // Fetch pending join requests count for admins/maintainers
  const { requests: pendingRequests } = useJoinRequests(
    isAdminOrMaintainer ? teamId : "",
    "pending"
  );
  const pendingCount = pendingRequests.length;

  if (isLoading) {
    return (
      <div className="max-w-4xl">
        <Skeleton className="h-5 w-20 mb-4" />
        <Skeleton className="h-8 w-48 mb-8" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !team) {
    return (
      <div className="max-w-4xl">
        <Link
          href="/teams"
          className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
        >
          <ArrowLeft className="h-4 w-4" />
          Teams
        </Link>
        <div className="rounded-[8px] border border-danger/30 bg-danger/5 p-8 text-center">
          <p className="text-[16px] font-medium text-text-primary">
            {error || "Team not found"}
          </p>
          <p className="text-[14px] text-text-secondary mt-1">
            You may not have access to this team.
          </p>
        </div>
      </div>
    );
  }

  const tabs: { key: Tab; label: string; icon: React.ReactNode; show: boolean }[] = [
    { key: "members", label: "Members", icon: <Users className="h-4 w-4" />, show: true },
    { key: "repositories", label: "Repositories", icon: <GitBranch className="h-4 w-4" />, show: true },
    { key: "settings", label: "Settings", icon: <Settings className="h-4 w-4" />, show: isAdmin },
  ];

  return (
    <div className="max-w-4xl">
      {/* Breadcrumb */}
      <Link
        href="/teams"
        className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-text-primary mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        Teams
      </Link>

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="font-display text-[28px] leading-[36px] tracking-[-0.01em] text-text-primary">
            {team.name}
          </h1>
          <div className="flex items-center gap-2 mt-1">
            <span className="text-[14px] text-text-secondary">{team.slug}</span>
            {myRole && <Badge variant="info">{myRole}</Badge>}
          </div>
        </div>
        {isAdminOrMaintainer && (
          <Link
            href={`/teams/${teamId}/requests`}
            className="inline-flex items-center gap-1.5 text-[14px] text-text-secondary hover:text-accent"
          >
            <ClipboardList className="h-4 w-4" />
            Join Requests
            {pendingCount > 0 && (
              <span className="ml-1 inline-flex items-center justify-center h-5 min-w-[20px] px-1.5 rounded-full bg-accent text-white text-[11px] font-semibold">
                {pendingCount}
              </span>
            )}
          </Link>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border mb-6">
        {tabs
          .filter((t) => t.show)
          .map((t) => (
            <button
              key={t.key}
              onClick={() => handleTabChange(t.key)}
              className={`flex items-center gap-1.5 px-4 py-2.5 text-[14px] font-medium border-b-2 transition-colors ${
                activeTab === t.key
                  ? "border-accent text-accent"
                  : "border-transparent text-text-secondary hover:text-text-primary"
              }`}
            >
              {t.icon}
              {t.label}
              {t.key === "members" && (
                <span className="ml-1 text-[12px] text-text-secondary">
                  ({team.members.filter((m) => m.status === "approved").length})
                </span>
              )}
              {t.key === "repositories" && (
                <span className="ml-1 text-[12px] text-text-secondary">
                  ({team.repositories.length})
                </span>
              )}
            </button>
          ))}
      </div>

      {/* Members Tab */}
      {activeTab === "members" && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <h2 className="text-[16px] font-semibold text-text-primary">Team Members</h2>
              {isAdminOrMaintainer && (
                <Button size="sm" onClick={() => setShowInviteDialog(true)}>
                  <UserPlus className="h-4 w-4" />
                  Invite
                </Button>
              )}
            </div>
          </CardHeader>
          <CardBody>
            <MemberList
              teamId={teamId}
              members={team.members}
              currentUserId={currentUserId}
              currentUserRole={myRole}
              onRefresh={refetch}
            />
          </CardBody>
        </Card>
      )}

      {/* Repositories Tab */}
      {activeTab === "repositories" && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <h2 className="text-[16px] font-semibold text-text-primary">Repositories</h2>
              {isAdminOrMaintainer && (
                <Button size="sm" onClick={() => setShowAssignDialog(true)}>
                  <Plus className="h-4 w-4" />
                  Assign Repository
                </Button>
              )}
            </div>
          </CardHeader>
          <CardBody>
            <RepoList
              teamId={teamId}
              repos={team.repositories}
              currentUserRole={myRole}
              onRefresh={refetch}
            />
          </CardBody>
        </Card>
      )}

      {/* Settings Tab (admin only) */}
      {activeTab === "settings" && isAdmin && (
        <>
          {/* GitHub PAT */}
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2 mb-1">
                <Globe className="h-5 w-5 text-text-primary" />
                <h2 className="text-[18px] font-semibold text-text-primary">
                  GitHub Integration
                </h2>
              </div>
              <p className="text-[14px] text-text-secondary">
                Optionally configure a team-level PAT to access private repositories.
                The Verdox service account PAT is used by default for forking and running tests.
              </p>
            </CardHeader>
            <CardBody className="space-y-4">
              {loadingPAT ? (
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin text-text-secondary" />
                  <span className="text-[14px] text-text-secondary">Checking PAT status...</span>
                </div>
              ) : patStatus?.is_configured === false && !patStatus?.valid ? (
                <div className="flex items-center gap-2 p-3 rounded-[6px] bg-yellow-500/5 border border-yellow-300/20">
                  <KeyRound className="h-4 w-4 text-yellow-600" />
                  <span className="text-[14px] text-text-primary">
                    No team PAT configured. The Verdox service account PAT is being used. Add a team PAT below only if your repos need a different account for access.
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
                  <Button variant="danger" size="sm" onClick={handleRevokePAT} loading={revoking}>
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
                    {showToken ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
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
                  <span className="text-[14px] text-danger">
                    This cannot be undone. Are you sure?
                  </span>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={handleDeleteTeam}
                    loading={deletingTeam}
                  >
                    Yes, delete team
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setShowDeleteConfirm(false)}>
                    Cancel
                  </Button>
                </div>
              )}
            </CardBody>
          </Card>
        </>
      )}

      {/* Dialogs */}
      <InviteMemberDialog
        teamId={teamId}
        open={showInviteDialog}
        onClose={() => setShowInviteDialog(false)}
        onSuccess={refetch}
      />
      <AssignRepoDialog
        teamId={teamId}
        open={showAssignDialog}
        onClose={() => setShowAssignDialog(false)}
        onSuccess={refetch}
      />
    </div>
  );
}
