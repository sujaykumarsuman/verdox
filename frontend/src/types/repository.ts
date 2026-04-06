export type CloneStatus = "pending" | "cloning" | "ready" | "failed" | "evicted";

export interface Repository {
  id: string;
  github_repo_id: number;
  github_full_name: string;
  name: string;
  description: string | null;
  default_branch: string;
  clone_status: CloneStatus;
  is_active: boolean;
  team_id: string;
  created_at: string;
  updated_at: string;
}

export interface Branch {
  name: string;
  commit_sha: string;
}

export interface Commit {
  sha: string;
  message: string;
  author: string;
  date: string;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface RepositoryListResponse {
  repositories: Repository[];
  meta: PaginationMeta;
}
