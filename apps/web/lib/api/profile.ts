import api from './client';
import { ApiResponse, User, SocialLink } from '@/lib/types';

interface UpdateProfileRequest {
  nama?: string;
  username?: string;
  bio?: string;
  email?: string;
}

interface UpdatePasswordRequest {
  current_password: string;
  new_password: string;
  new_password_confirmation: string;
}

interface CheckUsernameResponse {
  username: string;
  available: boolean;
}

export const profileApi = {
  getMe: async (): Promise<ApiResponse<User>> => {
    const response = await api.get<ApiResponse<User>>('/me');
    return response.data;
  },

  updateMe: async (data: UpdateProfileRequest): Promise<ApiResponse<User>> => {
    const response = await api.patch<ApiResponse<User>>('/me', data);
    return response.data;
  },

  updatePassword: async (data: UpdatePasswordRequest): Promise<ApiResponse<null>> => {
    const response = await api.patch<ApiResponse<null>>('/me/password', data);
    return response.data;
  },

  updateSocialLinks: async (social_links: SocialLink[]): Promise<ApiResponse<{ social_links: SocialLink[] }>> => {
    const response = await api.put<ApiResponse<{ social_links: SocialLink[] }>>('/me/social-links', { social_links });
    return response.data;
  },

  checkUsername: async (username: string): Promise<ApiResponse<CheckUsernameResponse>> => {
    const response = await api.get<ApiResponse<CheckUsernameResponse>>('/me/check-username', {
      params: { username },
    });
    return response.data;
  },
};
