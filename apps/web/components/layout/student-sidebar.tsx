'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/lib/stores/auth-store';
import { useUIStore } from '@/lib/stores/ui-store';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Home, Plus, Search, Users, FolderOpen, History, Shield, Send } from 'lucide-react';
import { getUnreadCount } from '@/lib/api/changelog';
import { NotificationBell } from '@/components/notifications/notification-bell';

function ChangelogNavItem({ pathname }: { pathname: string }) {
  const { data } = useQuery({
    queryKey: ['changelog-unread-count'],
    queryFn: () => getUnreadCount(),
    refetchInterval: 60000, // Refresh every minute
  });

  const unreadCount = data?.data?.data?.count || 0;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Link
          href="/changelog"
          className={cn(
            'relative flex h-10 w-10 items-center justify-center rounded-lg transition-all',
            pathname === '/changelog'
              ? 'bg-muted text-foreground'
              : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
          )}
        >
          <History className="h-5 w-5" />
          {unreadCount > 0 && (
            <Badge
              variant="destructive"
              className="absolute -right-1 -top-1 h-4 min-w-4 justify-center px-1 text-[10px]"
            >
              {unreadCount > 9 ? '9+' : unreadCount}
            </Badge>
          )}
        </Link>
      </TooltipTrigger>
      <TooltipContent side="right">
        Changelog {unreadCount > 0 && `(${unreadCount} baru)`}
      </TooltipContent>
    </Tooltip>
  );
}

export function StudentSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user } = useAuthStore();
  const { setViewMode } = useUIStore();

  const hasAdminAccess =
    user?.role === 'admin' ||
    (user?.special_roles && user.special_roles.length > 0) ||
    (user?.capabilities && user.capabilities.length > 0);

  const navItems = [
    { href: '/', label: 'Feed', icon: Home, exact: true },
    // { href: '/messages', label: 'Pesan', icon: Send },
    { href: `/${user?.username}/portfolios/new`, label: 'Buat Portofolio', icon: Plus },
  ];

  const isActive = (href: string, exact?: boolean) => {
    const basePath = href.split('#')[0];
    if (exact) return pathname === basePath;
    return pathname === basePath || (basePath !== '/' && pathname.startsWith(basePath));
  };

  const isSearchActive = pathname === '/users' || pathname === '/portfolios';

  return (
    <TooltipProvider delayDuration={0}>
      <aside className="fixed left-0 top-0 z-40 flex h-screen w-16 flex-col items-center border-r bg-muted/40 py-4">
        {/* Notification Bell - Top */}
        <div className="flex h-10 w-10 items-center justify-center">
          <NotificationBell />
        </div>

        {/* Navigation - Centered */}
        <nav className="flex flex-1 flex-col items-center justify-center gap-2">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.href, item.exact);

            return (
              <Tooltip key={item.href}>
                <TooltipTrigger asChild>
                  <Link
                    href={item.href}
                    className={cn(
                      'flex h-10 w-10 items-center justify-center rounded-lg transition-all',
                      active
                        ? 'bg-muted text-foreground'
                        : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                    )}
                  >
                    <Icon className="h-5 w-5" />
                  </Link>
                </TooltipTrigger>
                <TooltipContent side="right">{item.label}</TooltipContent>
              </Tooltip>
            );
          })}

          {/* Admin Panel Switcher */}
          {hasAdminAccess && (
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  onClick={() => {
                    setViewMode('admin');
                    router.push('/admin');
                  }}
                  className="flex h-10 w-10 items-center justify-center rounded-lg text-muted-foreground transition-all hover:bg-muted/60 hover:text-foreground"
                >
                  <Shield className="h-5 w-5" />
                </button>
              </TooltipTrigger>
              <TooltipContent side="right">Switch to Admin Panel</TooltipContent>
            </Tooltip>
          )}

          {/* Search Popover */}
          <Popover>
            <Tooltip>
              <TooltipTrigger asChild>
                <PopoverTrigger asChild>
                  <button
                    className={cn(
                      'flex h-10 w-10 items-center justify-center rounded-lg transition-all',
                      isSearchActive
                        ? 'bg-muted text-foreground'
                        : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                    )}
                  >
                    <Search className="h-5 w-5" />
                  </button>
                </PopoverTrigger>
              </TooltipTrigger>
              <TooltipContent side="right">Cari User & Portofolio</TooltipContent>
            </Tooltip>
            <PopoverContent side="right" align="center" className="w-48 p-2">
              <div className="space-y-1">
                <button
                  onClick={() => router.push('/users')}
                  className={cn(
                    'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors hover:bg-muted',
                    pathname === '/users' && 'bg-muted'
                  )}
                >
                  <Users className="h-4 w-4" />
                  <span>Cari User</span>
                </button>
                <button
                  onClick={() => router.push('/portfolios')}
                  className={cn(
                    'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors hover:bg-muted',
                    pathname === '/portfolios' && 'bg-muted'
                  )}
                >
                  <FolderOpen className="h-4 w-4" />
                  <span>Cari Portofolio</span>
                </button>
              </div>
            </PopoverContent>
          </Popover>
        </nav>

        {/* User Avatar - Above Changelog */}
        <Tooltip>
          <TooltipTrigger asChild>
            <Link
              href={`/${user?.username}`}
              className="flex h-10 w-10 items-center justify-center rounded-lg transition-colors hover:opacity-80"
            >
              <Avatar className="h-9 w-9 cursor-pointer border-2 border-transparent transition-all hover:border-primary">
                <AvatarImage src={user?.avatar_url} alt={user?.nama} />
                <AvatarFallback className="bg-primary text-sm font-medium text-primary-foreground">
                  {user?.nama?.charAt(0).toUpperCase()}
                </AvatarFallback>
              </Avatar>
            </Link>
          </TooltipTrigger>
          <TooltipContent side="right">Profil Saya</TooltipContent>
        </Tooltip>

        {/* Changelog - Bottom */}
        <ChangelogNavItem pathname={pathname} />
      </aside>
    </TooltipProvider>
  );
}
