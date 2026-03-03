export type NotificationType = 'new_follower' | 'portfolio_liked' | 'portfolio_approved' | 'portfolio_rejected' | 'feedback_updated' | 'new_comment' | 'reply_comment';

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message?: string;
  data?: Record<string, unknown>;
  is_read: boolean;
  read_at?: string;
  created_at: string;
}

export interface NotificationCountResponse {
  unread_count: number;
}

export interface NotificationListMeta {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  unread_count: number;
}
