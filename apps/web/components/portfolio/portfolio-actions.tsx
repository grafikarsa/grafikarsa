'use client';

import { useState, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Share2, MessageCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { AnimatedHeart } from '@/components/ui/animated-heart';
import { portfoliosApi } from '@/lib/api';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/lib/stores/auth-store';
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip';

interface PortfolioActionsProps {
    portfolio: {
        id: string;
        judul: string;
        is_liked: boolean;
        like_count: number;
        username: string;
        slug: string;
    };
}

export function PortfolioActions({ portfolio }: PortfolioActionsProps) {
    const queryClient = useQueryClient();
    const { isAuthenticated } = useAuthStore();
    
    // Local state for optimistic UI
    const [isLiked, setIsLiked] = useState(portfolio.is_liked);
    const [likeCount, setLikeCount] = useState(portfolio.like_count);

    // Sync with props when portfolio data changes
    useEffect(() => {
        setIsLiked(portfolio.is_liked);
        setLikeCount(portfolio.like_count);
    }, [portfolio.is_liked, portfolio.like_count]);

    const likeMutation = useMutation({
        mutationFn: () =>
            isLiked
                ? portfoliosApi.unlikePortfolio(portfolio.id)
                : portfoliosApi.likePortfolio(portfolio.id),
        onMutate: async () => {
            // Cancel outgoing refetches
            await queryClient.cancelQueries({ queryKey: ['portfolio', portfolio.username, portfolio.slug] });

            // Snapshot previous value
            const previousIsLiked = isLiked;
            const previousLikeCount = likeCount;

            // Optimistically update local state
            setIsLiked(!isLiked);
            setLikeCount(isLiked ? likeCount - 1 : likeCount + 1);

            return { previousIsLiked, previousLikeCount };
        },
        onError: (_err, _vars, context) => {
            // Rollback on error
            if (context) {
                setIsLiked(context.previousIsLiked);
                setLikeCount(context.previousLikeCount);
            }
            toast.error('Gagal. Silakan coba lagi.');
        },
        onSuccess: (data) => {
            // Update with server response
            if (data.data) {
                setIsLiked(data.data.is_liked);
                setLikeCount(data.data.like_count);
            }
            // Invalidate both portfolio detail and feed caches
            queryClient.invalidateQueries({ queryKey: ['portfolio', portfolio.username, portfolio.slug] });
            queryClient.invalidateQueries({ queryKey: ['feed'] });
        },
    });

    const handleShare = async () => {
        const url = window.location.href;
        if (navigator.share) {
            try {
                await navigator.share({ title: portfolio.judul, url });
            } catch (err) {
                // Ignore abort errors
            }
        } else {
            await navigator.clipboard.writeText(url);
            toast.success('Link berhasil disalin!');
        }
    };

    const scrollToComments = () => {
        document.getElementById('comments')?.scrollIntoView({ behavior: 'smooth' });
    };

    return (
        <>
            {/* Mobile Bottom Bar */}
            <div className="fixed bottom-0 left-0 right-0 z-40 flex h-16 items-center justify-between border-t bg-background/80 px-4 backdrop-blur-md md:hidden">
                <div className="flex items-center gap-4">
                    {/* Like Button */}
                    <Button
                        variant="ghost"
                        size="sm"
                        className={cn("gap-2 px-2 hover:bg-transparent", isLiked && "text-red-500")}
                        onClick={() => isAuthenticated ? likeMutation.mutate() : toast.error('Silakan login untuk menyukai')}
                        disabled={likeMutation.isPending}
                    >
                        <AnimatedHeart isLiked={isLiked} size={20} />
                        <span className="text-base font-semibold">{likeCount || 0}</span>
                    </Button>

                    {/* Comment Button */}
                    <Button
                        variant="ghost"
                        size="sm"
                        className="gap-2 px-2 hover:bg-transparent"
                        onClick={scrollToComments}
                    >
                        <MessageCircle className="h-5 w-5" />
                    </Button>
                </div>

                {/* Share Button */}
                <Button variant="ghost" size="icon" onClick={handleShare}>
                    <Share2 className="h-5 w-5" />
                </Button>
            </div>

            {/* Desktop Floating Stack (Above Feedback Button) */}
            <div className="fixed bottom-24 right-6 z-40 hidden flex-col gap-3 md:flex">
                <TooltipProvider delayDuration={100}>
                    {/* Share */}
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant="secondary"
                                size="icon"
                                className="h-12 w-12 rounded-full shadow-md transition-transform hover:scale-105"
                                onClick={handleShare}
                            >
                                <Share2 className="h-5 w-5" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent side="left">
                            <p>Bagikan</p>
                        </TooltipContent>
                    </Tooltip>

                    {/* Comments */}
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant="secondary"
                                size="icon"
                                className="h-12 w-12 rounded-full shadow-md transition-transform hover:scale-105"
                                onClick={scrollToComments}
                            >
                                <MessageCircle className="h-5 w-5" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent side="left">
                            <p>Komentar</p>
                        </TooltipContent>
                    </Tooltip>

                    {/* Like */}
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant={isLiked ? "default" : "secondary"}
                                size="icon"
                                className={cn(
                                    "h-12 w-12 rounded-full shadow-md transition-transform hover:scale-105",
                                    isLiked && "bg-red-500 hover:bg-red-600 text-white border-none"
                                )}
                                onClick={() => isAuthenticated ? likeMutation.mutate() : toast.error('Silakan login untuk menyukai')}
                                disabled={likeMutation.isPending}
                            >
                                <AnimatedHeart isLiked={isLiked} size={20} />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent side="left">
                            <p>{likeCount} Suka</p>
                        </TooltipContent>
                    </Tooltip>
                </TooltipProvider>
            </div>
        </>
    );
}
