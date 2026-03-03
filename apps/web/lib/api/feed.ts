import api from './client';
import {
  ApiResponse,
  FeedItem,
  FeedParams,
  FeedPreference,
  UpdateFeedPreferenceRequest,
} from '@/lib/types';

export const feedApi = {
  /**
   * Get feed with algorithm selection
   * @param params - algorithm (smart|recent|following), page, limit
   */
  getFeed: async (params?: FeedParams): Promise<ApiResponse<FeedItem[]>> => {
    const response = await api.get<ApiResponse<FeedItem[]>>('/feed', { params });
    return response.data;
  },

  /**
   * Get user's feed algorithm preference
   */
  getFeedPreferences: async (): Promise<ApiResponse<FeedPreference>> => {
    const response = await api.get<ApiResponse<FeedPreference>>('/feed/preferences');
    return response.data;
  },

  /**
   * Update user's feed algorithm preference
   * @param algorithm - smart, recent, or following
   */
  updateFeedPreferences: async (
    algorithm: UpdateFeedPreferenceRequest['algorithm']
  ): Promise<ApiResponse<FeedPreference>> => {
    const response = await api.put<ApiResponse<FeedPreference>>('/feed/preferences', {
      algorithm,
    });
    return response.data;
  },
};
