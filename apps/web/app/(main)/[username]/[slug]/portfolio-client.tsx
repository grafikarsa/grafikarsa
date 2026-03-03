'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { notFound } from 'next/navigation';

import { portfoliosApi } from '@/lib/api';
import { useAuthStore } from '@/lib/stores/auth-store';
import { BlockRenderer } from '@/components/portfolio/block-renderer';
import { CommentSection } from '@/components/portfolio/comments/comment-section';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Separator } from '@/components/ui/separator';

import { calculateReadingTime } from '@/lib/utils/reading-time';
import { ImageWithLightbox } from '@/components/ui/lightbox';
import { RelatedPortfolios } from '@/components/portfolio/related-portfolios';
import { Calendar, Edit, ArrowLeft, Clock, AlertCircle, Eye, BookOpen } from 'lucide-react';
import { PortfolioActions } from '@/components/portfolio/portfolio-actions';

import { formatDate, formatDistanceToNow } from '@/lib/utils/format';
import { Portfolio } from '@/lib/types';

interface PortfolioClientProps {
    username: string;
    slug: string;
    initialData: Portfolio | null;
}

function PortfolioSkeleton() {
    return (
        <div className="space-y-8">
            <Skeleton className="aspect-[21/9] w-full" />
            <div className="mx-auto max-w-3xl space-y-6 px-4">
                <Skeleton className="h-12 w-3/4" />
                <div className="flex gap-2">
                    <Skeleton className="h-6 w-20" />
                    <Skeleton className="h-6 w-20" />
                </div>
                <div className="flex items-center gap-3">
                    <Skeleton className="h-12 w-12 rounded-full" />
                    <div className="space-y-2">
                        <Skeleton className="h-4 w-32" />
                        <Skeleton className="h-3 w-24" />
                    </div>
                </div>
                <Skeleton className="h-40 w-full" />
            </div>
        </div>
    );
}

