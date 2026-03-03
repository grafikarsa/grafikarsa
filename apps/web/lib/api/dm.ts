import { api } from './client';
import {
    Conversation,
    Message,
    DMSettings,
    ChatStreak,
    MessageType
} from '@/lib/types/dm';

export const dmApi = {
    // Conversations
    getConversations: async (params?: { page?: number; limit?: number; include_archived?: boolean }) => {
        const { data } = await api.get('/conversations', { params });
        return data;
    },

    startConversation: async (recipientId: string, message: string) => {
        const { data } = await api.post('/conversations', {
            recipient_id: recipientId,
            message
        });
        return data;
    },

    getConversation: async (id: string) => {
        const { data } = await api.get(`/conversations/${id}`);
        return data;
    },

    archiveConversation: async (id: string) => {
        const { data } = await api.post(`/conversations/${id}/archive`);
        return data;
    },

    unarchiveConversation: async (id: string) => {
        const { data } = await api.delete(`/conversations/${id}/archive`);
        return data;
    },

    muteConversation: async (id: string) => {
        const { data } = await api.post(`/conversations/${id}/mute`);
        return data;
    },

    unmuteConversation: async (id: string) => {
        const { data } = await api.delete(`/conversations/${id}/mute`);
        return data;
    },

    markAsRead: async (id: string) => {
        const { data } = await api.post(`/conversations/${id}/read`);
        return data;
    },

    // Messages
    getMessages: async (conversationId: string, params?: { page?: number; limit?: number }) => {
        const { data } = await api.get(`/conversations/${conversationId}/messages`, { params });
        return data;
    },

    sendMessage: async (conversationId: string, content: string, type: MessageType = 'text', replyToId?: string) => {
        const { data } = await api.post(`/conversations/${conversationId}/messages`, {
            content,
            message_type: type,
            reply_to_id: replyToId
        });
        return data;
    },

    deleteMessage: async (id: string) => {
        const { data } = await api.delete(`/messages/${id}`);
        return data;
    },

    // Reactions
    addReaction: async (messageId: string, emoji: string) => {
        const { data } = await api.post(`/messages/${messageId}/reactions`, { emoji });
        return data;
    },

    removeReaction: async (messageId: string, emoji: string) => {
        const { data } = await api.delete(`/messages/${messageId}/reactions/${emoji}`);
        return data;
    },

    // Settings & Blocks
    getSettings: async () => {
        const { data } = await api.get('/dm/settings');
        return data;
    },

    updateSettings: async (settings: Partial<DMSettings>) => {
        const { data } = await api.patch('/dm/settings', settings);
        return data;
    },

    blockUser: async (userId: string) => {
        const { data } = await api.post(`/dm/block/${userId}`);
        return data;
    },

    unblockUser: async (userId: string) => {
        const { data } = await api.delete(`/dm/block/${userId}`);
        return data;
    },

    getBlockedUsers: async () => {
        const { data } = await api.get('/dm/blocked');
        return data;
    },

    getStreaks: async () => {
        const { data } = await api.get('/dm/streaks');
        return data;
    },
};
