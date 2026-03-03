import api from './client';
import { ApiResponse, Notification, NotificationCountResponse, NotificationListMeta } from '@/lib/types';

interface NotificationListParams {
  page?: number;
  limit?: number;
  unread_only?: boolean;
}

interface NotificationListResponse {
  success: boolean;
  data: Notification[];
  meta: NotificationListMeta;
}

export const notificationsApi = {
  getNotifications: async (params?: NotificationListParams): Promise<NotificationListResponse> => {
    const response = await api.get<NotificationListResponse>('/notifications', { params });
    return response.data;
  },

  getUnreadCount: async (): Promise<ApiResponse<NotificationCountResponse>> => {
    const response = await api.get<ApiResponse<NotificationCountResponse>>('/notifications/count');
    return response.data;
  },

  markAsRead: async (id: string): Promise<ApiResponse<null>> => {
    const response = await api.patch<ApiResponse<null>>(`/notifications/${id}/read`);
    return response.data;
  },

  markAllAsRead: async (): Promise<ApiResponse<null>> => {
    const response = await api.post<ApiResponse<null>>('/notifications/read-all');
    return response.data;
  },

  deleteNotification: async (id: string): Promise<ApiResponse<null>> => {
    const response = await api.delete<ApiResponse<null>>(`/notifications/${id}`);
    return response.data;
  },
};
