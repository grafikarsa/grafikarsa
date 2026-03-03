'use client';

import { useQuery } from '@tanstack/react-query';
import { commentsApi } from '@/lib/api';
import { CommentInput } from './comment-input';
import { CommentItem } from './comment-item';
import { Skeleton } from '@/components/ui/skeleton';
import { useAuthStore } from '@/lib/stores/auth-store';
import { MessageSquare } from 'lucide-react';

interface CommentSectionProps {
    portfolioId: string;
}

export function CommentSection({ portfolioId }: CommentSectionProps) {
    const { isAuthenticated } = useAuthStore();
    const { data, isLoading } = useQuery({
        queryKey: ['comments', portfolioId],
        queryFn: () => commentsApi.getByPortfolioID(portfolioId),
    });

    const comments = data?.data?.data || [];
    const commentCount = comments.length + comments.reduce((acc, c) => acc + (c.children?.length || 0), 0); // Warning: this only counts 2 levels if not recursive count. Ideally Backend sends count.

    // Helper to count recursively
    const countComments = (items: any[]): number => {
        return items.reduce((acc, item) => acc + 1 + (item.children ? countComments(item.children) : 0), 0);
    };
    const totalComments = countComments(comments);

    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-20 w-full" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center gap-2">
                <h3 className="text-lg font-semibold">Komentar</h3>
                <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium">
                    {totalComments}
                </span>
            </div>

            {isAuthenticated ? (
                <CommentInput portfolioId={portfolioId} />
            ) : (
                <div className="rounded-lg border bg-muted/50 p-4 text-center text-sm text-muted-foreground">
                    <MessageSquare className="mx-auto mb-2 h-6 w-6 opacity-50" />
                    Silakan <a href="/login" className="font-medium underline hover:text-foreground">login</a> untuk memberikan komentar.
                </div>
            )}

            <div className="divide-y">
                {comments.length === 0 ? (
                    <div className="py-8 text-center text-sm text-muted-foreground">
                        Belum ada komentar. Jadilah yang pertama berkomentar!
                    </div>
                ) : (
                    comments.map((comment) => (
                        <CommentItem key={comment.id} comment={comment} portfolioId={portfolioId} />
                    ))
                )}
            </div>
        </div>
    );
}
