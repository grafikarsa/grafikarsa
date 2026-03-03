import api from './client';
import { ApiResponse, User, UserCard, FollowUser, PaginationParams } from '@/lib/types';

interface UsersParams extends PaginationParams {
  search?: string;
  jurusan_id?: string;
  kelas_id?: string;
  role?: string;
}

interface FollowResponse {
  is_following: boolean;
  follower_count: number;
}

export const usersApi = {
  getUsers: async (params?: UsersParams): Promise<ApiResponse<UserCard[]>> => {
    const response = await api.get<ApiResponse<UserCard[]>>('/users', { params });
    return response.data;
  },

  search: async (query: string): Promise<ApiResponse<UserCard[]>> => {
    const response = await api.get<ApiResponse<UserCard[]>>('/users', { params: { search: query } });
    return response.data;
  },


  getUserByUsername: async (username: string): Promise<ApiResponse<User>> => {
    const response = await api.get<ApiResponse<User>>(`/users/${username}`);
    return response.data;
  },

  getFollowers: async (
    username: string,
    params?: PaginationParams & { search?: string }
  ): Promise<ApiResponse<FollowUser[]>> => {
    const response = await api.get<ApiResponse<FollowUser[]>>(`/users/${username}/followers`, { params });
    return response.data;
  },

  getFollowing: async (
    username: string,
    params?: PaginationParams & { search?: string }
  ): Promise<ApiResponse<FollowUser[]>> => {
    const response = await api.get<ApiResponse<FollowUser[]>>(`/users/${username}/following`, { params });
    return response.data;
  },

  follow: async (username: string): Promise<ApiResponse<FollowResponse>> => {
    const response = await api.post<ApiResponse<FollowResponse>>(`/users/${username}/follow`);
    return response.data;
  },

  unfollow: async (username: string): Promise<ApiResponse<FollowResponse>> => {
    const response = await api.delete<ApiResponse<FollowResponse>>(`/users/${username}/follow`);
    return response.data;
  },
};
