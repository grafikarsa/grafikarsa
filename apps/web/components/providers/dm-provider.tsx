'use client';

import {
    createContext,
    useContext,
    useEffect,
    useState,
    useCallback,
    useRef,
    type ReactNode
} from 'react';
import { useAuthStore } from '@/lib/stores/auth-store';
import {
    Conversation,
    Message,
    WSEvent,
    WSEventType,
    WSTypingPayload,
    WSMessageNew,
    WSMessageDeleted,
    WSMessageReaction,
    WSReadReceiptPayload,
    WSPresencePayload
} from '@/lib/types/dm';
import { dmApi } from '@/lib/api/dm';
import { toast } from 'sonner'; // Assuming sonner is used, or alert

interface DMContextType {
    conversations: Conversation[];
    activeConversation: Conversation | null;
    activeConversationId: string | null;
    setActiveConversationId: (id: string | null) => void;
    messages: Message[];
    isLoadingConversations: boolean;
    isLoadingMessages: boolean;
    sendMessage: (content: string, type?: 'text' | 'image' | 'portfolio', replyToId?: string) => Promise<void>;
    markAsRead: (conversationId: string) => Promise<void>;
    isTyping: Record<string, boolean>; // userId -> isTyping
    sendTyping: (isTyping: boolean) => void;
    onlineUsers: Set<string>;
    totalUnreadCount: number;
    refreshConversations: () => Promise<void>;
    startConversation: (userId: string) => Promise<string>;
}

const DMContext = createContext<DMContextType | null>(null);

