export type UserRole = "root" | "admin" | "moderator" | "user";

export interface User {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  avatar_url: string | null;
  is_active: boolean;
  is_banned: boolean;
  ban_reason: string | null;
  created_at: string;
  updated_at: string;
}
