import api from './client';
import { ApiResponse, User } from '@/lib/types';

interface LoginRequest {
  username: string;
  password: string;
}

interface LoginResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: User;
}

interface RefreshResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

interface Session {
  id: string;
  device_info: {
    user_agent: string;
    device_type: string;
    browser?: string;
    os?: string;
  };
  ip_address: string;
  created_at: string;
  last_used_at?: string;
  is_current: boolean;
}

export const authApi = {
  login: async (data: LoginRequest): Promise<ApiResponse<LoginResponse>> => {
    const response = await api.post<ApiResponse<LoginResponse>>('/auth/login', data);
    return response.data;
  },

  refresh: async (): Promise<ApiResponse<RefreshResponse>> => {
    const response = await api.post<ApiResponse<RefreshResponse>>('/auth/refresh');
    return response.data;
  },

  logout: async (): Promise<ApiResponse<null>> => {
    const response = await api.post<ApiResponse<null>>('/auth/logout');
    return response.data;
  },

  logoutAll: async (): Promise<ApiResponse<{ sessions_terminated: number }>> => {
    const response = await api.post<ApiResponse<{ sessions_terminated: number }>>('/auth/logout-all');
    return response.data;
  },

  getSessions: async (): Promise<ApiResponse<Session[]>> => {
    const response = await api.get<ApiResponse<Session[]>>('/auth/sessions');
    return response.data;
  },

  deleteSession: async (sessionId: string): Promise<ApiResponse<null>> => {
    const response = await api.delete<ApiResponse<null>>(`/auth/sessions/${sessionId}`);
    return response.data;
  },
};
