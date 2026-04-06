import type { User } from "./user";

export interface AdminStats {
  total_users: number;
  active_users: number;
  total_repos: number;
  total_teams: number;
  total_test_runs: number;
  pass_rate_7d: number;
  runs_today: number;
  total_suites: number;
}

export interface AdminUser extends User {
  team_count: number;
}

export interface BanReview {
  id: string;
  user_id: string;
  username: string;
  email: string;
  ban_reason: string;
  clarification: string;
  status: string;
  created_at: string;
  reviewed_at: string | null;
}

export interface PendingBanReviewsResponse {
  reviews: BanReview[];
  count: number;
}

export interface UserTeamEntry {
  team_id: string;
  team_name: string;
  team_slug: string;
  role: string;
}

export interface AdminTeamEntry {
  id: string;
  name: string;
  slug: string;
  member_count: number;
}

export interface AdminUserListResponse {
  users: AdminUser[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}
