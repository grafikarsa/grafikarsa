'use client';

import { useState } from 'react';
import Link from 'next/link';
import { Comment } from '@/lib/types';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { formatDistanceToNow } from '@/lib/utils/format';
import { MessageSquare, MoreHorizontal, Trash2 } from 'lucide-react';
import { useAuthStore } from '@/lib/stores/auth-store';
import { CommentInput } from './comment-input';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { commentsApi } from '@/lib/api';
import { toast } from 'sonner';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';

interface CommentItemProps {
    comment: Comment;
    portfolioId: string;
    depth?: number;
}

export function CommentItem({ comment, portfolioId, depth = 0 }: CommentItemProps) {
    const { user } = useAuthStore();
    const [isReplying, setIsReplying] = useState(false);
    const queryClient = useQueryClient();

    const isOwner = user?.id === comment.user.id;
    const isAdmin = user?.role === 'admin';
    const canDelete = isOwner || isAdmin;

    const deleteMutation = useMutation({
        mutationFn: commentsApi.delete,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', portfolioId] });
            toast.success('Komentar dihapus');
        },
        onError: () => toast.error('Gagal menghapus komentar'),
    });

    return (
        <div className={cn('group', depth > 0 && 'ml-4 sm:ml-8 border-l-2 border-muted pl-4')}>
            <div className="flex gap-3 py-3">
                <Link href={`/${comment.user.username}`} className="shrink-0">
                    <Avatar className="h-8 w-8">
                        <AvatarImage src={comment.user.avatar_url || ''} />
                        <AvatarFallback>{comment.user.nama?.[0]?.toUpperCase()}</AvatarFallback>
                    </Avatar>
                </Link>
                <div className="flex-1 space-y-1">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                            <Link href={`/${comment.user.username}`} className="text-sm font-semibold hover:underline">
                                {comment.user.nama}
                            </Link>
                            {comment.user.kelas && (
                                <span className="text-xs text-muted-foreground">
                                    • {comment.user.kelas.nama}
                                </span>
                            )}
                            <span className="text-xs text-muted-foreground">
                                • {formatDistanceToNow(comment.created_at)}
                            </span>
                        </div>
                        {canDelete && (
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                    <Button variant="ghost" size="icon" className="h-6 w-6 opacity-0 group-hover:opacity-100">
                                        <MoreHorizontal className="h-3 w-3" />
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end">
                                    <DropdownMenuItem
                                        className="text-red-600 focus:text-red-600"
                                        onClick={() => deleteMutation.mutate(comment.id)}
                                    >
                                        <Trash2 className="mr-2 h-3.5 w-3.5" />
                                        Hapus
                                    </DropdownMenuItem>
                                </DropdownMenuContent>
                            </DropdownMenu>
                        )}
                    </div>

                    <div className="text-sm text-foreground/90 whitespace-pre-line">
                        {comment.content}
                    </div>

                    <div className="flex items-center gap-4 pt-1">
                        {user && (
                            <Button
                                variant="ghost"
                                size="sm"
                                className="h-auto p-0 text-xs text-muted-foreground hover:text-foreground"
                                onClick={() => setIsReplying(!isReplying)}
                            >
                                <MessageSquare className="mr-1.5 h-3 w-3" />
                                Balas
                            </Button>
                        )}
                    </div>

                    {isReplying && (
                        <div className="pt-3">
                            <CommentInput
                                portfolioId={portfolioId}
                                parentId={comment.id}
                                autoFocus
                                onCancel={() => setIsReplying(false)}
                                onSuccess={() => setIsReplying(false)}
                            />
                        </div>
                    )}
                </div>
            </div>

            {comment.children && comment.children.length > 0 && (
                <div className="mt-2">
                    {comment.children.map((child) => (
                        <CommentItem
                            key={child.id}
                            comment={child}
                            portfolioId={portfolioId}
                            depth={depth + 1}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}
