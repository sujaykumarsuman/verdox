export type TeamRole = "admin" | "maintainer" | "viewer";
export type MemberStatus = "pending" | "approved" | "rejected";

export interface Team {
  id: string;
  name: string;
  slug: string;
  my_role?: TeamRole;
  member_count: number;
  repo_count: number;
  created_at: string;
  updated_at: string;
}

export interface TeamDetail extends Team {
  members: TeamMember[];
  repositories: TeamRepo[];
}

export interface TeamMember {
  id: string;
  user_id: string;
  username: string;
  email: string;
  avatar_url: string | null;
  role: TeamRole;
  status: MemberStatus;
  invited_by: string | null;
  created_at: string;
}

export interface TeamJoinRequest {
  id: string;
  user: {
    id: string;
    username: string;
    email: string;
    avatar_url: string | null;
  };
  message: string | null;
  status: MemberStatus;
  role_assigned: TeamRole | null;
  reviewed_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface TeamRepo {
  id: string;
  team_id: string;
  repository_id: string;
  repository_name: string;
  github_full_name: string;
  is_active: boolean;
  added_by: string | null;
  created_at: string;
}

export interface DiscoverableTeam {
  id: string;
  name: string;
  slug: string;
  member_count: number;
  repo_count: number;
  created_at: string;
  user_status: "pending" | "approved" | null;
}
