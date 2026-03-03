'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Bell, Check, UserPlus, Heart, CheckCircle, XCircle, MessageSquare } from 'lucide-react';
import { notificationsApi, usersApi } from '@/lib/api';
import { Notification, NotificationType } from '@/lib/types';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import { formatDistanceToNow } from '@/lib/utils/format';

const notificationIcons: Record<NotificationType, React.ReactNode> = {
  new_follower: <UserPlus className="h-4 w-4 text-blue-500" />,
  portfolio_liked: <Heart className="h-4 w-4 text-red-500" />,
  portfolio_approved: <CheckCircle className="h-4 w-4 text-green-500" />,
  portfolio_rejected: <XCircle className="h-4 w-4 text-amber-500" />,
  feedback_updated: <MessageSquare className="h-4 w-4 text-purple-500" />,
  new_comment: <MessageSquare className="h-4 w-4 text-blue-500" />,
  reply_comment: <MessageSquare className="h-4 w-4 text-green-500" />,
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
      return data.portfolio_slug ? `/me/portfolios` : null; // Logic check: Liked usually implies someone liked MY portfolio? But if I am liker? No, notification is for receiver.
    // If someone liked MY portfolio, I should go to my portfolio detail to see it? Or generic list?
    // Actually `portfolio_liked` notification data usually contains `slug` if implemented correctly in backend.
    // Let's assume `portfolio_slug` exists for these. 
    // Ideally should be `/${username}/${slug}` but we might not have username of owner (me) easily if not in data.
    // But wait, "me" routes are usually dashboard.
    // For comments, we want public page.
    case 'new_comment':
    case 'reply_comment':
      const ownerUsername = data.portfolio_owner_username;
      const slug = data.portfolio_slug;
      const commentId = data.comment_id;

      if (ownerUsername && slug) {
        return `/${ownerUsername}/${slug}#comment-${commentId || 'section'}`;
      }
      return null;
    case 'feedback_updated':
      return null; // No specific page for user feedback yet
    default:
      return null;
  }
}


export function NotificationBell() {
  const [open, setOpen] = useState(false);
  const queryClient = useQueryClient();

  const { data: countData } = useQuery({
    queryKey: ['notifications', 'count'],
    queryFn: () => notificationsApi.getUnreadCount(),
    refetchInterval: 30000, // Poll every 30 seconds
  });

  const { data: notificationsData } = useQuery({
    queryKey: ['notifications', 'preview'],
    queryFn: () => notificationsApi.getNotifications({ limit: 5 }),
    enabled: open,
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
    },
  });

  const unreadCount = countData?.data?.unread_count ?? 0;
  const notifications = (notificationsData?.data ?? []).filter(
    (n): n is Notification => n != null
  );

  const handleNotificationClick = (notification: Notification) => {
    if (notification && !notification.is_read) {
      markAsReadMutation.mutate(notification.id);
    }
    setOpen(false);
  };

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="relative">
          <Bell className="h-5 w-5" />
          {unreadCount > 0 && (
            <span className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-[10px] font-medium text-white">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        <DropdownMenuLabel className="flex items-center justify-between">
          <span>Notifikasi</span>
          {unreadCount > 0 && (
            <Button
              variant="ghost"
              size="sm"
              className="h-auto p-0 text-xs text-muted-foreground hover:text-foreground"
              onClick={() => markAllAsReadMutation.mutate()}
              disabled={markAllAsReadMutation.isPending}
            >
              <Check className="mr-1 h-3 w-3" />
              Tandai semua dibaca
            </Button>
          )}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {notifications.length === 0 ? (
          <div className="py-6 text-center text-sm text-muted-foreground">
            Tidak ada notifikasi
          </div>
        ) : (
          <>
            {notifications.map((notification) => {
              const link = getNotificationLink(notification);
              const isNewFollower = notification.type === 'new_follower';
              // Cast data to any to safely access enriched fields which might be mixed types
              const notifData = notification.data as any;

              const isFeedback = notification.type === 'feedback_updated';
              const feedbackData = isFeedback ? notifData : null;

              const showFollowBack = isNewFollower && notifData?.follower_username && !notifData?.follower_is_following;

              const handleFollowBack = (e: React.MouseEvent, username: string) => {
                e.preventDefault();
                e.stopPropagation();
                usersApi.follow(username).then(() => {
                  queryClient.invalidateQueries({ queryKey: ['notifications'] });
                  queryClient.invalidateQueries({ queryKey: ['followers'] });
                  queryClient.invalidateQueries({ queryKey: ['following'] });
                });
              };

              const content = (
                <div className="flex gap-3">
                  <div className="mt-0.5">{notificationIcons[notification.type]}</div>
                  <div className="flex-1 space-y-1">
                    <p className={cn('text-sm', !notification.is_read && 'font-medium')}>
                      {notification.message || notification.title}
                    </p>
                    {isFeedback && feedbackData && (
                      <div className="text-xs text-muted-foreground">
                        <p className="font-medium text-foreground/80">Oleh: {feedbackData.actor_role}</p>
                        {feedbackData.admin_note && (
                          <p className="mt-0.5 line-clamp-2 italic">"{feedbackData.admin_note}"</p>
                        )}
                      </div>
                    )}
                    {showFollowBack && (
                      <div className="mt-2">
                        <Button
                          size="sm"
                          variant="outline"
                          className="h-7 text-xs"
                          onClick={(e) => handleFollowBack(e, notifData.follower_username)}
                        >
                          <UserPlus className="mr-1.5 h-3 w-3" />
                          Follback
                        </Button>
                      </div>
                    )}
                    <p className="text-xs text-muted-foreground">
                      {formatDistanceToNow(notification.created_at)}
                    </p>
                  </div>
                  {!notification.is_read && (
                    <div className="mt-2 h-2 w-2 shrink-0 rounded-full bg-blue-500" />
                  )}
                </div>
              );

              return (
                <DropdownMenuItem
                  key={notification.id}
                  className={cn('cursor-pointer p-3', !notification.is_read && 'bg-muted/50')}
                  onClick={() => handleNotificationClick(notification)}
                  asChild={!!link}
                >
                  {link ? <Link href={link}>{content}</Link> : content}
                </DropdownMenuItem>
              );
            })}
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild className="cursor-pointer justify-center">
              <Link href="/notifications" className="text-sm text-primary">
                Lihat semua notifikasi
              </Link>
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
