import api from './client';
import { ApiResponse, Series } from '@/lib/types';

export interface JurusanPublic {
  id: string;
  nama: string;
  kode: string;
}

export interface KelasPublic {
  id: string;
  nama: string;
  tingkat: number;
  jurusan: {
    id: string;
    nama: string;
  };
}

interface KelasParams {
  jurusan_id?: string;
  tingkat?: number;
}

export const publicApi = {
  getJurusan: async (): Promise<ApiResponse<JurusanPublic[]>> => {
    const response = await api.get<ApiResponse<JurusanPublic[]>>('/jurusan');
    return response.data;
  },

  getKelas: async (params?: KelasParams): Promise<ApiResponse<KelasPublic[]>> => {
    const response = await api.get<ApiResponse<KelasPublic[]>>('/kelas', { params });
    return response.data;
  },
};

import { SeriesDetail } from '@/lib/types';

export const seriesApi = {
  getSeries: async (): Promise<ApiResponse<Series[]>> => {
    const response = await api.get<ApiResponse<Series[]>>('/series');
    return response.data;
  },

  getSeriesById: async (id: string): Promise<ApiResponse<SeriesDetail>> => {
    const response = await api.get<ApiResponse<SeriesDetail>>(`/series/${id}`);
    return response.data;
  },
};

// Top Students & Projects types
export interface TopStudent {
  id: string;
  username: string;
  nama: string;
  avatar_url: string | null;
  banner_url: string | null;
  kelas_nama: string | null;
  jurusan_nama: string | null;
  portfolio_count: number;
  total_likes: number;
  avg_assessment_score: number;
  follower_count: number;
  score: number;
}

export interface TopProject {
  id: string;
  judul: string;
  slug: string;
  thumbnail_url: string | null;
  published_at: string | null;
  user_id: string;
  username: string;
  user_nama: string;
  user_avatar: string | null;
  assessment_score: number;
  like_count: number;
  score: number;
}

export const topApi = {
  getTopStudents: async (): Promise<ApiResponse<TopStudent[]>> => {
    const response = await api.get<ApiResponse<TopStudent[]>>('/top-students');
    return response.data;
  },

  getTopProjects: async (): Promise<ApiResponse<TopProject[]>> => {
    const response = await api.get<ApiResponse<TopProject[]>>('/top-projects');
    return response.data;
  },
};
