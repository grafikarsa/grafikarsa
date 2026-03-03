'use client';

import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Check, ChevronsUpDown, X, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { searchUsers } from '@/lib/api/search';
import type { ChangelogContributorRequest } from '@/lib/types/changelog';
import { cn } from '@/lib/utils';
import { useDebounce } from '@/lib/hooks/use-debounce';

interface UserOption {
  id: string;
  username: string;
  nama: string;
  avatar_url?: string;
}

interface Props {
  value: ChangelogContributorRequest[];
  onChange: (value: ChangelogContributorRequest[]) => void;
  initialUserMap?: Record<string, UserOption>;
}

export function ContributorPicker({ value, onChange, initialUserMap = {} }: Props) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [selectedUser, setSelectedUser] = useState<UserOption | null>(null);
  const [contribution, setContribution] = useState('');
  const debouncedSearch = useDebounce(search, 300);

  const { data, isLoading } = useQuery({
    queryKey: ['search-users', debouncedSearch],
    queryFn: () => searchUsers(debouncedSearch, 1, 10),
    enabled: debouncedSearch.length >= 2,
  });

  // axios wraps response in .data, then our API wraps in .data again
  const users = data?.data?.data || [];

  // Track selected user info for display - merge with initial user map
  const [userMap, setUserMap] = useState<Record<string, UserOption>>(initialUserMap);

  // Update userMap when initialUserMap changes (e.g., when editing existing changelog)
  useEffect(() => {
    if (Object.keys(initialUserMap).length > 0) {
      setUserMap((prev) => ({ ...prev, ...initialUserMap }));
    }
  }, [initialUserMap]);

  const handleSelectUser = (user: UserOption) => {
    setSelectedUser(user);
    setUserMap((prev) => ({ ...prev, [user.id]: user }));
    setOpen(false);
    setSearch('');
  };

  const handleAddContributor = () => {
    if (!selectedUser || !contribution.trim()) return;

    // Check if already added
    if (value.some((c) => c.user_id === selectedUser.id)) {
      return;
    }

    onChange([...value, { user_id: selectedUser.id, contribution: contribution.trim() }]);
    setSelectedUser(null);
    setContribution('');
  };

  const handleRemoveContributor = (userId: string) => {
    onChange(value.filter((c) => c.user_id !== userId));
  };

  return (
    <div className="space-y-3">
      {/* Add new contributor */}
      <div className="flex gap-2">
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              role="combobox"
              aria-expanded={open}
              className="w-[200px] justify-between"
            >
              {selectedUser ? (
                <div className="flex items-center gap-2 truncate">
                  <Avatar className="h-5 w-5">
                    <AvatarImage src={selectedUser.avatar_url} />
                    <AvatarFallback className="text-[10px]">
                      {selectedUser.nama?.charAt(0)}
                    </AvatarFallback>
                  </Avatar>
                  <span className="truncate">{selectedUser.nama}</span>
                </div>
              ) : (
                'Pilih user...'
              )}
              <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-[300px] p-0" align="start">
            <Command shouldFilter={false}>
              <CommandInput
                placeholder="Cari user..."
                value={search}
                onValueChange={setSearch}
              />
              <CommandList>
                {isLoading ? (
                  <div className="flex items-center justify-center py-6">
                    <Loader2 className="h-4 w-4 animate-spin" />
                  </div>
                ) : users.length === 0 ? (
                  <CommandEmpty>
                    {debouncedSearch.length < 2
                      ? 'Ketik minimal 2 karakter'
                      : 'User tidak ditemukan'}
                  </CommandEmpty>
                ) : (
                  <CommandGroup>
                    {users.map((user: UserOption) => (
                      <CommandItem
                        key={user.id}
                        value={user.id}
                        onSelect={() => handleSelectUser(user)}
                        className="cursor-pointer"
                      >
                        <div className="flex items-center gap-2">
                          <Avatar className="h-6 w-6">
                            <AvatarImage src={user.avatar_url} />
                            <AvatarFallback className="text-[10px]">
                              {user.nama?.charAt(0)}
                            </AvatarFallback>
                          </Avatar>
                          <div className="flex flex-col">
                            <span className="text-sm">{user.nama}</span>
                            <span className="text-xs text-muted-foreground">
                              @{user.username}
                            </span>
                          </div>
                        </div>
                        <Check
                          className={cn(
                            'ml-auto h-4 w-4',
                            selectedUser?.id === user.id ? 'opacity-100' : 'opacity-0'
                          )}
                        />
                      </CommandItem>
                    ))}
                  </CommandGroup>
                )}
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>

        <Input
          placeholder="Kontribusi/Role"
          value={contribution}
          onChange={(e) => setContribution(e.target.value)}
          className="flex-1"
        />

        <Button
          type="button"
          onClick={handleAddContributor}
          disabled={!selectedUser || !contribution.trim()}
        >
          Tambah
        </Button>
      </div>

      {/* List of contributors */}
      {value.length > 0 && (
        <div className="space-y-2">
          {value.map((contrib, index) => {
            const user = userMap[contrib.user_id];
            return (
              <div
                key={contrib.user_id}
                className="flex items-center justify-between rounded-lg border p-2"
              >
                <div className="flex items-center gap-2">
                  <span className="text-sm text-muted-foreground">{index + 1}.</span>
                  {user ? (
                    <>
                      <Avatar className="h-6 w-6">
                        <AvatarImage src={user.avatar_url} />
                        <AvatarFallback className="text-[10px]">
                          {user.nama?.charAt(0)}
                        </AvatarFallback>
                      </Avatar>
                      <span className="text-sm font-medium">{user.nama}</span>
                    </>
                  ) : (
                    <span className="text-sm text-muted-foreground">User ID: {contrib.user_id}</span>
                  )}
                  <span className="text-sm text-muted-foreground">-</span>
                  <span className="text-sm">{contrib.contribution}</span>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={() => handleRemoveContributor(contrib.user_id)}
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
