'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import Image from 'next/image';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Users,
  FolderOpen,
  ClipboardCheck,
  TrendingUp,
  ArrowRight,
  Clock,
  UserPlus,
  GraduationCap,
  Shield,
  Activity,
  BarChart3,
  FileText,
} from 'lucide-react';
import api from '@/lib/api/client';
import { formatDate } from '@/lib/utils/format';

interface RecentUser {
  id: string;
  username: string;
  nama: string;
  avatar_url?: string;
  role: string;
  kelas_nama?: string;
  created_at: string;
}

interface RecentPendingPortfolio {
  id: string;
  judul: string;
  slug: string;
  thumbnail_url?: string;
  user_nama: string;
  user_username: string;
  user_avatar_url?: string;
  created_at: string;
}

interface DashboardStats {
  users: {
    total: number;
    students: number;
    alumni: number;
    admins: number;
    new_this_month: number;
  };
  portfolios: {
    total: number;
    published: number;
    pending_review: number;
    draft: number;
    rejected: number;
    archived: number;
    new_this_month: number;
  };
  jurusan: { total: number };
  kelas: { total: number; active_tahun_ajaran: number };
  recent_users: RecentUser[];
  recent_pending_portfolios: RecentPendingPortfolio[];
}

const roleStyles: Record<string, string> = {
  student: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  alumni: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  admin: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
};

const roleIcons: Record<string, React.ReactNode> = {
  student: <GraduationCap className="h-3 w-3" />,
  alumni: <Users className="h-3 w-3" />,
  admin: <Shield className="h-3 w-3" />,
};

