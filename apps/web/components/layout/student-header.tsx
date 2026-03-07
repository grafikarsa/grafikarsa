'use client';

import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { LogOut, Settings, History } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/lib/hooks/use-auth';
import { useThemeValue } from '@/lib/hooks/use-theme-value';
import { ThemeToggle } from './theme-toggle';
import { NotificationBell } from '@/components/notifications/notification-bell';
import { getUnreadCount } from '@/lib/api/changelog';

const pageTitles: Record<string, string> = {
  '/': 'Feed',
  '/search': 'Search',
  '/portfolios': 'Explore Portofolio',
  '/users': 'Siswa & Alumni',
};

export function StudentHeader() {
  const pathname = usePathname();
  const { logout, isLogoutPending } = useAuth();
  const { theme, mounted } = useThemeValue();

  // Fetch changelog unread count
  const { data } = useQuery({
    queryKey: ['changelog-unread-count'],
    queryFn: () => getUnreadCount(),
    refetchInterval: 60000,
  });

  const unreadCount = data?.data?.data?.count || 0;

  const getTitle = () => {
    if (pageTitles[pathname]) return pageTitles[pathname];
    if (pathname.includes('/edit')) return 'Edit Profil';
    if (pathname.includes('/portfolios/new')) return 'Buat Portofolio';
    if (pathname.includes('/followers')) return 'Followers';
    if (pathname.includes('/following')) return 'Following';
    if (pathname.includes('/settings')) return 'Pengaturan';
    return '';
  };

  const title = getTitle();

  const logoSrc =
    theme === 'dark'
      ? '/images/logos/logo_white.svg'
      : '/images/logos/logo_black.svg';

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b bg-background px-4 md:px-6">
      {/* Mobile: Logo on left */}
      <div className="flex items-center gap-2 md:hidden">
        <Link href="/" className="flex items-center gap-2">
          {mounted && (
            <Image
              src={logoSrc}
              alt="Grafikarsa"
              width={24}
              height={24}
              className="h-6 w-6"
            />
          )}
          <span className="font-semibold">Grafikarsa</span>
        </Link>
      </div>

      {/* Desktop: Page title on left */}
      <h1 className="hidden text-lg font-semibold md:block">{title || 'Grafikarsa'}</h1>

      {/* Desktop: Center Logo */}
      <Link
        href="/"
        className="absolute left-1/2 top-1/2 hidden -translate-x-1/2 -translate-y-1/2 items-center gap-2 md:flex"
      >
        {mounted && (
          <Image
            src={logoSrc}
            alt="Grafikarsa"
            width={24}
            height={24}
            className="h-6 w-6"
          />
        )}
        <span className="font-semibold">Grafikarsa</span>
      </Link>

      {/* Mobile: Changelog + Theme + Notification on right */}
      <div className="flex items-center gap-1 md:hidden">
        <Link
          href="/changelog"
          className="relative flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
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
        <div className="flex h-9 w-9 items-center justify-center">
          <ThemeToggle />
        </div>
        <NotificationBell />
      </div>

      {/* Desktop: Theme toggle + Logout on right */}
      <div className="hidden items-center gap-2 md:flex">
        <ThemeToggle />
        <Button
          variant="ghost"
          size="sm"
          onClick={() => logout()}
          disabled={isLogoutPending}
          className="gap-2 text-muted-foreground hover:text-destructive"
        >
          <LogOut className="h-4 w-4" />
          <span className="hidden sm:inline">Logout</span>
        </Button>
      </div>
    </header>
  );
}
