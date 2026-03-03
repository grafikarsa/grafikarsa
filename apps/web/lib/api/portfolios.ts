import api from './client';
import { ApiResponse, Portfolio, PortfolioCard, Tag, ContentBlock, PaginationParams } from '@/lib/types';

interface PortfoliosParams extends PaginationParams {
  search?: string;
  tag_ids?: string;
  jurusan_id?: string;
  kelas_id?: string;
  user_id?: string;
  series_id?: string;
  status?: string;
  sort?: string;
}

interface CreatePortfolioRequest {
  judul: string;
  tag_ids?: string[];
  series_id?: string;
}

interface UpdatePortfolioRequest {
  judul?: string;
  thumbnail_url?: string;
  tag_ids?: string[];
  series_id?: string;
}

interface LikeResponse {
  is_liked: boolean;
  like_count: number;
}

export const portfoliosApi = {
  getPortfolios: async (params?: PortfoliosParams): Promise<ApiResponse<PortfolioCard[]>> => {
    const response = await api.get<ApiResponse<PortfolioCard[]>>('/portfolios', { params });
    return response.data;
  },

  getPortfolioBySlug: async (slug: string, username?: string): Promise<ApiResponse<Portfolio>> => {
    const response = await api.get<ApiResponse<Portfolio>>(`/portfolios/${slug}`, {
      params: username ? { username } : undefined,
    });
    return response.data;
  },

  getPortfolioById: async (id: string): Promise<ApiResponse<Portfolio>> => {
    const response = await api.get<ApiResponse<Portfolio>>(`/portfolios/id/${id}`);
    return response.data;
  },

  getMyPortfolios: async (params?: PaginationParams & { status?: string }): Promise<ApiResponse<PortfolioCard[]>> => {
    const response = await api.get<ApiResponse<PortfolioCard[]>>('/me/portfolios', { params });
    return response.data;
  },

  createPortfolio: async (data: CreatePortfolioRequest): Promise<ApiResponse<Portfolio>> => {
    const response = await api.post<ApiResponse<Portfolio>>('/portfolios', data);
    return response.data;
  },

  updatePortfolio: async (id: string, data: UpdatePortfolioRequest): Promise<ApiResponse<Portfolio>> => {
    const response = await api.patch<ApiResponse<Portfolio>>(`/portfolios/${id}`, data);
    return response.data;
  },

  deletePortfolio: async (id: string): Promise<ApiResponse<null>> => {
    const response = await api.delete<ApiResponse<null>>(`/portfolios/${id}`);
    return response.data;
  },

  submitPortfolio: async (id: string): Promise<ApiResponse<Portfolio>> => {
    const response = await api.post<ApiResponse<Portfolio>>(`/portfolios/${id}/submit`);
    return response.data;
  },

  archivePortfolio: async (id: string): Promise<ApiResponse<Portfolio>> => {
    const response = await api.post<ApiResponse<Portfolio>>(`/portfolios/${id}/archive`);
    return response.data;
  },

  unarchivePortfolio: async (id: string): Promise<ApiResponse<Portfolio>> => {
    const response = await api.post<ApiResponse<Portfolio>>(`/portfolios/${id}/unarchive`);
    return response.data;
  },

  likePortfolio: async (id: string): Promise<ApiResponse<LikeResponse>> => {
    const response = await api.post<ApiResponse<LikeResponse>>(`/portfolios/${id}/like`);
    return response.data;
  },

  unlikePortfolio: async (id: string): Promise<ApiResponse<LikeResponse>> => {
    const response = await api.delete<ApiResponse<LikeResponse>>(`/portfolios/${id}/like`);
    return response.data;
  },

  // Content Blocks
  addBlock: async (portfolioId: string, block: Omit<ContentBlock, 'id'>): Promise<ApiResponse<ContentBlock>> => {
    const response = await api.post<ApiResponse<ContentBlock>>(`/portfolios/${portfolioId}/blocks`, block);
    return response.data;
  },

  updateBlock: async (
    portfolioId: string,
    blockId: string,
    data: Partial<ContentBlock>
  ): Promise<ApiResponse<ContentBlock>> => {
    const response = await api.patch<ApiResponse<ContentBlock>>(`/portfolios/${portfolioId}/blocks/${blockId}`, data);
    return response.data;
  },

  reorderBlocks: async (
    portfolioId: string,
    block_orders: Array<{ id: string; order: number }>
  ): Promise<ApiResponse<null>> => {
    const response = await api.put<ApiResponse<null>>(`/portfolios/${portfolioId}/blocks/reorder`, { block_orders });
    return response.data;
  },

  deleteBlock: async (portfolioId: string, blockId: string): Promise<ApiResponse<null>> => {
    const response = await api.delete<ApiResponse<null>>(`/portfolios/${portfolioId}/blocks/${blockId}`);
    return response.data;
  },
};

// Tags API
export const tagsApi = {
  getTags: async (search?: string): Promise<ApiResponse<Tag[]>> => {
    const response = await api.get<ApiResponse<Tag[]>>('/tags', { params: search ? { search } : undefined });
    return response.data;
  },
};
