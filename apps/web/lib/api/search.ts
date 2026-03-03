import api from './client';
import type { PaginatedResponse } from '../types/api';

export interface SearchUser {
  id: string;
  username: string;
  nama: string;
  avatar_url?: string;
  bio?: string;
  role: string;
}

export interface SearchPortfolio {
  id: string;
  judul: string;
  slug: string;
  thumbnail_url?: string;
  user: {
    id: string;
    username: string;
    nama: string;
    avatar_url?: string;
  };
}

export async function searchUsers(query: string, page = 1, limit = 10) {
  const params = new URLSearchParams({
    q: query,
    page: page.toString(),
    limit: limit.toString(),
  });
  return api.get<PaginatedResponse<SearchUser[]>>(`/search/users?${params.toString()}`);
}

export async function searchPortfolios(query: string, page = 1, limit = 10) {
  const params = new URLSearchParams({
    q: query,
    page: page.toString(),
    limit: limit.toString(),
  });
  return api.get<PaginatedResponse<SearchPortfolio[]>>(`/search/portfolios?${params.toString()}`);
}
