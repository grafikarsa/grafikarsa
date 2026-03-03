'use client';

import { useQuery } from '@tanstack/react-query';
import { Sparkles } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { PortfolioCard } from '@/components/portfolio/portfolio-card';
import { topApi } from '@/lib/api/public';
import { PortfolioCard as PortfolioCardType } from '@/lib/types';
import { cn } from '@/lib/utils';
import { RankingInfo } from '@/components/shared/ranking-info';

const rankStyles = {
  1: 'bg-amber-100 text-amber-700 border-amber-300 dark:bg-amber-900/30 dark:text-amber-400 dark:border-amber-700',
  2: 'bg-gray-100 text-gray-700 border-gray-300 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600',
  3: 'bg-orange-100 text-orange-700 border-orange-300 dark:bg-orange-900/30 dark:text-orange-400 dark:border-orange-700',
};

const rankLabels = {
  1: '# 1',
  2: '# 2',
  3: '# 3',
};

function ProjectCardSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="mx-auto h-6 w-20 rounded-full" />
      <div className="rounded-xl border bg-card overflow-hidden p-3">
        <Skeleton className="aspect-[4/3] rounded-xl" />
        <div className="mt-3 space-y-2">
          <Skeleton className="h-5 w-3/4" />
          <div className="flex items-center gap-2">
            <Skeleton className="h-8 w-8 rounded-full" />
            <div className="space-y-1">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-16" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export function TopProjectsSection() {
  const { data, isLoading } = useQuery({
    queryKey: ['top-projects'],
    queryFn: () => topApi.getTopProjects(),
    staleTime: 5 * 60 * 1000,
  });

  const projects = data?.data || [];

  if (isLoading) {
    return (
      <section className="py-16">
        <div className="mx-auto max-w-6xl px-6">
          <div className="mb-10 text-center">
            <div className="inline-flex items-center gap-2 rounded-full bg-primary/10 px-4 py-1.5 text-sm font-medium text-primary mb-4">
              <Sparkles className="h-4 w-4" />
              Top Projects
            </div>
            <h2 className="text-3xl font-semibold tracking-tight flex justify-center items-center gap-2">
              Karya Terbaik
              <RankingInfo type="projects" />
            </h2>
            <p className="mt-2 text-muted-foreground">Portofolio dengan kualitas dan apresiasi tertinggi</p>
          </div>
          <div className="grid gap-6 md:grid-cols-3">
            {[1, 2, 3].map((i) => (
              <ProjectCardSkeleton key={i} />
            ))}
          </div>
        </div>
      </section>
    );
  }

  if (projects.length === 0) {
    return null;
  }

  return (
    <section className="py-16">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mb-10 text-center">
          <div className="inline-flex items-center gap-2 rounded-full bg-primary/10 px-4 py-1.5 text-sm font-medium text-primary mb-4">
            <Sparkles className="h-4 w-4" />
            Top Projects
          </div>
          <h2 className="text-3xl font-semibold tracking-tight flex justify-center items-center gap-2">
            Karya Terbaik
            <RankingInfo type="projects" />
          </h2>
          <p className="mt-2 text-muted-foreground">Portofolio dengan kualitas dan apresiasi tertinggi</p>
        </div>
        <div className="grid gap-6 md:grid-cols-3">
          {projects.map((project, index) => {
            const rank = (index + 1) as 1 | 2 | 3;
            // Convert TopProject to PortfolioCard type
            const portfolioCardData: PortfolioCardType = {
              id: project.id,
              judul: project.judul,
              slug: project.slug,
              thumbnail_url: project.thumbnail_url || undefined,
              published_at: project.published_at || undefined,
              created_at: project.published_at || new Date().toISOString(),
              updated_at: project.published_at || new Date().toISOString(),
              like_count: project.like_count,
              user: {
                id: project.user_id,
                username: project.username,
                nama: project.user_nama,
                avatar_url: project.user_avatar || undefined,
                role: 'student',
              },
            };

            return (
              <div key={project.id} className="relative">
                {/* Rank Badge */}
                <div className="flex justify-center mb-2">
                  <Badge variant="outline" className={cn('text-sm px-3 py-1 font-medium', rankStyles[rank])}>
                    {rankLabels[rank]}
                  </Badge>
                </div>
                <PortfolioCard portfolio={portfolioCardData} />
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
