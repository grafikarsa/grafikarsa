'use client';

import { useDM } from '@/components/providers/dm-provider';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { formatDistanceToNow } from 'date-fns';
import { id } from 'date-fns/locale';
import { useAuthStore } from '@/lib/stores/auth-store';

export function ConversationList() {
    const {
        conversations,
        activeConversationId,
        setActiveConversationId,
        onlineUsers
    } = useDM();
    const { user } = useAuthStore();

    if (conversations.length === 0) {
        return (
            <div className="h-full flex flex-col items-center justify-center p-6 text-center text-muted-foreground gap-2">
                <p className="font-medium">Direct Messages</p>
                <p className="text-xs">Send private photos and messages to a friend or group.</p>
            </div>
        );
    }

    return (
        <ScrollArea className="h-full">
            <div className="flex flex-col py-2">
                {conversations.map((conv) => {
                    // Determine other participant
                    const otherUser = conv.participants.find(p => p.id !== user?.id) || conv.participants[0];
                    const isContextUserOnline = onlineUsers.has(otherUser.id);
                    const isActive = activeConversationId === conv.id;
                    const isUnread = conv.unread_count > 0;

                    return (
                        <button
                            key={conv.id}
                            onClick={() => setActiveConversationId(conv.id)}
                            className={cn(
                                "flex items-center gap-3 px-5 py-3 text-left transition-colors hover:bg-muted/30 w-full",
                                isActive && "bg-muted/50"
                            )}
                        >
                            <div className="relative flex-shrink-0">
                                <Avatar className="h-12 w-12 border border-border/40">
                                    <AvatarImage src={otherUser.avatar_url} />
                                    <AvatarFallback>{otherUser.nama[0]}</AvatarFallback>
                                </Avatar>
                                {isContextUserOnline && (
                                    <span className="absolute bottom-0 right-0 h-3.5 w-3.5 rounded-full border-2 border-background bg-green-500" />
                                )}
                            </div>

                            <div className="flex-1 min-w-0">
                                <div className="flex items-center justify-between mb-0.5">
                                    <span className={cn("text-sm truncate", isUnread ? "font-bold text-foreground" : "font-medium text-foreground")}>
                                        {otherUser.nama}
                                    </span>
                                </div>
                                <div className="flex items-center gap-1.5 text-xs">
                                    <p className={cn("truncate max-w-[160px]", isUnread ? "text-foreground font-semibold" : "text-muted-foreground")}>
                                        {/* Prefix if YOU sent last message */}
                                        {conv.last_message?.sender_id === user?.id && "You: "}
                                        {conv.last_message?.content || 'Start a conversation'}
                                    </p>
                                    {conv.last_message && (
                                        <>
                                            <span className="text-muted-foreground">•</span>
                                            <span className="text-muted-foreground whitespace-nowrap">
                                                {formatDistanceToNow(new Date(conv.last_message.created_at), { addSuffix: false, locale: id })
                                                    .replace('yang lalu', '')
                                                    .replace('sekitar ', '')
                                                    .replace('jam', 'h')
                                                    .replace('menit', 'm')}
                                            </span>
                                        </>
                                    )}
                                </div>
                            </div>

                            {isUnread && (
                                <div className="flex-shrink-0">
                                    <span className="h-2.5 w-2.5 rounded-full bg-blue-500 block" />
                                </div>
                            )}
                        </button>
                    );
                })}
            </div>
        </ScrollArea>
    );
}

