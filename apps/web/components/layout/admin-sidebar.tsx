'use client';

import Link from 'next/link';
import Image from 'next/image';
import { usePathname, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/lib/stores/auth-store';
import { useUIStore } from '@/lib/stores/ui-store';
import { useThemeValue } from '@/lib/hooks/use-theme-value';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Button } from '@/components/ui/button';
import {
  LayoutDashboard,
  Users,
  FolderOpen,
  ClipboardCheck,
  GraduationCap,
  School,
  Calendar,
  Tags,
  MessageSquare,
  Star,
  BarChart3,
  Layers,
  Shield,
  ArrowLeft,
  Upload,
  History,
} from 'lucide-react';
import api from '@/lib/api/client';

interface DashboardStats {
  portfolios: {
    pending_review: number;
  };
}

// Map href to capability key
const capabilityMap: Record<string, string> = {
  '/admin': 'dashboard',
  '/admin/portfolios': 'portfolios',
  '/admin/moderation': 'moderation',
  '/admin/assessments': 'assessments',
  '/admin/assessment-metrics': 'assessment_metrics',
  '/admin/tags': 'tags',
  '/admin/series': 'series',
  '/admin/users': 'users',
  '/admin/import': 'users',
  '/admin/special-roles': 'special_roles',
  '/admin/majors': 'majors',
  '/admin/classes': 'classes',
  '/admin/academic-years': 'academic_years',
  '/admin/feedback': 'feedback',
  '/admin/changelogs': 'changelog',
};

// Navigation item type
interface NavItem {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  iconColor: string;
  exact?: boolean;
  badge?: 'pending' | 'assessment';
}

interface NavSection {
  title: string;
  items: NavItem[];
}

// Navigation sections with grouped items and icon colors
const navSections: NavSection[] = [
  {
    title: 'Overview',
    items: [
      { href: '/admin', label: 'Dashboard', icon: LayoutDashboard, iconColor: 'text-blue-500', exact: true },
    ],
  },
  {
    title: 'Konten',
    items: [
      { href: '/admin/portfolios', label: 'Portfolios', icon: FolderOpen, iconColor: 'text-emerald-500' },
      { href: '/admin/moderation', label: 'Moderasi', icon: ClipboardCheck, iconColor: 'text-orange-500', badge: 'pending' },
      { href: '/admin/assessments', label: 'Penilaian', icon: Star, iconColor: 'text-yellow-500', badge: 'assessment' },
      { href: '/admin/assessment-metrics', label: 'Metrik Penilaian', icon: BarChart3, iconColor: 'text-indigo-500' },
      { href: '/admin/tags', label: 'Tags', icon: Tags, iconColor: 'text-pink-500' },
      { href: '/admin/series', label: 'Series', icon: Layers, iconColor: 'text-blue-500' },
    ],
  },
  {
    title: 'Pengguna',
    items: [
      { href: '/admin/users', label: 'Users', icon: Users, iconColor: 'text-violet-500' },
      { href: '/admin/import', label: 'Import Siswa', icon: Upload, iconColor: 'text-green-500' },
      { href: '/admin/special-roles', label: 'Special Roles', icon: Shield, iconColor: 'text-amber-500' },
    ],
  },
  {
    title: 'Akademik',
    items: [
      { href: '/admin/majors', label: 'Jurusan', icon: GraduationCap, iconColor: 'text-cyan-500' },
      { href: '/admin/classes', label: 'Kelas', icon: School, iconColor: 'text-amber-500' },
      { href: '/admin/academic-years', label: 'Tahun Ajaran', icon: Calendar, iconColor: 'text-rose-500' },
    ],
  },
  {
    title: 'Lainnya',
    items: [
      { href: '/admin/feedback', label: 'Feedback', icon: MessageSquare, iconColor: 'text-teal-500' },
      { href: '/admin/changelogs', label: 'Changelog', icon: History, iconColor: 'text-purple-500' },
    ],
  },
];

