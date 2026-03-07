'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { motion, AnimatePresence } from 'framer-motion';
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
import { Home, Plus, Search, Users, FolderOpen, Shield, Sparkles } from 'lucide-react';
import { NotificationBell } from '@/components/notifications/notification-bell';

export function StudentSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user } = useAuthStore();
  const { setViewMode, sidebarCollapsed } = useUIStore();

  const hasAdminAccess =
    user?.role === 'admin' ||
    (user?.special_roles && user.special_roles.length > 0) ||
    (user?.capabilities && user.capabilities.length > 0);

  const aiEnabled = process.env.NEXT_PUBLIC_AI_FEATURES_ENABLED === 'true';

  const navItems = [
    { href: '/', label: 'Feed', icon: Home, exact: true },
    ...(aiEnabled ? [{ href: '/ai-ideas', label: 'AI Ide Proyek', icon: Sparkles }] : []),
    { href: `/${user?.username}/portfolios/new`, label: 'Buat Portofolio', icon: Plus, isPrimary: true },
  ];

  const isActive = (href: string, exact?: boolean) => {
    const basePath = href.split('#')[0];
    if (exact) return pathname === basePath;
    return pathname === basePath || (basePath !== '/' && pathname.startsWith(basePath));
  };

  const isSearchActive = pathname === '/users' || pathname === '/portfolios';

  return (
    <TooltipProvider delayDuration={0}>
      <motion.aside
        initial={false}
        animate={{
          width: sidebarCollapsed ? 64 : 240,
        }}
        transition={{
          type: 'spring',
          stiffness: 300,
          damping: 30,
          mass: 0.8,
        }}
        className="fixed left-0 top-0 z-40 flex h-screen flex-col border-r bg-muted/40 py-4"
      >
        {/* Notification Bell with Label */}
        <div className="px-2 mb-2">
          <div
            className={cn(
              'flex items-center rounded-lg transition-colors',
              sidebarCollapsed ? 'h-10 w-10 justify-center mx-auto' : 'h-10 w-full gap-3 px-3'
            )}
          >
            <NotificationBell />
            <AnimatePresence>
              {!sidebarCollapsed && (
                <motion.span
                  initial={{ opacity: 0, width: 0 }}
                  animate={{ opacity: 1, width: 'auto' }}
                  exit={{ opacity: 0, width: 0 }}
                  transition={{ duration: 0.2 }}
                  className="overflow-hidden whitespace-nowrap text-sm font-medium"
                >
                  Notifikasi
                </motion.span>
              )}
            </AnimatePresence>
          </div>
        </div>

        {/* Navigation - Centered */}
        <nav className="flex flex-1 flex-col items-center justify-center gap-2 px-2">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.href, item.exact);

            // Primary action button (Create Portfolio)
            if (item.isPrimary) {
              return (
                <Tooltip key={item.href} delayDuration={sidebarCollapsed ? 0 : 99999}>
                  <TooltipTrigger asChild>
                    <Link
                      href={item.href}
                      className={cn(
                        'relative flex items-center rounded-xl transition-colors my-4',
                        'bg-primary text-primary-foreground hover:bg-primary/90',
                        sidebarCollapsed ? 'h-11 w-11 justify-center' : 'h-11 w-full gap-3 px-4 justify-start'
                      )}
                    >
                      <Icon className="h-5 w-5 shrink-0" />
                      <AnimatePresence>
                        {!sidebarCollapsed && (
                          <motion.span
                            initial={{ opacity: 0, width: 0 }}
                            animate={{ opacity: 1, width: 'auto' }}
                            exit={{ opacity: 0, width: 0 }}
                            transition={{ duration: 0.2 }}
                            className="overflow-hidden whitespace-nowrap text-sm font-medium"
                          >
                            {item.label}
                          </motion.span>
                        )}
                      </AnimatePresence>
                    </Link>
                  </TooltipTrigger>
                  <TooltipContent side="right" className="font-medium">
                    {item.label}
                  </TooltipContent>
                </Tooltip>
              );
            }

            // Regular navigation items
            return (
              <Tooltip key={item.href} delayDuration={sidebarCollapsed ? 0 : 99999}>
                <TooltipTrigger asChild>
                  <Link
                    href={item.href}
                    className={cn(
                      'flex items-center rounded-lg transition-all',
                      sidebarCollapsed ? 'h-10 w-10 justify-center' : 'h-10 w-full gap-3 px-3',
                      active
                        ? 'bg-muted text-foreground'
                        : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                    )}
                  >
                    <Icon className="h-5 w-5 shrink-0" />
                    <AnimatePresence>
                      {!sidebarCollapsed && (
                        <motion.span
                          initial={{ opacity: 0, width: 0 }}
                          animate={{ opacity: 1, width: 'auto' }}
                          exit={{ opacity: 0, width: 0 }}
                          transition={{ duration: 0.2 }}
                          className="overflow-hidden whitespace-nowrap text-sm"
                        >
                          {item.label}
                        </motion.span>
                      )}
                    </AnimatePresence>
                  </Link>
                </TooltipTrigger>
                <TooltipContent side="right">{item.label}</TooltipContent>
              </Tooltip>
            );
          })}

          {/* Admin Panel Switcher */}
          {hasAdminAccess && (
            <Tooltip delayDuration={sidebarCollapsed ? 0 : 99999}>
              <TooltipTrigger asChild>
                <button
                  onClick={() => {
                    setViewMode('admin');
                    router.push('/admin');
                  }}
                  className={cn(
                    'flex items-center rounded-lg text-muted-foreground transition-all hover:bg-muted/60 hover:text-foreground',
                    sidebarCollapsed ? 'h-10 w-10 justify-center' : 'h-10 w-full gap-3 px-3'
                  )}
                >
                  <Shield className="h-5 w-5 shrink-0" />
                  <AnimatePresence>
                    {!sidebarCollapsed && (
                      <motion.span
                        initial={{ opacity: 0, width: 0 }}
                        animate={{ opacity: 1, width: 'auto' }}
                        exit={{ opacity: 0, width: 0 }}
                        transition={{ duration: 0.2 }}
                        className="overflow-hidden whitespace-nowrap text-sm"
                      >
                        Admin Panel
                      </motion.span>
                    )}
                  </AnimatePresence>
                </button>
              </TooltipTrigger>
              <TooltipContent side="right">Switch to Admin Panel</TooltipContent>
            </Tooltip>
          )}

          {/* Search Popover */}
          <Popover>
            <Tooltip delayDuration={sidebarCollapsed ? 0 : 99999}>
              <TooltipTrigger asChild>
                <PopoverTrigger asChild>
                  <button
                    className={cn(
                      'flex items-center rounded-lg transition-all',
                      sidebarCollapsed ? 'h-10 w-10 justify-center' : 'h-10 w-full gap-3 px-3',
                      isSearchActive
                        ? 'bg-muted text-foreground'
                        : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                    )}
                  >
                    <Search className="h-5 w-5 shrink-0" />
                    <AnimatePresence>
                      {!sidebarCollapsed && (
                        <motion.span
                          initial={{ opacity: 0, width: 0 }}
                          animate={{ opacity: 1, width: 'auto' }}
                          exit={{ opacity: 0, width: 0 }}
                          transition={{ duration: 0.2 }}
                          className="overflow-hidden whitespace-nowrap text-sm"
                        >
                          Cari
                        </motion.span>
                      )}
                    </AnimatePresence>
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
        <div className="px-2">
          <Tooltip delayDuration={sidebarCollapsed ? 0 : 99999}>
            <TooltipTrigger asChild>
              <Link
                href={`/${user?.username}`}
                className={cn(
                  'flex items-center rounded-lg transition-colors hover:opacity-80',
                  sidebarCollapsed ? 'h-10 w-10 justify-center' : 'h-10 w-full gap-3 px-2'
                )}
              >
                <Avatar className="h-9 w-9 cursor-pointer border-2 border-transparent transition-all hover:border-primary shrink-0">
                  <AvatarImage src={user?.avatar_url} alt={user?.nama} />
                  <AvatarFallback className="bg-primary text-sm font-medium text-primary-foreground">
                    {user?.nama?.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <AnimatePresence>
                  {!sidebarCollapsed && (
                    <motion.div
                      initial={{ opacity: 0, width: 0 }}
                      animate={{ opacity: 1, width: 'auto' }}
                      exit={{ opacity: 0, width: 0 }}
                      transition={{ duration: 0.2 }}
                      className="overflow-hidden min-w-0 flex-1"
                    >
                      <p className="truncate text-sm font-medium">{user?.nama}</p>
                      <p className="truncate text-xs text-muted-foreground">@{user?.username}</p>
                    </motion.div>
                  )}
                </AnimatePresence>
              </Link>
            </TooltipTrigger>
            <TooltipContent side="right">Profil Saya</TooltipContent>
          </Tooltip>
        </div>
      </motion.aside>
    </TooltipProvider>
  );
}
