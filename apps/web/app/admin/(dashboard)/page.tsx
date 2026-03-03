'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import Image from 'next/image';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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

  const stats = [
    {
      title: 'Total Users',
      value: data?.users.total ?? 0,
      icon: Users,
      color: 'text-blue-500',
      bg: 'bg-blue-500/10',
      href: '/admin/users',
    },
    {
      title: 'Total Portfolios',
      value: data?.portfolios.total ?? 0,
      icon: FolderOpen,
      color: 'text-green-500',
      bg: 'bg-green-500/10',
      href: '/admin/portfolios',
    },
    {
      title: 'Pending Review',
      value: data?.portfolios.pending_review ?? 0,
      icon: ClipboardCheck,
      color: 'text-orange-500',
      bg: 'bg-orange-500/10',
      href: '/admin/moderation',
    },
    {
      title: 'User Baru Bulan Ini',
      value: data?.users.new_this_month ?? 0,
      icon: TrendingUp,
      color: 'text-purple-500',
      bg: 'bg-purple-500/10',
      href: '/admin/users',
    },
  ];

  return (
    <div className="space-y-6">
      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <Link key={stat.title} href={stat.href}>
              <Card className="transition-shadow hover:shadow-md">
                <CardHeader className="flex flex-row items-center justify-between pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    {stat.title}
                  </CardTitle>
                  <div className={`rounded-full p-2 ${stat.bg}`}>
                    <Icon className={`h-4 w-4 ${stat.color}`} />
                  </div>
                </CardHeader>
                <CardContent>
                  {isLoading ? (
                    <Skeleton className="h-8 w-20" />
                  ) : (
                    <p className="text-2xl font-bold">{stat.value.toLocaleString()}</p>
                  )}
                </CardContent>
              </Card>
            </Link>
          );
        })}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Pending Review Quick Link */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <ClipboardCheck className="h-5 w-5 text-orange-500" />
              Portfolio Menunggu Review
            </CardTitle>
            <Link href="/admin/moderation">
              <Button variant="outline" size="sm">
                Lihat Semua
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </Link>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-3">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full" />
                ))}
              </div>
            ) : !data?.recent_pending_portfolios?.length ? (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <ClipboardCheck className="h-10 w-10 text-green-500" />
                <p className="mt-2 font-medium text-green-600">Tidak ada portfolio pending</p>
                <p className="text-sm text-muted-foreground">Semua portfolio sudah direview</p>
              </div>
            ) : (
              <div className="space-y-3">
                {data.recent_pending_portfolios.map((portfolio) => (
                  <Link
                    key={portfolio.id}
                    href="/admin/moderation"
                    className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted"
                  >
                    <div className="relative h-12 w-16 flex-shrink-0 overflow-hidden rounded bg-muted">
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
                      <p className="truncate font-medium">{portfolio.judul}</p>
                      <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <Avatar className="h-4 w-4">
                          <AvatarImage src={portfolio.user_avatar_url} />
                          <AvatarFallback className="text-[8px]">
                            {portfolio.user_nama?.charAt(0)}
                          </AvatarFallback>
                        </Avatar>
                        <span className="truncate">{portfolio.user_nama}</span>
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <Badge variant="outline" className="bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400">
                        Pending
                      </Badge>
                      <span className="flex items-center gap-1 text-xs text-muted-foreground">
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

        {/* Recent Users */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <UserPlus className="h-5 w-5 text-blue-500" />
              User Terbaru
            </CardTitle>
            <Link href="/admin/users">
              <Button variant="outline" size="sm">
                Lihat Semua
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </Link>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-3">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-14 w-full" />
                ))}
              </div>
            ) : !data?.recent_users?.length ? (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <Users className="h-10 w-10 text-muted-foreground" />
                <p className="mt-2 text-muted-foreground">Belum ada user</p>
              </div>
            ) : (
              <div className="space-y-3">
                {data.recent_users.map((user) => (
                  <Link
                    key={user.id}
                    href="/admin/users"
                    className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted"
                  >
                    <Avatar className="h-10 w-10">
                      <AvatarImage src={user.avatar_url} />
                      <AvatarFallback>{user.nama?.charAt(0)}</AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium">{user.nama}</p>
                      <p className="truncate text-sm text-muted-foreground">
                        @{user.username}
                        {user.kelas_nama && ` · ${user.kelas_nama}`}
                      </p>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <Badge className={`gap-1 text-xs capitalize ${roleStyles[user.role] || ''}`}>
                        {roleIcons[user.role]}
                        {user.role === 'student' ? 'Siswa' : user.role === 'alumni' ? 'Alumni' : 'Admin'}
                      </Badge>
                      <span className="flex items-center gap-1 text-xs text-muted-foreground">
                        <Clock className="h-3 w-3" />
                        {formatDate(user.created_at)}
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* System Info */}
      <Card>
        <CardHeader>
          <CardTitle>Ringkasan Sistem</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {[...Array(4)].map((_, i) => (
                <Skeleton key={i} className="h-20" />
              ))}
            </div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-lg border p-4">
                <p className="text-sm text-muted-foreground">Siswa Aktif</p>
                <p className="text-2xl font-bold">{data?.users.students.toLocaleString()}</p>
              </div>
              <div className="rounded-lg border p-4">
                <p className="text-sm text-muted-foreground">Alumni</p>
                <p className="text-2xl font-bold">{data?.users.alumni.toLocaleString()}</p>
              </div>
              <div className="rounded-lg border p-4">
                <p className="text-sm text-muted-foreground">Portfolio Published</p>
                <p className="text-2xl font-bold">{data?.portfolios.published.toLocaleString()}</p>
              </div>
              <div className="rounded-lg border p-4">
                <p className="text-sm text-muted-foreground">Jurusan</p>
                <p className="text-2xl font-bold">{data?.jurusan.total.toLocaleString()}</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
