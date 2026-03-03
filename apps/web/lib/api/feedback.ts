import api from './client';
import { ApiResponse } from '@/lib/types';

export type FeedbackKategori = 'bug' | 'saran' | 'lainnya';

export interface CreateFeedbackRequest {
    kategori: FeedbackKategori;
    pesan: string;
}

export const feedbackApi = {
    createFeedback: async (data: CreateFeedbackRequest): Promise<ApiResponse<void>> => {
        const response = await api.post<ApiResponse<void>>('/feedback', data);
        return response.data;
    },
};
