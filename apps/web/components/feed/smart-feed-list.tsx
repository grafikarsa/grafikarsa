'use client';

import { useEffect, useCallback, useRef, useState, memo } from 'react';
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

const ITEMS_PER_PAGE = 15; // Reduced from 20 for faster initial load

function FeedSkeleton() {
  return (
    <div className="space-y-0">
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="p-3 md:p-4 border-b">
          <div className="flex items-start gap-2.5 md:gap-3">
            <Skeleton className="h-9 w-9 md:h-10 md:w-10 rounded-full shrink-0" />
            <div className="flex-1 space-y-1.5 md:space-y-2 min-w-0">
              <Skeleton className="h-3.5 md:h-4 w-28 md:w-32" />
              <Skeleton className="h-3 w-20 md:w-24" />
            </div>
          </div>
          <div className="mt-3 space-y-1.5 md:space-y-2">
            <Skeleton className="h-4 md:h-5 w-3/4" />
            <Skeleton className="h-3 md:h-4 w-full" />
            <Skeleton className="h-3 md:h-4 w-2/3" />
          </div>
          <Skeleton className="mt-3 aspect-video w-full rounded-lg md:rounded-xl" />
          <div className="mt-3 flex gap-2">
            <Skeleton className="h-7 md:h-8 w-16 md:w-20 rounded-md" />
            <Skeleton className="h-7 md:h-8 w-16 md:w-20 rounded-md" />
            <Skeleton className="h-7 md:h-8 w-10 md:w-12 rounded-md" />
          </div>
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
    <div className="flex min-h-[60vh] items-center justify-center px-4 md:px-6 py-8 md:py-12">
      <div className="w-full max-w-md text-center">
        {/* Icon */}
        <div className="mx-auto mb-4 md:mb-6 flex h-16 w-16 md:h-20 md:w-20 items-center justify-center rounded-full bg-primary/10">
          <Icon className="h-8 w-8 md:h-10 md:w-10 text-primary" />
        </div>

        {/* Title & Description */}
        <h3 className="mb-2 text-lg md:text-xl font-semibold text-foreground">
          {config.title}
        </h3>
        <p className="mb-6 md:mb-8 text-sm text-muted-foreground leading-relaxed px-2">
          {config.description}
        </p>

        {/* Tips */}
        <div className="mb-6 md:mb-8 rounded-lg md:rounded-xl border bg-muted/30 p-4 md:p-6 text-left">
          <div className="mb-2.5 md:mb-3 flex items-center gap-2 text-xs md:text-sm font-medium">
            <Heart className="h-3.5 w-3.5 md:h-4 md:w-4 text-primary" />
            <span>Tips buat kamu:</span>
          </div>
          <ul className="space-y-2 md:space-y-2.5">
            {config.tips.map((tip, index) => (
              <li key={index} className="flex items-start gap-2 md:gap-2.5 text-xs md:text-sm text-muted-foreground">
                <span className="mt-1 flex h-1.5 w-1.5 shrink-0 rounded-full bg-primary" />
                <span className="leading-relaxed">{tip}</span>
              </li>
            ))}
          </ul>
        </div>

        {/* CTA Button */}
        {config.cta && (
          <Button asChild size="default" className="w-full sm:w-auto">
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
    staleTime: 1000 * 60 * 2, // Cache for 2 minutes
    gcTime: 1000 * 60 * 5, // Keep in cache for 5 minutes
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

  // Intersection observer for infinite scroll with optimized threshold
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
      { 
        threshold: 0.1,
        rootMargin: '200px' // Prefetch 200px before reaching the end
      }
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

  // Share handler - memoized
  const handleShare = useCallback(async (item: FeedItem) => {
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
  }, []);

  // Flatten pages into single array
  const feedItems = data?.pages.flatMap((page) => page.data || []) || [];

  return (
    <div className="w-full max-w-2xl mx-auto">
      {/* Sticky Tab Switcher - positioned right below header */}
      <div className="sticky top-14 md:top-14 z-40 bg-background border-b">
        <FeedAlgorithmSwitcher
          value={algorithm}
          onChange={handleAlgorithmChange}
          isAuthenticated={isAuthenticated}
        />
      </div>

      {/* Pull to refresh button (mobile) */}
      <div className="flex justify-center py-3 border-b md:hidden">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleRefresh}
          disabled={isPulling}
          className="gap-2 text-xs"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${isPulling ? 'animate-spin' : ''}`} />
          Refresh Feed
        </Button>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="px-0 md:px-0">
          <FeedSkeleton />
        </div>
      ) : isError ? (
        <div className="p-6 md:p-8 text-center">
          <p className="text-sm md:text-base text-muted-foreground">Gagal memuat feed.</p>
          <Button variant="outline" onClick={() => refetch()} className="mt-4" size="sm">
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
          <div ref={loadMoreRef} className="py-6 md:py-8 flex justify-center">
            {isFetchingNextPage ? (
              <Loader2 className="h-5 w-5 md:h-6 md:w-6 animate-spin text-muted-foreground" />
            ) : hasNextPage ? (
              <span className="text-xs md:text-sm text-muted-foreground">Scroll untuk memuat lebih banyak</span>
            ) : feedItems.length > 0 ? (
              <div className="text-center space-y-2">
                <span className="block text-xs md:text-sm text-muted-foreground">Tidak ada lagi portfolio</span>
                <span className="block text-xs text-muted-foreground/70">Kamu sudah sampai di akhir feed</span>
              </div>
            ) : null}
          </div>
        </>
      )}
    </div>
  );
}
