export type NotificationType =
  | "system"
  | "admin_message"
  | "ban_review"
  | "test_complete"
  | "team_invite"
  | "team_join_request";

export interface Notification {
  id: string;
  type: NotificationType;
  subject: string;
  body: string;
  is_read: boolean;
  action_type: string | null;
  action_payload: Record<string, unknown> | null;
  sender_id: string | null;
  sender_username: string | null;
  created_at: string;
}

export interface NotificationListResponse {
  notifications: Notification[];
  total: number;
  page: number;
  per_page: number;
}

export interface UnreadCountResponse {
  count: number;
}