export function AdminSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user } = useAuthStore();
  const { setViewMode } = useUIStore();
  const { theme, mounted } = useThemeValue();

  // Check if user is full admin or has special roles
  const isFullAdmin = user?.role === 'admin';
  const userCapabilities = user?.capabilities || [];

  // Fetch pending count for moderation badge
  const { data: stats } = useQuery({
    queryKey: ['admin-sidebar-stats'],
    queryFn: async () => {
      const response = await api.get<{ data: DashboardStats }>('/admin/dashboard/stats');
      return response.data.data;
    },
    refetchInterval: 60000, // Refresh every minute
    enabled: isFullAdmin || userCapabilities.includes('dashboard') || userCapabilities.includes('moderation'),
  });

  // Fetch assessment stats for badge
  const { data: assessmentStats } = useQuery({
    queryKey: ['admin-sidebar-assessment-stats'],
    queryFn: async () => {
      const response = await api.get<{ data: { total_published: number; assessed: number; pending: number } }>('/admin/assessments/stats');
      return response.data.data;
    },
    refetchInterval: 60000,
    enabled: isFullAdmin || userCapabilities.includes('assessments'),
  });

  const pendingCount = stats?.portfolios?.pending_review || 0;
  const pendingAssessmentCount = assessmentStats?.pending || 0;

  const isActive = (href: string, exact?: boolean) => {
    if (exact) return pathname === href;
    return pathname === href || pathname.startsWith(href + '/');
  };

  // Check if user has access to a menu item
  const hasAccess = (href: string): boolean => {
    if (isFullAdmin) return true;
    const capability = capabilityMap[href];
    return capability ? userCapabilities.includes(capability) : false;
  };

  // Filter sections based on user capabilities
  const filteredSections = navSections
    .map((section) => ({
      ...section,
      items: section.items.filter((item) => hasAccess(item.href)),
    }))
    .filter((section) => section.items.length > 0);

  const logoSrc =
    theme === 'dark'
      ? '/images/logos/logo_white.svg'
      : '/images/logos/logo_black.svg';

  return (
    <aside className="fixed left-0 top-0 z-40 h-screen w-56 border-r bg-muted/40">
      <div className="flex h-full flex-col">
        {/* Logo/Brand */}
        <div className="flex h-14 items-center gap-2.5 border-b bg-background/50 px-4">
          {mounted && (
            <Image
              src={logoSrc}
              alt="Grafikarsa"
              width={32}
              height={32}
              className="h-8 w-8"
            />
          )}
          <div>
            <h1 className="text-sm font-semibold">Grafikarsa</h1>
            <p className="text-[10px] text-muted-foreground">
              {isFullAdmin ? 'Admin Panel' : 'Limited Access'}
            </p>
          </div>
        </div>

        {/* Limited Access Badge */}
        {!isFullAdmin && (
          <div className="border-b px-4 py-2">
            <Badge variant="outline" className="w-full justify-center text-xs">
              Akses Terbatas
            </Badge>
          </div>
        )}

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto px-2 py-3">
          {filteredSections.map((section, sectionIndex) => (
            <div key={section.title} className={cn(sectionIndex > 0 && 'mt-4')}>
              <h2 className="mb-1.5 px-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                {section.title}
              </h2>
              <div className="space-y-0.5">
                {section.items.map((item) => {
                  const Icon = item.icon;
                  const active = isActive(item.href, item.exact);
                  const showBadge = item.badge === 'pending' && pendingCount > 0;

                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={cn(
                        'group flex items-center gap-2.5 rounded-md px-2 py-1.5 text-xs font-medium transition-all',
                        active
                          ? 'bg-muted text-foreground'
                          : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
                      )}
                    >
                      <Icon className={cn('h-4 w-4 flex-shrink-0', item.iconColor)} />
                      <span className="flex-1">{item.label}</span>
                      {showBadge && (
                        <Badge
                          variant="destructive"
                          className="h-4 min-w-4 justify-center px-1 text-[10px]"
                        >
                          {pendingCount > 99 ? '99+' : pendingCount}
                        </Badge>
                      )}
                      {item.badge === 'assessment' && pendingAssessmentCount > 0 && (
                        <Badge
                          variant="destructive"
                          className="h-4 min-w-4 justify-center px-1 text-[10px]"
                        >
                          {pendingAssessmentCount > 9999 ? '9999+' : pendingAssessmentCount}
                        </Badge>
                      )}
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </nav>

        <Separator />

        {/* Switch to User App (for admins/special roles) */}
        <div className="border-b p-2">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-start gap-2 text-muted-foreground hover:text-foreground"
            onClick={() => {
              setViewMode('user');
              router.push('/');
            }}
          >
            <ArrowLeft className="h-4 w-4" />
            Switch to App View
          </Button>
        </div>

        {/* User Profile */}
        <div className="p-2">
          <div className="flex items-center gap-2.5 rounded-md bg-background/50 p-2">
            <Avatar className="h-8 w-8 border shadow-sm">
              <AvatarImage src={user?.avatar_url} alt={user?.nama} />
              <AvatarFallback className="bg-primary text-xs text-primary-foreground font-medium">
                {user?.nama?.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div className="flex-1 overflow-hidden">
              <p className="truncate text-xs font-medium">{user?.nama}</p>
              <p className="truncate text-[10px] text-muted-foreground">@{user?.username}</p>
            </div>
          </div>
        </div>
      </div>
    </aside>
  );
}