export default function AdminDashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['admin-dashboard-stats'],
    queryFn: async () => {
      const response = await api.get<{ data: DashboardStats }>('/admin/dashboard/stats');
      return response.data.data;
    },
  });

  // Calculate pending assessment (published but not assessed)
  const pendingAssessment = (data?.portfolios.published ?? 0) - (data?.portfolios.published ?? 0); // TODO: Need actual assessed count from backend

  const stats = [
    {
      title: 'Pending Review',
      value: data?.portfolios.pending_review ?? 0,
      icon: ClipboardCheck,
      color: 'text-orange-500',
      bg: 'bg-orange-500/10',
      href: '/admin/moderation',
      urgent: (data?.portfolios.pending_review ?? 0) > 0,
    },
    {
      title: 'Pending Assessment',
      value: pendingAssessment,
      icon: BarChart3,
      color: 'text-blue-500',
      bg: 'bg-blue-500/10',
      href: '/admin/assessments',
      actionable: pendingAssessment > 0,
    },
    {
      title: 'Portfolio Published',
      value: data?.portfolios.published ?? 0,
      icon: FileText,
      color: 'text-green-500',
      bg: 'bg-green-500/10',
      href: '/admin/portfolios',
    },
    {
      title: 'Active Students',
      value: data?.users.students ?? 0,
      icon: GraduationCap,
      color: 'text-purple-500',
      bg: 'bg-purple-500/10',
      href: '/admin/users',
    },
    {
      title: 'Portfolio Bulan Ini',
      value: data?.portfolios.new_this_month ?? 0,
      icon: TrendingUp,
      color: 'text-indigo-500',
      bg: 'bg-indigo-500/10',
      href: '/admin/portfolios',
    },
  ];

  return (
    <div className="space-y-6">
      {/* Stats Cards - Compact */}
      <div className="grid gap-2 grid-cols-2 lg:grid-cols-5">
        {stats.map((stat) => {
          const Icon = stat.icon;
          const isUrgent = 'urgent' in stat && stat.urgent;
          const isActionable = 'actionable' in stat && stat.actionable;
          return (
            <Link key={stat.title} href={stat.href} className="group">
              <Card className={`transition-all hover:shadow-md ${isUrgent ? 'border-orange-500/50 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-500/30' : 'hover:border-primary/50'}`}>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-1 p-3">
                  <CardTitle className="text-xs font-medium text-muted-foreground group-hover:text-foreground transition-colors">
                    {stat.title}
                  </CardTitle>
                  <div className={`rounded-md p-1 ${stat.bg} dark:bg-opacity-20 group-hover:scale-110 transition-transform ${isUrgent ? 'animate-pulse' : ''}`}>
                    <Icon className={`h-4 w-4 ${stat.color} dark:opacity-90`} strokeWidth={2} />
                  </div>
                </CardHeader>
                <CardContent className="p-3 pt-0">
                  {isLoading ? (
                    <Skeleton className="h-7 w-14" />
                  ) : (
                    <div>
                      <p className="text-2xl font-bold tracking-tight leading-none">{stat.value.toLocaleString()}</p>
                      <p className="text-[10px] text-muted-foreground flex items-center gap-0.5 mt-1">
                        <ArrowRight className="h-2.5 w-2.5" />
                        {isUrgent ? 'Review' : isActionable ? 'Nilai' : 'Detail'}
                      </p>
                    </div>
                  )}
                </CardContent>
              </Card>
            </Link>
          );
        })}
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Pending Review - Priority Section */}
        <Card className="flex flex-col lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2.5 text-lg">
                <div className="rounded-lg bg-orange-500/10 p-2 dark:bg-orange-500/20">
                  <ClipboardCheck className="h-5 w-5 text-orange-500 dark:text-orange-400" strokeWidth={2} />
                </div>
                Portfolio Menunggu Review
              </CardTitle>
              <CardDescription>
                Prioritas utama - Review dan setujui portfolio
              </CardDescription>
            </div>
            <Link href="/admin/moderation">
              <Button size="sm" className="gap-2">
                Review Sekarang
                <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
          </CardHeader>
          <CardContent className="flex-1">
            {isLoading ? (
              <div className="space-y-3">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-20 w-full rounded-lg" />
                ))}
              </div>
            ) : !data?.recent_pending_portfolios?.length ? (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <div className="rounded-full bg-green-500/10 p-4 dark:bg-green-500/20">
                  <ClipboardCheck className="h-8 w-8 text-green-500 dark:text-green-400" />
                </div>
                <p className="mt-3 font-semibold text-green-600 dark:text-green-400">Tidak ada portfolio pending</p>
                <p className="text-sm text-muted-foreground mt-1">Semua portfolio sudah direview</p>
              </div>
            ) : (
              <div className="space-y-2">
                {data.recent_pending_portfolios.map((portfolio) => (
                  <Link
                    key={portfolio.id}
                    href="/admin/moderation"
                    className="flex items-center gap-3 rounded-lg border p-3 transition-all hover:bg-muted hover:border-primary/50"
                  >
                    <div className="relative h-14 w-20 flex-shrink-0 overflow-hidden rounded-md bg-muted">
                      {portfolio.thumbnail_url ? (
                        <Image
                          src={portfolio.thumbnail_url}
                          alt={portfolio.judul}
                          fill
                          className="object-cover"
                        />
                      ) : (
                        <div className="flex h-full items-center justify-center">
                          <FolderOpen className="h-5 w-5 text-muted-foreground" />
                        </div>
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-semibold text-sm">{portfolio.judul}</p>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground mt-1">
                        <Avatar className="h-4 w-4">
                          <AvatarImage src={portfolio.user_avatar_url} />
                          <AvatarFallback className="text-[8px]">
                            {portfolio.user_nama?.charAt(0)}
                          </AvatarFallback>
                        </Avatar>
                        <span className="truncate">{portfolio.user_nama}</span>
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-1.5">
                      <Badge variant="outline" className="bg-yellow-50 text-yellow-800 border-yellow-300 dark:bg-yellow-500/20 dark:text-yellow-300 dark:border-yellow-500/40">
                        Pending
                      </Badge>
                      <span className="flex items-center gap-1 text-xs text-muted-foreground whitespace-nowrap">
                        <Clock className="h-3 w-3" />
                        {formatDate(portfolio.created_at)}
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card className="flex flex-col">
          <CardHeader className="space-y-1 pb-4">
            <CardTitle className="flex items-center gap-2.5 text-lg">
              <div className="rounded-lg bg-primary/10 p-2">
                <Activity className="h-5 w-5 text-primary" strokeWidth={2} />
              </div>
              Quick Actions
            </CardTitle>
            <CardDescription>
              Akses cepat ke fitur admin
            </CardDescription>
          </CardHeader>
          <CardContent className="flex-1 space-y-2">
            <Link href="/admin/moderation" className="block">
              <Button variant="outline" className="w-full justify-start gap-3 h-auto py-3" size="lg">
                <div className="rounded-md bg-orange-500/10 p-2 dark:bg-orange-500/20">
                  <ClipboardCheck className="h-5 w-5 text-orange-500 dark:text-orange-400" />
                </div>
                <div className="flex-1 text-left">
                  <p className="font-semibold text-sm">Moderasi Portfolio</p>
                  <p className="text-xs text-muted-foreground">Review portfolio pending</p>
                </div>
                {!isLoading && data?.portfolios.pending_review ? (
                  <Badge variant="destructive" className="ml-auto">
                    {data.portfolios.pending_review}
                  </Badge>
                ) : null}
              </Button>
            </Link>

            <Link href="/admin/assessments" className="block">
              <Button variant="outline" className="w-full justify-start gap-3 h-auto py-3" size="lg">
                <div className="rounded-md bg-blue-500/10 p-2 dark:bg-blue-500/20">
                  <BarChart3 className="h-5 w-5 text-blue-500 dark:text-blue-400" />
                </div>
                <div className="flex-1 text-left">
                  <p className="font-semibold text-sm">Penilaian Portfolio</p>
                  <p className="text-xs text-muted-foreground">Beri nilai portfolio</p>
                </div>
              </Button>
            </Link>

            <Link href="/admin/users" className="block">
              <Button variant="outline" className="w-full justify-start gap-3 h-auto py-3" size="lg">
                <div className="rounded-md bg-green-500/10 p-2 dark:bg-green-500/20">
                  <Users className="h-5 w-5 text-green-500 dark:text-green-400" />
                </div>
                <div className="flex-1 text-left">
                  <p className="font-semibold text-sm">Kelola Users</p>
                  <p className="text-xs text-muted-foreground">Manage akun pengguna</p>
                </div>
              </Button>
            </Link>

            <Link href="/admin/portfolios" className="block">
              <Button variant="outline" className="w-full justify-start gap-3 h-auto py-3" size="lg">
                <div className="rounded-md bg-purple-500/10 p-2 dark:bg-purple-500/20">
                  <FolderOpen className="h-5 w-5 text-purple-500 dark:text-purple-400" />
                </div>
                <div className="flex-1 text-left">
                  <p className="font-semibold text-sm">Semua Portfolio</p>
                  <p className="text-xs text-muted-foreground">Lihat & edit portfolio</p>
                </div>
              </Button>
            </Link>

            <Link href="/admin/feedback" className="block">
              <Button variant="outline" className="w-full justify-start gap-3 h-auto py-3" size="lg">
                <div className="rounded-md bg-amber-500/10 p-2 dark:bg-amber-500/20">
                  <FileText className="h-5 w-5 text-amber-500 dark:text-amber-400" />
                </div>
                <div className="flex-1 text-left">
                  <p className="font-semibold text-sm">Feedback</p>
                  <p className="text-xs text-muted-foreground">Lihat feedback user</p>
                </div>
              </Button>
            </Link>
          </CardContent>
        </Card>
      </div>

      {/* Bottom Section - 2 Columns */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Recent Activity - 2 cols */}
        <Card className="lg:col-span-2">
          <CardHeader className="space-y-1 pb-4">
            <CardTitle className="flex items-center gap-2.5 text-lg">
              <div className="rounded-lg bg-primary/10 p-2">
                <Activity className="h-5 w-5 text-primary" strokeWidth={2} />
              </div>
              Recent Activity
            </CardTitle>
            <CardDescription>
              Aktivitas terbaru di platform
            </CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-3">
                {[...Array(5)].map((_, i) => (
                  <Skeleton key={i} className="h-16 rounded-lg" />
                ))}
              </div>
            ) : (
              <div className="space-y-2">
                {/* Recent Pending Portfolios */}
                {data?.recent_pending_portfolios?.slice(0, 2).map((portfolio) => (
                  <Link
                    key={portfolio.id}
                    href="/admin/moderation"
                    className="flex items-center gap-3 rounded-lg border p-3 transition-all hover:bg-muted hover:border-primary/50"
                  >
                    <div className="rounded-md bg-orange-500/10 p-2 dark:bg-orange-500/20">
                      <ClipboardCheck className="h-4 w-4 text-orange-500 dark:text-orange-400" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium text-sm">{portfolio.judul}</p>
                      <p className="text-xs text-muted-foreground">
                        {portfolio.user_nama} menunggu review
                      </p>
                    </div>
                    <Badge variant="outline" className="bg-orange-50 text-orange-800 border-orange-300 dark:bg-orange-500/20 dark:text-orange-300 dark:border-orange-500/40 text-xs">
                      Pending
                    </Badge>
                  </Link>
                ))}

                {/* Recent Users */}
                {data?.recent_users?.slice(0, 3).map((user) => (
                  <Link
                    key={user.id}
                    href="/admin/users"
                    className="flex items-center gap-3 rounded-lg border p-3 transition-all hover:bg-muted hover:border-primary/50"
                  >
                    <Avatar className="h-9 w-9 border">
                      <AvatarImage src={user.avatar_url} />
                      <AvatarFallback className="text-xs font-medium">{user.nama?.charAt(0)}</AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium text-sm">{user.nama}</p>
                      <p className="text-xs text-muted-foreground">
                        User baru mendaftar
                      </p>
                    </div>
                    <Badge className={`text-xs ${roleStyles[user.role] || ''}`}>
                      {user.role === 'student' ? 'Siswa' : user.role === 'alumni' ? 'Alumni' : 'Admin'}
                    </Badge>
                  </Link>
                ))}

                {/* Empty State */}
                {!data?.recent_pending_portfolios?.length && !data?.recent_users?.length && (
                  <div className="flex flex-col items-center justify-center py-8 text-center">
                    <div className="rounded-full bg-muted p-3">
                      <Activity className="h-6 w-6 text-muted-foreground" />
                    </div>
                    <p className="mt-2 text-sm font-medium text-muted-foreground">Belum ada aktivitas</p>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Portfolio Stats Chart - 1 col */}
        <Card>
          <CardHeader className="space-y-1 pb-4">
            <CardTitle className="flex items-center gap-2.5 text-lg">
              <div className="rounded-lg bg-primary/10 p-2">
                <BarChart3 className="h-5 w-5 text-primary" strokeWidth={2} />
              </div>
              Portfolio Stats
            </CardTitle>
            <CardDescription>
              Breakdown status portfolio
            </CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-3">
                {[...Array(5)].map((_, i) => (
                  <Skeleton key={i} className="h-12 rounded-lg" />
                ))}
              </div>
            ) : (
              <div className="space-y-3">
                {/* Published */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <div className="h-2 w-2 rounded-full bg-green-500 dark:bg-green-400" />
                      <span className="font-medium">Published</span>
                    </div>
                    <span className="font-bold">{data?.portfolios.published ?? 0}</span>
                  </div>
                  <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                    <div 
                      className="h-full bg-green-500 dark:bg-green-400 transition-all"
                      style={{ 
                        width: `${((data?.portfolios.published ?? 0) / (data?.portfolios.total || 1)) * 100}%` 
                      }}
                    />
                  </div>
                </div>

                {/* Pending Review */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <div className="h-2 w-2 rounded-full bg-orange-500 dark:bg-orange-400" />
                      <span className="font-medium">Pending Review</span>
                    </div>
                    <span className="font-bold">{data?.portfolios.pending_review ?? 0}</span>
                  </div>
                  <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                    <div 
                      className="h-full bg-orange-500 dark:bg-orange-400 transition-all"
                      style={{ 
                        width: `${((data?.portfolios.pending_review ?? 0) / (data?.portfolios.total || 1)) * 100}%` 
                      }}
                    />
                  </div>
                </div>

                {/* Draft */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <div className="h-2 w-2 rounded-full bg-gray-500 dark:bg-gray-400" />
                      <span className="font-medium">Draft</span>
                    </div>
                    <span className="font-bold">{data?.portfolios.draft ?? 0}</span>
                  </div>
                  <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                    <div 
                      className="h-full bg-gray-500 dark:bg-gray-400 transition-all"
                      style={{ 
                        width: `${((data?.portfolios.draft ?? 0) / (data?.portfolios.total || 1)) * 100}%` 
                      }}
                    />
                  </div>
                </div>

                {/* Rejected */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <div className="h-2 w-2 rounded-full bg-red-500 dark:bg-red-400" />
                      <span className="font-medium">Rejected</span>
                    </div>
                    <span className="font-bold">{data?.portfolios.rejected ?? 0}</span>
                  </div>
                  <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                    <div 
                      className="h-full bg-red-500 dark:bg-red-400 transition-all"
                      style={{ 
                        width: `${((data?.portfolios.rejected ?? 0) / (data?.portfolios.total || 1)) * 100}%` 
                      }}
                    />
                  </div>
                </div>

                {/* Archived */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <div className="h-2 w-2 rounded-full bg-gray-400 dark:bg-gray-500" />
                      <span className="font-medium">Archived</span>
                    </div>
                    <span className="font-bold">{data?.portfolios.archived ?? 0}</span>
                  </div>
                  <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
                    <div 
                      className="h-full bg-gray-400 dark:bg-gray-500 transition-all"
                      style={{ 
                        width: `${((data?.portfolios.archived ?? 0) / (data?.portfolios.total || 1)) * 100}%` 
                      }}
                    />
                  </div>
                </div>

                {/* Total */}
                <div className="pt-3 mt-3 border-t">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold">Total Portfolio</span>
                    <span className="text-xl font-bold">{data?.portfolios.total ?? 0}</span>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
