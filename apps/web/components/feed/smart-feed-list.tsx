'use client';

import { useEffect, useCallback, useRef, useState } from 'react';
import { useInfiniteQuery, useQuery, useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { RefreshCw, Loader2, Sparkles, Clock, Users, Heart, UserPlus, Search } from 'lucide-react';
import { feedApi } from '@/lib/api';
import { FeedAlgorithm, FeedItem } from '@/lib/types';
import { useAuthStore } from '@/lib/stores/auth-store';
import { FeedAlgorithmSwitcher } from './feed-algorithm-switcher';
import { TimelineFeedItem } from './timeline-feed-item';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';

const ITEMS_PER_PAGE = 20;

function FeedSkeleton() {
  return (
    <div className="space-y-4">
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="p-4 border-b">
          <div className="flex items-start gap-3">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-3 w-24" />
            </div>
          </div>
          <div className="mt-3 space-y-2">
            <Skeleton className="h-5 w-3/4" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-2/3" />
          </div>
          <Skeleton className="mt-3 aspect-video w-full rounded-xl" />
        </div>
      ))}
    </div>
  );
}

function EmptyState({ algorithm }: { algorithm: FeedAlgorithm }) {
  const configs: Record<FeedAlgorithm, {
    icon: React.ElementType;
    title: string;
    description: string;
    tips: string[];
    cta?: { label: string; href: string };
  }> = {
    smart: {
      icon: Sparkles,
      title: 'Belum Ada Rekomendasi',
      description: 'Algoritma kami sedang mempelajari preferensimu. Bantu kami memberikan rekomendasi yang lebih baik!',
      tips: [
        'Like portfolio yang kamu sukai',
        'Follow kreator favoritmu',
        'Berikan komentar pada karya menarik',
        'Eksplorasi berbagai kategori portfolio'
      ],
      cta: { label: 'Jelajahi Portfolio', href: '/explore' }
    },
    recent: {
      icon: Clock,
      title: 'Belum Ada Portfolio Terbaru',
      description: 'Jadilah yang pertama! Bagikan karya kreatifmu dan inspirasi teman-teman lainnya.',
      tips: [
        'Upload portfolio pertamamu',
        'Showcase project terbaikmu',
        'Dapatkan feedback dari komunitas',
        'Bangun personal branding'
      ],
      cta: { label: 'Buat Portfolio', href: '/portfolio/new' }
    },
    following: {
      icon: Users,
      title: 'Belum Follow Siapa Pun',
      description: 'Temukan dan follow kreator berbakat dari SMKN 4 Malang untuk melihat karya terbaru mereka!',
      tips: [
        'Cari teman sekelas atau alumni',
        'Follow kreator dari jurusanmu',
        'Temukan inspirasi dari senior',
        'Bangun network profesional'
      ],
      cta: { label: 'Cari User', href: '/explore' }
    },
  };

  const config = configs[algorithm];
  const Icon = config.icon;

  return (
    <div className="flex min-h-[60vh] items-center justify-center px-4 py-12">
      <div className="w-full max-w-md text-center">
        {/* Icon */}
        <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-primary/10">
          <Icon className="h-10 w-10 text-primary" />
        </div>

        {/* Title & Description */}
        <h3 className="mb-2 text-xl font-semibold text-foreground">
          {config.title}
        </h3>
        <p className="mb-8 text-sm text-muted-foreground leading-relaxed">
          {config.description}
        </p>

        {/* Tips */}
        <div className="mb-8 rounded-xl border bg-muted/30 p-6 text-left">
          <div className="mb-3 flex items-center gap-2 text-sm font-medium">
            <Heart className="h-4 w-4 text-primary" />
            <span>Tips buat kamu:</span>
          </div>
          <ul className="space-y-2.5">
            {config.tips.map((tip, index) => (
              <li key={index} className="flex items-start gap-2.5 text-sm text-muted-foreground">
                <span className="mt-1 flex h-1.5 w-1.5 shrink-0 rounded-full bg-primary" />
                <span>{tip}</span>
              </li>
            ))}
          </ul>
        </div>

        {/* CTA Button */}
        {config.cta && (
          <Button asChild size="lg" className="w-full sm:w-auto">
            <a href={config.cta.href}>
              {algorithm === 'following' ? (
                <UserPlus className="mr-2 h-4 w-4" />
              ) : algorithm === 'recent' ? (
                <Sparkles className="mr-2 h-4 w-4" />
              ) : (
                <Search className="mr-2 h-4 w-4" />
              )}
              {config.cta.label}
            </a>
          </Button>
        )}
      </div>
    </div>
  );
}


