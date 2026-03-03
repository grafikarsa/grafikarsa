'use client';

import { useDM } from '@/components/providers/dm-provider';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useAuthStore } from '@/lib/stores/auth-store';
import { cn } from '@/lib/utils';
import { format } from 'date-fns';
import { Send, Image as ImageIcon, Info, MessageCircle, Phone, Video } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip';

interface ChatWindowProps {
    onOpenInfo?: () => void;
}

export function ChatWindow({ onOpenInfo }: ChatWindowProps) {
    const {
        activeConversation,
        messages,
        sendMessage,
        sendTyping,
        isTyping,
        onlineUsers,
        isLoadingMessages
    } = useDM();
    const { user } = useAuthStore();
    const [inputValue, setInputValue] = useState('');
    const scrollRef = useRef<HTMLDivElement>(null);
    const typingTimeoutRef = useRef<NodeJS.Timeout>(undefined);

    const otherUser = activeConversation?.participants.find(p => p.id !== user?.id);
    const isOnline = otherUser ? onlineUsers.has(otherUser.id) : false;
    const isPartnerTyping = otherUser ? isTyping[otherUser.id] : false;

    // Auto scroll to bottom
    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [messages, isPartnerTyping, activeConversation?.id]); // Added activeConversation.id dependency

    const handleSend = async (e?: React.FormEvent) => {
        e?.preventDefault();
        if (!inputValue.trim()) return;

        await sendMessage(inputValue);
        setInputValue('');
        sendTyping(false);
    };

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setInputValue(e.target.value);

        sendTyping(true);
        if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
        typingTimeoutRef.current = setTimeout(() => sendTyping(false), 3000);
    };

    if (!activeConversation) {
        return (
            <div className="flex h-full flex-col items-center justify-center text-center p-8 text-muted-foreground gap-4">
                <div className="h-24 w-24 rounded-full bg-muted flex items-center justify-center">
                    <MessageCircle className="h-12 w-12 text-muted-foreground/50" />
                </div>
                <div>
                    <h3 className="text-xl font-semibold mb-1">Your Messages</h3>
                    <p className="text-sm">Send private photos and messages to a friend or group.</p>
                </div>
            </div>
        );
    }

    return (
        <div className="flex h-full flex-col">
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b h-[72px] bg-background/50 backdrop-blur-sm z-10">
                <div className="flex items-center gap-4">
                    <Avatar className="h-10 w-10 cursor-pointer">
                        <AvatarImage src={otherUser?.avatar_url} />
                        <AvatarFallback>{otherUser?.nama[0]}</AvatarFallback>
                    </Avatar>
                    <div>
                        <h3 className="font-semibold text-sm leading-none mb-1">{otherUser?.nama}</h3>
                        <p className="text-xs text-muted-foreground">
                            {isOnline ? 'Active now' : 'Offline'}
                        </p>
                    </div>
                </div>
                <div className="flex gap-2">
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground">
                                    <Phone className="h-6 w-6" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>Coming Soon</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground">
                                    <Video className="h-6 w-6" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>Coming Soon</TooltipContent>
                        </Tooltip>
                        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground" onClick={onOpenInfo}>
                            <Info className="h-6 w-6" />
                        </Button>
                    </TooltipProvider>
                </div>
            </div>

            {/* Messages */}
            <ScrollArea className="flex-1 px-4 py-2">
                <div className="flex flex-col gap-1.5 py-4">
                    {/* Timestamp for conversation start could go here */}

                    {messages.map((msg, idx) => {
                        const isMe = msg.sender_id === user?.id;
                        const isLastInGroup = idx === messages.length - 1 || messages[idx + 1].sender_id !== msg.sender_id;
                        const isFirstInGroup = idx === 0 || messages[idx - 1].sender_id !== msg.sender_id;

                        // Bubble Shapes logic
                        const myRadius = isLastInGroup
                            ? isFirstInGroup ? "rounded-[22px]" : "rounded-[22px] rounded-br-sm"
                            : isFirstInGroup ? "rounded-[22px] rounded-tr-sm" : "rounded-[22px] rounded-r-sm";

                        const otherRadius = isLastInGroup
                            ? isFirstInGroup ? "rounded-[22px]" : "rounded-[22px] rounded-bl-sm"
                            : isFirstInGroup ? "rounded-[22px] rounded-tl-sm" : "rounded-[22px] rounded-l-sm";

                        return (
                            <div
                                key={msg.id}
                                className={cn(
                                    "flex gap-2 max-w-[65%]",
                                    isMe ? "ml-auto flex-row-reverse" : "items-end"
                                )}
                            >
                                <div className={cn(
                                    "px-4 py-2.5 break-words text-[15px] leading-snug",
                                    isMe
                                        ? `bg-[#0095F6] text-white ${myRadius}`
                                        : `bg-muted dark:bg-[#262626] text-foreground ${otherRadius}`
                                )}>
                                    {msg.message_type === 'image' ? (
                                        // eslint-disable-next-line @next/next/no-img-element
                                        <img src={msg.content} alt="Image" className="rounded-lg max-w-full" />
                                    ) : (
                                        <p>{msg.content}</p>
                                    )}
                                </div>
                                {isLastInGroup && !isMe && (
                                    <span className="text-[10px] text-muted-foreground self-center opacity-0 group-hover:opacity-100 transition-opacity">
                                        {format(new Date(msg.created_at), 'HH:mm')}
                                    </span>
                                )}
                            </div>
                        );
                    })}

                    {isPartnerTyping && (
                        <div className="flex gap-2 text-sm text-muted-foreground items-center ml-2 mt-2">
                            <Avatar className="h-6 w-6">
                                <AvatarImage src={otherUser?.avatar_url} />
                            </Avatar>
                            <span className="bg-muted px-3 py-1.5 rounded-full text-xs animate-pulse">
                                ...
                            </span>
                        </div>
                    )}
                    <div ref={scrollRef} className="h-px" />
                </div>
            </ScrollArea>

            {/* Input Area */}
            <div className="p-4 m-4 mt-0 bg-background">
                <form onSubmit={handleSend} className="flex items-center gap-2 p-1.5 rounded-[22px] border bg-background focus-within:ring-1 focus-within:ring-muted-foreground/20 transition-all">
                    <Button type="button" variant="ghost" size="icon" className="h-9 w-9 rounded-full text-muted-foreground hover:text-foreground">
                        <ImageIcon className="h-5 w-5" />
                    </Button>
                    <Input
                        value={inputValue}
                        onChange={handleInputChange}
                        placeholder="Message..."
                        className="flex-1 border-none shadow-none focus-visible:ring-0 px-2 py-6 h-auto max-h-32 text-base"
                    />
                    {inputValue.trim() && (
                        <Button type="submit" variant="ghost" size="sm" className="font-semibold text-primary hover:text-primary/80 hover:bg-transparent px-3 mr-1">
                            Send
                        </Button>
                    )}
                </form>
            </div>
        </div>
    );
}

