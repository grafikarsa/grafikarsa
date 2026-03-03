'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { ScrollArea } from '@/components/ui/scroll-area';
import { usersApi } from '@/lib/api';
import { useAuthStore } from '@/lib/stores/auth-store';
import { FollowUser } from '@/lib/types';
import { toast } from 'sonner';
import { Search, Loader2 } from 'lucide-react';
import { useDebounce } from '@/lib/hooks/use-debounce';

interface FollowModalProps {
  username: string;
  type: 'followers' | 'following';
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function FollowModal({ username, type, open, onOpenChange }: FollowModalProps) {
  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search, 300);
  const { user: currentUser, isAuthenticated } = useAuthStore();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: [type, username, debouncedSearch],
    queryFn: () =>
      type === 'followers'
        ? usersApi.getFollowers(username, { search: debouncedSearch || undefined })
        : usersApi.getFollowing(username, { search: debouncedSearch || undefined }),
    enabled: open,
  });

  const users = data?.data || [];

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setSearch('');
    }
    onOpenChange(newOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-h-[85vh] p-0 sm:max-w-md">
        <DialogHeader className="border-b px-4 py-3">
          <DialogTitle>{type === 'followers' ? 'Followers' : 'Following'}</DialogTitle>
          <DialogDescription className="sr-only">
            List of users who {type === 'followers' ? 'follow' : 'are followed by'} this user.
          </DialogDescription>
        </DialogHeader>

        <div className="px-4 py-3">
          <div className="relative">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Cari..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        <ScrollArea className="max-h-[60vh] px-4 pb-4">
          {isLoading ? (
            <div className="space-y-3">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="h-10 w-10 rounded-full" />
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-4 w-28" />
                    <Skeleton className="h-3 w-20" />
                  </div>
                </div>
              ))}
            </div>
          ) : users.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">
              {search
                ? 'Tidak ditemukan'
                : type === 'followers'
                  ? 'Belum ada followers'
                  : 'Belum mengikuti siapapun'}
            </p>
          ) : (
            <div className="space-y-1">
              {users.map((user) => (
                <FollowUserItem
                  key={user.id}
                  user={user}
                  currentUser={currentUser}
                  isAuthenticated={isAuthenticated}
                  queryClient={queryClient}
                  onClose={() => handleOpenChange(false)}
                />
              ))}
            </div>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}

function FollowUserItem({
  user,
  currentUser,
  isAuthenticated,
  queryClient,
  onClose,
}: {
  user: FollowUser;
  currentUser: { username: string } | null;
  isAuthenticated: boolean;
  queryClient: ReturnType<typeof useQueryClient>;
  onClose: () => void;
}) {
  const isCurrentUser = currentUser?.username === user.username;

  const followMutation = useMutation({
    mutationFn: () =>
      user.is_following ? usersApi.unfollow(user.username) : usersApi.follow(user.username),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['followers'] });
      queryClient.invalidateQueries({ queryKey: ['following'] });
      queryClient.invalidateQueries({ queryKey: ['user'] });
      toast.success(user.is_following ? 'Berhasil unfollow' : 'Berhasil follow');
    },
    onError: () => {
      toast.error('Gagal. Silakan coba lagi.');
    },
  });

  return (
    <div className="flex items-center gap-3 rounded-lg p-2 hover:bg-muted/50">
      <Link href={`/${user.username}`} onClick={onClose} className="shrink-0">
        <Avatar className="h-10 w-10">
          <AvatarImage src={user.avatar_url} alt={user.nama} />
          <AvatarFallback>{user.nama?.charAt(0).toUpperCase()}</AvatarFallback>
        </Avatar>
      </Link>
      <div className="flex-1 min-w-0 overflow-hidden">
        <Link href={`/${user.username}`} onClick={onClose} className="block truncate hover:underline">
          <span className="text-sm font-medium">{user.nama}</span>
        </Link>
        <p className="truncate text-xs text-muted-foreground">@{user.username}</p>
      </div>
      {isAuthenticated && !isCurrentUser && (
        <Button
          variant={user.is_following ? 'outline' : 'default'}
          size="sm"
          className="h-8 shrink-0 text-xs"
          onClick={() => followMutation.mutate()}
          disabled={followMutation.isPending}
        >
          {followMutation.isPending ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : user.is_following ? (
            'Unfollow'
          ) : (
            'Follow'
          )}
        </Button>
      )}
    </div>
  );
}
