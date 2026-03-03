'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import Image from 'next/image';
import { portfoliosApi } from '@/lib/api';
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Eye, Heart } from 'lucide-react';
import { formatDate } from '@/lib/utils/format';

interface RelatedPortfoliosProps {
    userId: string;
    currentPortfolioId: string; // To exclude current one
    username: string; // For linking
}

export function RelatedPortfolios({ userId, currentPortfolioId, username }: RelatedPortfoliosProps) {
    const { data, isLoading } = useQuery({
        queryKey: ['portfolios', 'related', userId],
        queryFn: () => portfoliosApi.getPortfolios({ user_id: userId, limit: 4 }), // Fetch 4 to have buffer if current is included
    });

    const portfolios = data?.data?.filter(p => p.id !== currentPortfolioId).slice(0, 3) || [];

    if (isLoading) {
        return <RelatedSkeleton />;
    }

    if (portfolios.length === 0) {
        return null;
    }

    return (
        <div className="mt-12">
            <div className="mb-6 flex items-center justify-between">
                <h3 className="text-xl font-bold">Karya Lainnya dari @{username}</h3>
                <Link href={`/${username}`} className="text-sm font-medium text-primary hover:underline">
                    Lihat Semua
                </Link>
            </div>

            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3">
                {portfolios.map((portfolio) => (
                    <Link key={portfolio.id} href={`/${portfolio.user?.username}/${portfolio.slug}`}>
                        <Card className="group h-full overflow-hidden transition-all hover:shadow-md">
                            <div className="aspect-video relative overflow-hidden bg-muted">
                                {portfolio.thumbnail_url ? (
                                    <Image
                                        src={portfolio.thumbnail_url}
                                        alt={portfolio.judul}
                                        fill
                                        className="object-cover transition-transform duration-300 group-hover:scale-105"
                                    />
                                ) : (
                                    <div className="flex h-full w-full items-center justify-center bg-gray-100 text-gray-400">
                                        No Image
                                    </div>
                                )}
                            </div>
                            <CardContent className="p-4">
                                <h4 className="line-clamp-1 font-semibold group-hover:text-primary">{portfolio.judul}</h4>
                                <p className="mt-1 text-xs text-muted-foreground">{formatDate(portfolio.published_at || portfolio.created_at)}</p>
                            </CardContent>
                            <CardFooter className="flex items-center justify-between border-t p-4 text-xs text-muted-foreground">
                                <div className="flex items-center gap-1">
                                    <Heart className="h-3 w-3" />
                                    <span>{portfolio.like_count}</span>
                                </div>
                                {/* Only show view count if available */}
                                {portfolio.view_count !== undefined && (
                                    <div className="flex items-center gap-1">
                                        <Eye className="h-3 w-3" />
                                        <span>{portfolio.view_count}</span>
                                    </div>
                                )}
                            </CardFooter>
                        </Card>
                    </Link>
                ))}
            </div>
        </div>
    );
}

function RelatedSkeleton() {
    return (
        <div className="mt-12 space-y-6">
            <Skeleton className="h-8 w-48" />
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3">
                {[1, 2, 3].map((i) => (
                    <div key={i} className="space-y-3">
                        <Skeleton className="aspect-video w-full rounded-lg" />
                        <Skeleton className="h-4 w-3/4" />
                        <Skeleton className="h-3 w-1/2" />
                    </div>
                ))}
            </div>
        </div>
    )
}
