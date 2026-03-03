'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { usersApi } from '@/lib/api';
import { useDM } from '@/components/providers/dm-provider';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Search, Loader2 } from 'lucide-react';
import { useDebounce } from '@/lib/hooks/use-debounce';

import { toast } from 'sonner';

interface NewMessageModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export function NewMessageModal({ open, onOpenChange }: NewMessageModalProps) {
    const router = useRouter();
    const { startConversation } = useDM();
    const [searchQuery, setSearchQuery] = useState('');
    const debouncedSearch = useDebounce(searchQuery, 500);

    const { data: users, isLoading } = useQuery({
        queryKey: ['users-search', debouncedSearch],
        queryFn: () => usersApi.search(debouncedSearch),
        enabled: debouncedSearch.length > 0,
    });

    const handleUserSelect = async (userId: string) => {
        try {
            await startConversation(userId);
            onOpenChange(false);
            toast.success('Percakapan dimulai');
        } catch (error) {
            console.error('Failed to start conversation:', error);
            toast.error('Gagal memulai percakapan');
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[400px] p-0 gap-0 overflow-hidden">
                <DialogHeader className="p-4 border-b">
                    <DialogTitle className="text-center font-bold">New Message</DialogTitle>
                    <DialogDescription className="sr-only">
                        Search for people to start a new private message.
                    </DialogDescription>
                </DialogHeader>

                <div className="p-2 border-b">
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <Input
                            placeholder="Search people..."
                            className="pl-9 h-10 border-none focus-visible:ring-0 bg-transparent"
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            autoFocus
                        />
                    </div>
                </div>

                <ScrollArea className="h-[300px] p-2">
                    {isLoading ? (
                        <div className="flex h-full items-center justify-center">
                            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                        </div>
                    ) : debouncedSearch && users?.data?.length === 0 ? (
                        <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
                            No users found.
                        </div>
                    ) : (
                        <div className="space-y-1">
                            {users?.data?.map((user: import('@/lib/types').UserCard) => (
                                <button
                                    key={user.id}
                                    onClick={() => handleUserSelect(user.id)}
                                    className="w-full flex items-center gap-3 p-3 rounded-xl hover:bg-muted/80 transition-all text-left cursor-pointer active:scale-[0.98]"
                                >
                                    <Avatar className="h-11 w-11 border">
                                        <AvatarImage src={user.avatar_url} />
                                        <AvatarFallback>{user.nama[0]}</AvatarFallback>
                                    </Avatar>
                                    <div className="flex-1 overflow-hidden">
                                        <p className="font-semibold text-sm truncate">{user.nama}</p>
                                        <p className="text-xs text-muted-foreground truncate">@{user.username}</p>
                                    </div>
                                </button>
                            ))}
                            {!debouncedSearch && (
                                <div className="p-12 text-center">
                                    <p className="text-sm text-muted-foreground">Cari pengguna untuk memulai pesan baru.</p>
                                </div>
                            )}
                        </div>
                    )}
                </ScrollArea>
            </DialogContent>
        </Dialog>
    );
}
