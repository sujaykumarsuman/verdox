export type UserRole = "root" | "moderator" | "user";

export interface User {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  avatar_url: string | null;
  created_at: string;
  updated_at: string;
}
