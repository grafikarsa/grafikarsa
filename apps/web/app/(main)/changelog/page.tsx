'use client';

import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { History } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { getChangelogs, markAllChangelogsAsRead } from '@/lib/api/changelog';
import { ChangelogTimelineItem } from '@/components/changelog/changelog-timeline-item';
import { useAuthStore } from '@/lib/stores/auth-store';
import type { Changelog } from '@/lib/types/changelog';

export default function ChangelogPage() {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ['changelogs', page],
    queryFn: () => getChangelogs(page, 10),
  });

  // Mark all as read when user visits the page
  const markReadMutation = useMutation({
    mutationFn: () => markAllChangelogsAsRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['changelog-unread-count'] });
    },
  });

  useEffect(() => {
    if (user) {
      markReadMutation.mutate();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user]);

  // axios wraps response in .data, then our API wraps in .data again
  const changelogs = data?.data?.data || [];
  const pagination = data?.data?.meta;

  // Group changelogs by date
  const groupedChangelogs = changelogs.reduce(
    (acc: Record<string, Changelog[]>, changelog: Changelog) => {
      const date = changelog.release_date;
      if (!acc[date]) {
        acc[date] = [];
      }
      acc[date].push(changelog);
      return acc;
    },
    {}
  );

  if (isLoading) {
    return (
      <div className="mx-auto max-w-4xl px-4 pb-12 pt-20 sm:px-6 lg:px-8">
        <div className="mb-12 text-center">
          <Skeleton className="mx-auto h-10 w-48" />
          <Skeleton className="mx-auto mt-3 h-5 w-80" />
        </div>
        <div className="space-y-8">
          {[...Array(3)].map((_, i) => (
            <Skeleton key={i} className="h-64" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl px-4 pb-12 pt-20 sm:px-6 lg:px-8">
      {/* Header */}
      <div className="mb-12 text-center">
        <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">Changelog</h1>
        <p className="mt-3 text-lg text-muted-foreground">
          Lihat update dan perubahan terbaru di Grafikarsa
        </p>
      </div>

      {/* Timeline */}
      {changelogs.length === 0 ? (
        <Card className="border-dashed py-16">
          <div className="flex flex-col items-center justify-center">
            <History className="h-12 w-12 text-muted-foreground" />
            <h3 className="mt-4 text-lg font-semibold">Belum ada changelog</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Changelog akan muncul di sini saat ada update baru
            </p>
          </div>
        </Card>
      ) : (
        <div className="relative">
          {/* Timeline */}
          {Object.entries(groupedChangelogs).map(([date, items]) => {
            const releaseDate = new Date(date);
            const formattedDate = releaseDate.toLocaleDateString('id-ID', {
              day: 'numeric',
              month: 'long',
              year: 'numeric',
            });

            return (
              <div key={date} className="relative flex gap-6 pb-10 last:pb-0">
                {/* Left side - Date (sticky) */}
                <div className="hidden w-32 shrink-0 md:block">
                  <div className="sticky top-24 text-right">
                    <span className="text-sm font-medium text-muted-foreground">
                      {formattedDate}
                    </span>
                  </div>
                </div>

                {/* Timeline line and dot */}
                <div className="relative flex flex-col items-center">
                  <div className="h-3 w-3 shrink-0 rounded-full border-2 border-primary bg-background" />
                  <div className="w-0.5 grow bg-border" />
                </div>

                {/* Right side - Content */}
                <div className="flex-1 pb-2">
                  {/* Mobile date */}
                  <div className="mb-3 text-sm font-medium text-muted-foreground md:hidden">
                    {formattedDate}
                  </div>

                  {/* Changelog items for this date */}
                  <div className="space-y-6">
                    {(items as Changelog[]).map((changelog: Changelog) => (
                      <ChangelogTimelineItem key={changelog.id} changelog={changelog} />
                    ))}
                  </div>
                </div>
              </div>
            );
          })}

          {/* Pagination */}
          {pagination && pagination.total_pages > 1 && (
            <div className="mt-8 flex items-center justify-center gap-3">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                Sebelumnya
              </Button>
              <span className="text-sm text-muted-foreground">
                Halaman {page} dari {pagination.total_pages}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
                disabled={page === pagination.total_pages}
              >
                Selanjutnya
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
