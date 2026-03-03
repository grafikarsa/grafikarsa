import { Tag } from './portfolio';

export type FeedAlgorithm = 'smart' | 'recent' | 'following';

export interface FeedUser {
  id: string;
  username: string;
  nama: string;
  avatar_url?: string;
  role: string;
  kelas_nama?: string;
}

export interface FeedItem {
  id: string;
  judul: string;
  slug: string;
  thumbnail_url?: string;
  preview_text?: string;
  published_at?: string;
  created_at?: string;
  like_count: number;
  view_count: number;
  is_liked: boolean;
  ranking_score?: number;
  user?: FeedUser;
  tags?: Tag[];
}

export interface FeedPreference {
  algorithm: FeedAlgorithm;
}

export interface FeedParams {
  algorithm?: FeedAlgorithm;
  page?: number;
  limit?: number;
}

export interface UpdateFeedPreferenceRequest {
  algorithm: FeedAlgorithm;
}
