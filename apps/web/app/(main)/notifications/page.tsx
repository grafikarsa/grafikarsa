'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from '@tanstack/react-query';
import { Bell, Check, Trash2, UserPlus, Heart, CheckCircle, XCircle, Loader2 } from 'lucide-react';
import { notificationsApi } from '@/lib/api';
import { Notification, NotificationType } from '@/lib/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import { formatDistanceToNow } from '@/lib/utils/format';
import { toast } from 'sonner';

const notificationIcons: Record<NotificationType, React.ReactNode> = {
  new_follower: <UserPlus className="h-5 w-5 text-blue-500" />,
  portfolio_liked: <Heart className="h-5 w-5 text-red-500" />,
  portfolio_approved: <CheckCircle className="h-5 w-5 text-green-500" />,
  portfolio_rejected: <XCircle className="h-5 w-5 text-amber-500" />,
};

function getNotificationLink(notification: Notification | null | undefined): string | null {
  if (!notification) return null;
  const data = notification.data as Record<string, string> | undefined | null;
  if (!data) return null;

  switch (notification.type) {
    case 'new_follower':
      return data.follower_username ? `/${data.follower_username}` : null;
    case 'portfolio_liked':
    case 'portfolio_approved':
    case 'portfolio_rejected':
      return data.portfolio_slug ? `/me/portfolios` : null;
    default:
      return null;
  }
}

export default function NotificationsPage() {
  const [filter, setFilter] = useState<'all' | 'unread'>('all');
  const queryClient = useQueryClient();

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
  } = useInfiniteQuery({
    queryKey: ['notifications', 'list', filter],
    queryFn: ({ pageParam = 1 }) =>
      notificationsApi.getNotifications({
        page: pageParam,
        limit: 20,
        unread_only: filter === 'unread',
      }),
    getNextPageParam: (lastPage) => {
      if (lastPage.meta.page < lastPage.meta.total_pages) {
        return lastPage.meta.page + 1;
      }
      return undefined;
    },
    initialPageParam: 1,
  });

  const markAsReadMutation = useMutation({
    mutationFn: notificationsApi.markAsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  const markAllAsReadMutation = useMutation({
    mutationFn: notificationsApi.markAllAsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      toast.success('Semua notifikasi ditandai sudah dibaca');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: notificationsApi.deleteNotification,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      toast.success('Notifikasi dihapus');
    },
  });

  const notifications = (data?.pages.flatMap((page) => page.data) ?? []).filter(
    (n): n is Notification => n != null
  );
  const unreadCount = data?.pages[0]?.meta.unread_count ?? 0;

  const handleNotificationClick = (notification: Notification) => {
    if (notification && !notification.is_read) {
      markAsReadMutation.mutate(notification.id);
    }
  };

  if (isLoading) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Bell className="h-6 w-6" />
          <h1 className="text-2xl font-bold">Notifikasi</h1>
          {unreadCount > 0 && (
            <span className="rounded-full bg-red-500 px-2 py-0.5 text-xs font-medium text-white">
              {unreadCount} belum dibaca
            </span>
          )}
        </div>
        {unreadCount > 0 && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => markAllAsReadMutation.mutate()}
            disabled={markAllAsReadMutation.isPending}
          >
            <Check className="mr-2 h-4 w-4" />
            Tandai semua dibaca
          </Button>
        )}
      </div>

      <Tabs value={filter} onValueChange={(v) => setFilter(v as 'all' | 'unread')}>
        <TabsList>
          <TabsTrigger value="all">Semua</TabsTrigger>
          <TabsTrigger value="unread">Belum Dibaca</TabsTrigger>
        </TabsList>
      </Tabs>

      {notifications.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Bell className="h-12 w-12 text-muted-foreground/50" />
            <p className="mt-4 text-muted-foreground">
              {filter === 'unread' ? 'Tidak ada notifikasi yang belum dibaca' : 'Tidak ada notifikasi'}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {notifications.map((notification) => {
            const link = getNotificationLink(notification);

            return (
              <Card
                key={notification.id}
                className={cn(
                  'transition-colors hover:bg-muted/50',
                  !notification.is_read && 'border-l-4 border-l-blue-500 bg-muted/30'
                )}
              >
                <CardContent className="flex items-start gap-4 p-4">
                  <div className="mt-0.5">{notificationIcons[notification.type]}</div>
                  <div className="flex-1 min-w-0">
                    {link ? (
                      <Link
                        href={link}
                        onClick={() => handleNotificationClick(notification)}
                        className="block"
                      >
                        <p className={cn('text-sm', !notification.is_read && 'font-medium')}>
                          {notification.message || notification.title}
                        </p>
                      </Link>
                    ) : (
                      <p className={cn('text-sm', !notification.is_read && 'font-medium')}>
                        {notification.message || notification.title}
                      </p>
                    )}
                    <p className="mt-1 text-xs text-muted-foreground">
                      {formatDistanceToNow(notification.created_at)}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    {!notification.is_read && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => markAsReadMutation.mutate(notification.id)}
                        disabled={markAsReadMutation.isPending}
                      >
                        <Check className="h-4 w-4" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-muted-foreground hover:text-destructive"
                      onClick={() => deleteMutation.mutate(notification.id)}
                      disabled={deleteMutation.isPending}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })}

          {hasNextPage && (
            <div className="flex justify-center pt-4">
              <Button
                variant="outline"
                onClick={() => fetchNextPage()}
                disabled={isFetchingNextPage}
              >
                {isFetchingNextPage ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Memuat...
                  </>
                ) : (
                  'Muat lebih banyak'
                )}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
