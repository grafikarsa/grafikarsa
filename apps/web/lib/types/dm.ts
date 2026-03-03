export type MessageType = 'text' | 'image' | 'portfolio' | 'system';

export interface User {
    id: string; // UUID
    username: string;
    nama: string;
    avatar_url?: string;
    is_online?: boolean; // Hydrated from presence
}

export interface Conversation {
    id: string;
    type: 'private' | 'group';
    participants: User[];
    last_message?: Message;
    unread_count: number;
    is_muted: boolean;
    is_archived: boolean;
    created_at: string;
    updated_at: string;
}

export interface Message {
    id: string;
    conversation_id: string;
    sender_id: string;
    sender?: User; // Hydrated
    message_type: MessageType;
    content: string;
    read_at?: string;
    created_at: string;
    updated_at: string;
    reply_to_id?: string;
    reply_to?: Message;
    reactions?: Reaction[];
    is_deleted: boolean;
}

export interface Reaction {
    id: string;
    message_id: string;
    user_id: string;
    emoji: string;
    created_at: string;
}

export interface DMSettings {
    user_id: string;
    dm_privacy: 'open' | 'followers' | 'mutual' | 'closed';
    show_read_receipts: boolean;
    show_typing_indicator: boolean;
}

export interface ChatStreak {
    user_id: string;
    partner_id: string;
    current_streak: number;
    last_message_at: string;
    created_at: string;
    updated_at: string;
}

// WebSocket Payload Types
export interface WSEvent<T = any> {
    type: WSEventType;
    payload: T;
}

export type WSEventType =
    | 'message.new'
    | 'message.deleted'
    | 'message.reaction'
    | 'typing'
    | 'presence'
    | 'read.receipt';

export interface WSTypingPayload {
    conversation_id: string;
    user_id: string;
    username: string;
    is_typing: boolean;
}

export interface WSPresencePayload {
    user_id: string;
    is_online: boolean;
}

export interface WSMessageNew {
    conversation_id: string;
    message: Message;
}

export interface WSMessageDeleted {
    conversation_id: string;
    message_id: string;
}

export interface WSMessageReaction {
    conversation_id: string;
    message_id: string;
    reaction: Reaction;
    action: 'add' | 'remove';
}

export interface WSReadReceiptPayload {
    conversation_id: string;
    user_id: string;
    read_at: string;
}
