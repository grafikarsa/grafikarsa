'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Heart, Share2, MessageCircle } from 'lucide-react';
import { Button } from '@/components/ui/button'; // Assuming you have a Button component
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

    // Optimistic UI state could be added here, but relying on RQ cache invalidation for now

    const likeMutation = useMutation({
        mutationFn: () =>
            portfolio.is_liked
                ? portfoliosApi.unlikePortfolio(portfolio.id)
                : portfoliosApi.likePortfolio(portfolio.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['portfolio', portfolio.username, portfolio.slug] });
        },
        onError: () => {
            toast.error('Gagal. Silakan coba lagi.');
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
                        className={cn("gap-2 px-2 hover:bg-transparent", portfolio.is_liked && "text-red-500")}
                        onClick={() => isAuthenticated ? likeMutation.mutate() : toast.error('Silakan login untuk menyukai')}
                    >
                        <Heart className={cn("h-5 w-5", portfolio.is_liked && "fill-current")} />
                        <span className="text-base font-semibold">{portfolio.like_count || 0}</span>
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
                                variant={portfolio.is_liked ? "default" : "secondary"}
                                size="icon"
                                className={cn(
                                    "h-12 w-12 rounded-full shadow-md transition-transform hover:scale-105",
                                    portfolio.is_liked && "bg-red-500 hover:bg-red-600 text-white border-none"
                                )}
                                onClick={() => isAuthenticated ? likeMutation.mutate() : toast.error('Silakan login untuk menyukai')}
                            >
                                <Heart className={cn("h-5 w-5", portfolio.is_liked && "fill-current")} />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent side="left">
                            <p>{portfolio.like_count} Suka</p>
                        </TooltipContent>
                    </Tooltip>
                </TooltipProvider>
            </div>
        </>
    );
}
