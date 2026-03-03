'use client';

import { useQuery } from '@tanstack/react-query';
import { feedApi } from '@/lib/api';
import { PortfolioCard } from '@/components/portfolio/portfolio-card';
import { Skeleton } from '@/components/ui/skeleton';

function FeedSkeleton() {
  return (
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="space-y-3">
          <Skeleton className="aspect-video w-full rounded-lg" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ))}
    </div>
  );
}

export function FeedList() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['feed'],
    queryFn: () => feedApi.getFeed({ limit: 20 }),
  });

  if (isLoading) {
    return (
      <div className="container mx-auto">
        <h1 className="mb-6 text-2xl font-bold">Feed</h1>
        <FeedSkeleton />
      </div>
    );
  }

  if (error) {
    return (
      <div className="container mx-auto">
        <h1 className="mb-6 text-2xl font-bold">Feed</h1>
        <p className="text-muted-foreground">Gagal memuat feed. Silakan coba lagi.</p>
      </div>
    );
  }

  const portfolios = data?.data || [];

  if (portfolios.length === 0) {
    return (
      <div className="container mx-auto">
        <h1 className="mb-6 text-2xl font-bold">Feed</h1>
        <div className="rounded-lg border border-dashed p-12 text-center">
          <p className="text-muted-foreground">
            Belum ada portofolio di feed. Follow user lain untuk melihat karya mereka!
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto">
      <h1 className="mb-6 text-2xl font-bold">Feed</h1>
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {portfolios.map((portfolio) => (
          <PortfolioCard key={portfolio.id} portfolio={portfolio} />
        ))}
      </div>
    </div>
  );
}