export function PortfolioClient({ username, slug, initialData }: PortfolioClientProps) {
    const router = useRouter();
    const { user: currentUser } = useAuthStore();
    const isOwner = currentUser?.username === username;

    const { data, isLoading } = useQuery({
        queryKey: ['portfolio', username, slug],
        queryFn: () => portfoliosApi.getPortfolioBySlug(slug, username),
        initialData: initialData ? { data: initialData, message: '', success: true } : undefined,
    });

    const portfolio = (data as any)?.data || initialData;

    if (isLoading && !portfolio) {
        return <PortfolioSkeleton />;
    }

    if (!portfolio) {
        notFound();
    }

    const readingTime = calculateReadingTime(portfolio.content_blocks || []);

    return (
        <article className="pb-24 md:pb-16">
            <div className="mx-auto max-w-3xl px-4">
                {/* Top Navigation */}
                <div className="mb-6 mt-4 flex items-center justify-between">
                    <button
                        onClick={() => router.back()}
                        className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
                    >
                        <ArrowLeft className="h-4 w-4" />
                        Kembali
                    </button>
                    {isOwner && (
                        <Link href={`/${username}/${slug}/edit`}>
                            <Button variant="outline" size="sm">
                                <Edit className="mr-2 h-4 w-4" />
                                Edit
                            </Button>
                        </Link>
                    )}
                </div>

                {/* Header Section */}
                <header className="mb-8">
                    {isOwner && portfolio.status !== 'published' && (
                        <Badge
                            variant={portfolio.status === 'rejected' ? 'destructive' : 'secondary'}
                            className="mb-3"
                        >
                            {portfolio.status === 'pending_review' ? 'Menunggu Review' :
                                portfolio.status === 'draft' ? 'Draft' : portfolio.status}
                        </Badge>
                    )}

                    <h1 className="mb-4 text-2xl font-bold leading-tight sm:text-3xl md:text-4xl">
                        {portfolio.judul}
                    </h1>

                    <div className="mb-6 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
                        {/* Author */}
                        {portfolio.user && (
                            <Link
                                href={`/${portfolio.user.username}`}
                                className="flex items-center gap-2 transition-colors hover:text-foreground"
                            >
                                <Avatar className="h-6 w-6 ring-1 ring-border">
                                    <AvatarImage src={portfolio.user.avatar_url} alt={portfolio.user.nama} />
                                    <AvatarFallback>{portfolio.user.nama?.charAt(0)}</AvatarFallback>
                                </Avatar>
                                <span className="font-medium text-foreground">{portfolio.user.nama}</span>
                            </Link>
                        )}

                        <span className="text-muted-foreground/50">•</span>

                        {/* Date */}
                        <span className="flex items-center gap-1.5">
                            {formatDate(portfolio.published_at || portfolio.created_at)}
                        </span>

                        {/* Reading Time */}
                        <span className="text-muted-foreground/50">•</span>
                        <span className="flex items-center gap-1.5" title="Estimasi waktu baca">
                            <BookOpen className="h-3.5 w-3.5" />
                            {readingTime} min read
                        </span>

                        {/* View Count (if available) */}
                        {portfolio.view_count !== undefined && (
                            <>
                                <span className="text-muted-foreground/50">•</span>
                                <span className="flex items-center gap-1.5" title={`${portfolio.view_count} kali dilihat`}>
                                    <Eye className="h-3.5 w-3.5" />
                                    {portfolio.view_count}
                                </span>
                            </>
                        )}

                        {portfolio.updated_at && portfolio.updated_at !== portfolio.created_at && (
                            <>
                                <span className="text-muted-foreground/50">•</span>
                                <span className="flex items-center gap-1.5" title={`Diperbarui ${formatDistanceToNow(portfolio.updated_at)}`}>
                                    <Clock className="h-3.5 w-3.5" />
                                    Updated
                                </span>
                            </>
                        )}
                    </div>

                    {/* Thumbnail with Lightbox */}
                    {portfolio.thumbnail_url && (
                        <div className="relative mb-6 aspect-video w-full overflow-hidden rounded-lg bg-muted">
                            <ImageWithLightbox
                                src={portfolio.thumbnail_url}
                                alt={portfolio.judul}
                                className="object-contain"
                            />
                        </div>
                    )}

                    {/* Tags */}
                    {portfolio.tags && portfolio.tags.length > 0 && (
                        <div className="flex flex-wrap gap-2">
                            {portfolio.tags.map((tag: any) => (
                                <Badge key={tag.id} variant="secondary" className="font-normal">
                                    {tag.nama}
                                </Badge>
                            ))}

                            {portfolio.series && (
                                <Badge className="bg-blue-500 font-normal text-white hover:bg-blue-600">
                                    {portfolio.series.nama}
                                </Badge>
                            )}
                        </div>
                    )}
                </header>

                {/* Admin Note */}
                {isOwner && portfolio.admin_review_note && (
                    <div className="mb-8 flex gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4">
                        <AlertCircle className="h-5 w-5 shrink-0 text-amber-500" />
                        <div>
                            <p className="text-sm font-medium text-amber-700 dark:text-amber-400">Catatan dari Admin</p>
                            <p className="mt-1 text-sm text-amber-600 dark:text-amber-300">{portfolio.admin_review_note}</p>
                        </div>
                    </div>
                )}

                <Separator className="mb-8 mt-4" />

                {/* Content Blocks */}
                {portfolio.content_blocks && portfolio.content_blocks.length > 0 ? (
                    <div className="prose prose-neutral dark:prose-invert max-w-none">
                        <BlockRenderer blocks={portfolio.content_blocks} />
                    </div>
                ) : (
                    <div className="py-12 text-center text-muted-foreground">
                        <p>Belum ada konten</p>
                    </div>
                )}

                {/* Author Card (Bottom) */}
                {portfolio.user && (
                    <div className="mt-16 rounded-xl border bg-card p-6">
                        <div className="mb-4 text-sm font-medium text-muted-foreground">Tentang Penulis</div>
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
                            <Link href={`/${portfolio.user.username}`}>
                                <Avatar className="h-16 w-16 border">
                                    <AvatarImage src={portfolio.user.avatar_url} alt={portfolio.user.nama} />
                                    <AvatarFallback className="text-xl">{portfolio.user.nama?.charAt(0)}</AvatarFallback>
                                </Avatar>
                            </Link>
                            <div className="flex-1">
                                <Link href={`/${portfolio.user.username}`} className="hover:underline">
                                    <h3 className="text-lg font-semibold">{portfolio.user.nama}</h3>
                                </Link>
                                <div className="flex gap-2 text-sm text-muted-foreground">
                                    <span>@{portfolio.user.username}</span>
                                    {portfolio.user.kelas_nama && <span>• {portfolio.user.kelas_nama}</span>}
                                </div>
                            </div>
                            <Link href={`/${portfolio.user.username}`}>
                                <Button variant="outline">Lihat Profil</Button>
                            </Link>
                        </div>
                    </div>
                )}

                {/* Related Portfolios */}
                {portfolio.user && (
                    <RelatedPortfolios
                        userId={portfolio.user.id}
                        currentPortfolioId={portfolio.id}
                        username={portfolio.user.username}
                    />
                )}

                <Separator className="my-12" />

                {/* Comments Section */}
                <section id="comments">
                    <CommentSection portfolioId={portfolio.id} />
                </section>
            </div>

            <PortfolioActions
                portfolio={{
                    id: portfolio.id,
                    judul: portfolio.judul,
                    is_liked: portfolio.is_liked ?? false,
                    like_count: portfolio.like_count ?? 0,
                    username: portfolio.user?.username ?? '',
                    slug: portfolio.slug
                }}
            />
        </article>
    );
}
