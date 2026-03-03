'use client';

import Link from 'next/link';
import Image from 'next/image';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Heart, Share2, Eye } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { portfoliosApi } from '@/lib/api';
import { ApiResponse, FeedItem } from '@/lib/types';
import { formatDistanceToNow } from '@/lib/utils/format';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/lib/stores/auth-store';
import { InfiniteData } from '@tanstack/react-query';

interface TimelineFeedItemProps {
  item: FeedItem;
  algorithm: string;
  onShare?: (item: FeedItem) => void;
}

export function TimelineFeedItem({ item, algorithm, onShare }: TimelineFeedItemProps) {
  const { user: currentUser } = useAuthStore();
  const queryClient = useQueryClient();

  const likeMutation = useMutation({
    mutationFn: () =>
      item.is_liked ? portfoliosApi.unlikePortfolio(item.id) : portfoliosApi.likePortfolio(item.id),
    onMutate: async () => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['feed', algorithm] });

      // Snapshot previous value
      const previousData = queryClient.getQueryData<InfiniteData<ApiResponse<FeedItem[]>>>(['feed', algorithm]);

      // Optimistically update cache
      queryClient.setQueryData<InfiniteData<ApiResponse<FeedItem[]>>>(['feed', algorithm], (old) => {
        if (!old) return old;
        return {
          ...old,
          pages: old.pages.map((page) => ({
            ...page,
            data: page.data?.map((feedItem) =>
              feedItem.id === item.id
                ? {
                    ...feedItem,
                    is_liked: !feedItem.is_liked,
                    like_count: feedItem.is_liked ? feedItem.like_count - 1 : feedItem.like_count + 1,
                  }
                : feedItem
            ),
          })),
        };
      });

      return { previousData };
    },
    onError: (_err, _vars, context) => {
      // Rollback on error
      if (context?.previousData) {
        queryClient.setQueryData(['feed', algorithm], context.previousData);
      }
      toast.error('Gagal memproses like');
    },
    onSuccess: (data) => {
      // Update with server response
      if (data.data) {
        queryClient.setQueryData<InfiniteData<ApiResponse<FeedItem[]>>>(['feed', algorithm], (old) => {
          if (!old) return old;
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              data: page.data?.map((feedItem) =>
                feedItem.id === item.id
                  ? { ...feedItem, is_liked: data.data!.is_liked, like_count: data.data!.like_count }
                  : feedItem
              ),
            })),
          };
        });
      }
    },
  });

  const handleLike = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!currentUser) {
      toast.error('Login untuk menyukai portfolio');
      return;
    }
    likeMutation.mutate();
  };

  const handleShare = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onShare?.(item);
  };

  const username = item.user?.username || 'user';

  return (
    <Card className="overflow-hidden border-0 border-b rounded-none shadow-none hover:bg-muted/30 transition-colors">
      <div className="p-4">
        {/* Author Header */}
        <div className="flex items-start gap-3">
          <Link href={`/${username}`} onClick={(e) => e.stopPropagation()}>
            <Avatar className="h-10 w-10">
              <AvatarImage src={item.user?.avatar_url} alt={item.user?.nama} />
              <AvatarFallback>{item.user?.nama?.charAt(0) || 'U'}</AvatarFallback>
            </Avatar>
          </Link>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <Link
                href={`/${username}`}
                className="font-semibold text-sm hover:underline"
                onClick={(e) => e.stopPropagation()}
              >
                {item.user?.nama}
              </Link>
              <Link
                href={`/${username}`}
                className="text-muted-foreground text-sm hover:underline"
                onClick={(e) => e.stopPropagation()}
              >
                @{username}
              </Link>
              {item.user?.kelas_nama && (
                <span className="text-muted-foreground text-xs">• {item.user.kelas_nama}</span>
              )}
            </div>
            {(item.published_at || item.created_at) && (
              <p className="text-xs text-muted-foreground mt-0.5">
                {formatDistanceToNow(item.published_at || item.created_at!)}
              </p>
            )}
          </div>
        </div>

        {/* Content */}
        <Link href={`/${username}/${item.slug}`} className="block mt-3">
          {/* Title */}
          <h3 className="font-semibold text-base leading-tight mb-2 hover:text-primary transition-colors">
            {item.judul}
          </h3>

          {/* Preview Text */}
          {item.preview_text && (
            <p className="text-sm text-muted-foreground line-clamp-3 mb-3">{item.preview_text}</p>
          )}

          {/* Thumbnail */}
          {item.thumbnail_url && (
            <div className="relative aspect-video w-full overflow-hidden rounded-xl bg-muted">
              <Image src={item.thumbnail_url} alt={item.judul} fill className="object-cover" />
            </div>
          )}
        </Link>

        {/* Tags */}
        {item.tags && item.tags.length > 0 && (
          <div className="flex gap-2 mt-3 overflow-x-auto scrollbar-hide pb-1">
            {item.tags.map((tag) => (
              <Badge
                key={tag.id}
                variant="secondary"
                className="rounded-full px-2.5 py-0.5 text-xs font-normal whitespace-nowrap shrink-0"
              >
                {tag.nama}
              </Badge>
            ))}
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center gap-1 mt-3 -ml-2">
          {/* Like Button */}
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              'h-8 px-2 gap-1.5 text-muted-foreground hover:text-red-500',
              item.is_liked && 'text-red-500'
            )}
            onClick={handleLike}
            disabled={likeMutation.isPending}
          >
            <Heart className={cn('h-4 w-4', item.is_liked && 'fill-current')} />
            <span className="text-xs">{item.like_count}</span>
          </Button>

          {/* View Count */}
          <div className="flex items-center gap-1.5 px-2 text-muted-foreground">
            <Eye className="h-4 w-4" />
            <span className="text-xs">{item.view_count}</span>
          </div>

          {/* Share Button */}
          <Button
            variant="ghost"
            size="sm"
            className="h-8 px-2 text-muted-foreground hover:text-primary"
            onClick={handleShare}
          >
            <Share2 className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </Card>
  );
}
