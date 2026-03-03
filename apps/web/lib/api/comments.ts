import { api } from './client';
import { Comment, CreateCommentRequest } from '@/lib/types';
import { AxiosPromise } from 'axios';

export const commentsApi = {
    create: (data: CreateCommentRequest): AxiosPromise<{ data: Comment }> => {
        return api.post(`/portfolios/${data.portfolio_id}/comments`, data);
    },

    getByPortfolioID: (portfolioID: string): AxiosPromise<{ data: Comment[] }> => {
        return api.get(`/portfolios/${portfolioID}/comments`);
    },

    delete: (id: string): AxiosPromise<void> => {
        return api.delete(`/comments/${id}`);
    },
};
