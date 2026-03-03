'use client';

import { useState, useEffect, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { Search, X, Loader2 } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { usersApi } from '@/lib/api';
import { useDebounce } from '@/lib/hooks/use-debounce';
import { UserCard } from '@/lib/types';

interface SearchPanelProps {
  trigger?: React.ReactNode;
}

export function SearchPanel({ trigger }: SearchPanelProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search, 300);
  const inputRef = useRef<HTMLInputElement>(null);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['search-users', debouncedSearch],
    queryFn: () => usersApi.getUsers({ search: debouncedSearch, limit: 20 }),
    enabled: debouncedSearch.length >= 2,
  });

  const users = data?.data || [];

  useEffect(() => {
    if (open && inputRef.current) {
      setTimeout(() => inputRef.current?.focus(), 100);
    }
  }, [open]);

  const handleSelect = () => {
    setOpen(false);
    setSearch('');
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        {trigger || (
          <Button variant="ghost" size="icon">
            <Search className="h-5 w-5" />
          </Button>
        )}
      </SheetTrigger>
      <SheetContent side="left" className="w-full sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Cari Pengguna</SheetTitle>
        </SheetHeader>
        <div className="mt-4 space-y-4">
          <div className="relative">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              ref={inputRef}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari nama, username, kelas..."
              className="pl-9 pr-9"
            />
            {search && (
              <Button
                variant="ghost"
                size="icon"
                className="absolute right-1 top-1 h-7 w-7"
                onClick={() => setSearch('')}
              >
                <X className="h-4 w-4" />
              </Button>
            )}
          </div>

          <ScrollArea className="h-[calc(100vh-200px)]">
            {!debouncedSearch || debouncedSearch.length < 2 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                Ketik minimal 2 karakter untuk mencari
              </p>
            ) : isLoading || isFetching ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : users.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                Tidak ditemukan hasil untuk &quot;{debouncedSearch}&quot;
              </p>
            ) : (
              <div className="space-y-2">
                {users.map((user) => (
                  <SearchResultItem key={user.id} user={user} onSelect={handleSelect} />
                ))}
              </div>
            )}
          </ScrollArea>
        </div>
      </SheetContent>
    </Sheet>
  );
}

function SearchResultItem({ user, onSelect }: { user: UserCard; onSelect: () => void }) {
  return (
    <Link href={`/${user.username}`} onClick={onSelect}>
      <div className="flex items-center gap-3 rounded-lg p-3 transition-colors hover:bg-muted">
        <Avatar className="h-10 w-10">
          <AvatarImage src={user.avatar_url} alt={user.nama} />
          <AvatarFallback>{user.nama?.charAt(0).toUpperCase()}</AvatarFallback>
        </Avatar>
        <div className="flex-1 overflow-hidden">
          <p className="truncate font-medium">{user.nama}</p>
          <p className="truncate text-sm text-muted-foreground">@{user.username}</p>
        </div>
        <div className="flex flex-col items-end gap-1">
          <Badge variant="outline" className="text-xs capitalize">
            {user.role}
          </Badge>
          {user.kelas && (
            <span className="text-xs text-muted-foreground">{user.kelas.nama}</span>
          )}
        </div>
      </div>
    </Link>
  );
}