export function DMProvider({ children }: { children: ReactNode }) {
    const { accessToken, user } = useAuthStore();
    const [conversations, setConversations] = useState<Conversation[]>([]);
    const [activeConversationId, setActiveConversationId] = useState<string | null>(null);
    const [messages, setMessages] = useState<Message[]>([]);
    const [isLoadingConversations, setIsLoadingConversations] = useState(false);
    const [isLoadingMessages, setIsLoadingMessages] = useState(false);
    const [onlineUsers, setOnlineUsers] = useState<Set<string>>(new Set());
    const [isTyping, setIsTyping] = useState<Record<string, boolean>>({});

    const wsRef = useRef<WebSocket | null>(null);
    const typingTimeoutRef = useRef<Record<string, NodeJS.Timeout>>({});

    // Calculate total unread
    const totalUnreadCount = conversations.reduce((acc, conv) => acc + conv.unread_count, 0);

    const activeConversation = conversations.find(c => c.id === activeConversationId) || null;

    // Initial fetch
    const refreshConversations = useCallback(async () => {
        if (!accessToken) return;
        try {
            setIsLoadingConversations(true);
            const res = await dmApi.getConversations();
            if (res.success) {
                setConversations(res.data);
            }
        } catch (error) {
            console.error('Failed to fetch conversations:', error);
        } finally {
            setIsLoadingConversations(false);
        }
    }, [accessToken]);

    // Fetch messages when active conversation changes
    useEffect(() => {
        if (!activeConversationId || !accessToken) {
            setMessages([]);
            return;
        }

        const fetchMessages = async () => {
            try {
                setIsLoadingMessages(true);
                const res = await dmApi.getMessages(activeConversationId);
                if (res.success) {
                    // Reverse if needed, assuming API returns newest first or oldest first?
                    // Usually chat is oldest at top, newest at bottom. 
                    // If API returns paged (newest first), we might need to reverse.
                    // Let's assume API returns standard list. We'll verify later.
                    // For now, sorting by created_at.
                    const sorted = res.data.sort((a: Message, b: Message) =>
                        new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
                    );
                    setMessages(sorted);

                    // Mark as read immediately when opening
                    dmApi.markAsRead(activeConversationId);
                }
            } catch (error) {
                console.error('Failed to fetch messages:', error);
            } finally {
                setIsLoadingMessages(false);
            }
        };

        fetchMessages();
    }, [activeConversationId, accessToken]);

    // WebSocket Connection
    useEffect(() => {
        if (!accessToken) return;

        const wsUrl = process.env.NEXT_PUBLIC_API_URL
            ? process.env.NEXT_PUBLIC_API_URL.replace('http', 'ws') + '/ws/chat'
            : 'ws://localhost:8080/api/v1/ws/chat';

        const ws = new WebSocket(`${wsUrl}?token=${accessToken}`);

        ws.onopen = () => {
            console.log('DM WebSocket Connected');
        };

        ws.onmessage = (event) => {
            try {
                const data: WSEvent = JSON.parse(event.data);
                handleWSEvent(data);
            } catch (err) {
                console.error('WS Parse Error:', err);
            }
        };

        ws.onclose = () => {
            console.log('DM WebSocket Disconnected');
            // Reconnect logic could be added here
        };

        wsRef.current = ws;

        return () => {
            ws.close();
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [accessToken]);

    const handleWSEvent = (event: WSEvent) => {
        switch (event.type) {
            case 'message.new': {
                const payload = event.payload as WSMessageNew;
                const newMsg = payload.message;

                // Update messages if in active conversation
                if (activeConversationId && newMsg.conversation_id === activeConversationId) {
                    setMessages(prev => [...prev, newMsg]);
                    // Mark as read if window is open
                    dmApi.markAsRead(activeConversationId);
                }

                // Update conversation list (unread count, last message)
                setConversations(prev => {
                    const exists = prev.find(c => c.id === newMsg.conversation_id);
                    if (exists) {
                        return prev.map(c => {
                            if (c.id === newMsg.conversation_id) {
                                return {
                                    ...c,
                                    last_message: newMsg,
                                    unread_count: c.id === activeConversationId ? 0 : c.unread_count + 1,
                                    updated_at: newMsg.created_at
                                };
                            }
                            return c;
                        }).sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
                    } else {
                        // New conversation started by someone else? 
                        // We should fetch it or pessimistically add it if we had full payload.
                        // Ideally we re-fetch conversations.
                        refreshConversations();
                        return prev;
                    }
                });
                break;
            }

            case 'typing': {
                const payload = event.payload as WSTypingPayload;
                if (payload.user_id === user?.id) return; // Ignore self

                if (payload.is_typing) {
                    setIsTyping(prev => ({ ...prev, [payload.user_id]: true }));

                    // Clear existing timeout
                    if (typingTimeoutRef.current[payload.user_id]) {
                        clearTimeout(typingTimeoutRef.current[payload.user_id]);
                    }

                    // Auto clear after 3 seconds
                    typingTimeoutRef.current[payload.user_id] = setTimeout(() => {
                        setIsTyping(prev => ({ ...prev, [payload.user_id]: false }));
                    }, 3000);
                } else {
                    setIsTyping(prev => ({ ...prev, [payload.user_id]: false }));
                }
                break;
            }

            case 'presence': {
                const payload = event.payload as WSPresencePayload;
                setOnlineUsers(prev => {
                    const newSet = new Set(prev);
                    if (payload.is_online) {
                        newSet.add(payload.user_id);
                    } else {
                        newSet.delete(payload.user_id);
                    }
                    return newSet;
                });
                break;
            }

            // Handle other events (deleted, reaction, read receipt) as needed
        }
    };

    const sendMessage = async (content: string, type: 'text' | 'image' | 'portfolio' = 'text', replyToId?: string) => {
        if (!activeConversationId) return;

        // Optimistic update could go here
        try {
            await dmApi.sendMessage(activeConversationId, content, type, replyToId);
            // The socket will return the message
        } catch (error) {
            console.error('Send message failed', error);
            toast.error('Failed to send message'); // Requires toast provider
        }
    };

    const sendTyping = (typing: boolean) => {
        if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN || !activeConversationId) return;

        wsRef.current.send(JSON.stringify({
            type: 'typing',
            payload: {
                conversation_id: activeConversationId,
                is_typing: typing
            }
        }));
    };

    const markAsRead = async (conversationId: string) => {
        try {
            await dmApi.markAsRead(conversationId);
            setConversations(prev => prev.map(c =>
                c.id === conversationId ? { ...c, unread_count: 0 } : c
            ));
        } catch (err) {
            console.error(err);
        }
    };

    // Initial load
    useEffect(() => {
        refreshConversations();
    }, [refreshConversations]);

    return (
        <DMContext.Provider value={{
            conversations,
            activeConversation,
            activeConversationId,
            setActiveConversationId,
            messages,
            isLoadingConversations,
            isLoadingMessages,
            sendMessage,
            markAsRead,
            isTyping,
            sendTyping,
            onlineUsers,
            totalUnreadCount,
            refreshConversations,
            startConversation: async (userId: string) => {
                const res = await dmApi.startConversation(userId, '');
                if (res.success) {
                    await refreshConversations(); // Reload list
                    setActiveConversationId(res.data.id);
                    return res.data.id;
                }
                throw new Error('Failed to start conversation');
            }
        }}>
            {children}
        </DMContext.Provider>
    );
}

export function useDM() {
    const context = useContext(DMContext);
    if (!context) {
        throw new Error('useDM must be used within a DMProvider');
    }
    return context;
}