export function SmartFeedList() {
  const { isAuthenticated } = useAuthStore();
  const observerRef = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const [algorithm, setAlgorithm] = useState<FeedAlgorithm>(
    isAuthenticated ? 'smart' : 'recent'
  );
  const [isPulling, setIsPulling] = useState(false);
  const [prefLoaded, setPrefLoaded] = useState(false);

  // Load user's saved preference on mount
  const { data: prefData } = useQuery({
    queryKey: ['feed-preferences'],
    queryFn: () => feedApi.getFeedPreferences(),
    enabled: isAuthenticated && !prefLoaded,
  });

  // Apply saved preference when loaded
  useEffect(() => {
    if (prefData?.data?.algorithm && !prefLoaded) {
      setAlgorithm(prefData.data.algorithm);
      setPrefLoaded(true);
    }
  }, [prefData, prefLoaded]);

  // Fetch feed with infinite scroll
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    isError,
    refetch,
  } = useInfiniteQuery({
    queryKey: ['feed', algorithm],
    queryFn: async ({ pageParam = 1 }) => {
      const response = await feedApi.getFeed({
        algorithm,
        page: pageParam,
        limit: ITEMS_PER_PAGE,
      });
      return response;
    },
    getNextPageParam: (lastPage) => {
      if (!lastPage.meta) return undefined;
      const { current_page, total_pages } = lastPage.meta;
      return current_page < total_pages ? current_page + 1 : undefined;
    },
    initialPageParam: 1,
    enabled: algorithm === 'recent' || isAuthenticated,
  });

  // Save preference mutation
  const savePrefMutation = useMutation({
    mutationFn: (algo: FeedAlgorithm) => feedApi.updateFeedPreferences(algo),
    onError: () => {
      // Silent fail - preference saving is not critical
    },
  });

  // Handle algorithm change
  const handleAlgorithmChange = useCallback(
    (newAlgorithm: FeedAlgorithm) => {
      setAlgorithm(newAlgorithm);
      if (isAuthenticated) {
        savePrefMutation.mutate(newAlgorithm);
      }
    },
    [isAuthenticated, savePrefMutation]
  );

  // Intersection observer for infinite scroll
  useEffect(() => {
    if (observerRef.current) {
      observerRef.current.disconnect();
    }

    observerRef.current = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
        }
      },
      { threshold: 0.1 }
    );

    if (loadMoreRef.current) {
      observerRef.current.observe(loadMoreRef.current);
    }

    return () => {
      observerRef.current?.disconnect();
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  // Pull to refresh handler
  const handleRefresh = async () => {
    setIsPulling(true);
    await refetch();
    setIsPulling(false);
    toast.success('Feed diperbarui');
  };

  // Share handler
  const handleShare = async (item: FeedItem) => {
    const url = `${window.location.origin}/${item.user?.username}/${item.slug}`;
    if (navigator.share) {
      try {
        await navigator.share({
          title: item.judul,
          url,
        });
      } catch {
        // User cancelled or error
      }
    } else {
      await navigator.clipboard.writeText(url);
      toast.success('Link disalin ke clipboard');
    }
  };

  // Flatten pages into single array
  const feedItems = data?.pages.flatMap((page) => page.data || []) || [];

  return (
    <div className="max-w-2xl mx-auto -mt-4 md:-mt-6">
      {/* Sticky Tab Switcher - positioned right below header */}
      <div className="sticky top-14 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b">
        <FeedAlgorithmSwitcher
          value={algorithm}
          onChange={handleAlgorithmChange}
          isAuthenticated={isAuthenticated}
        />
      </div>

      {/* Pull to refresh button (mobile) */}
      <div className="flex justify-center py-2 sm:hidden">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleRefresh}
          disabled={isPulling}
          className="gap-2"
        >
          <RefreshCw className={`h-4 w-4 ${isPulling ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>

      {/* Content */}
      {isLoading ? (
        <FeedSkeleton />
      ) : isError ? (
        <div className="p-8 text-center">
          <p className="text-muted-foreground">Gagal memuat feed.</p>
          <Button variant="outline" onClick={() => refetch()} className="mt-4">
            Coba Lagi
          </Button>
        </div>
      ) : feedItems.length === 0 ? (
        <EmptyState algorithm={algorithm} />
      ) : (
        <>
          {/* Feed Items */}
          <div className="divide-y">
            {feedItems.map((item) => (
              <TimelineFeedItem key={item.id} item={item} algorithm={algorithm} onShare={handleShare} />
            ))}
          </div>

          {/* Load more trigger */}
          <div ref={loadMoreRef} className="py-8 flex justify-center">
            {isFetchingNextPage ? (
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            ) : hasNextPage ? (
              <span className="text-sm text-muted-foreground">Scroll untuk memuat lebih banyak</span>
            ) : feedItems.length > 0 ? (
              <span className="text-sm text-muted-foreground">Tidak ada lagi portfolio</span>
            ) : null}
          </div>
        </>
      )}
    </div>
  );
}
