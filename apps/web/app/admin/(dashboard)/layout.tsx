'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/stores/auth-store';
import { AdminSidebar } from '@/components/layout/admin-sidebar';
import { AdminHeader } from '@/components/layout/admin-header';
import { Skeleton } from '@/components/ui/skeleton';

export default function AdminDashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, isAuthenticated, isLoading } = useAuthStore();

  // Check if user has admin access (admin role OR has special roles with capabilities)
  const hasAdminAccess =
    user?.role === 'admin' ||
    (user?.special_roles && user.special_roles.length > 0) ||
    (user?.capabilities && user.capabilities.length > 0);

  useEffect(() => {
    if (!isLoading) {
      if (!isAuthenticated) {
        router.push('/admin/loginadmin');
      } else if (!hasAdminAccess) {
        router.push('/');
      }
    }
  }, [isLoading, isAuthenticated, hasAdminAccess, router]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="space-y-4 text-center">
          <Skeleton className="mx-auto h-12 w-12 rounded-full" />
          <Skeleton className="mx-auto h-4 w-32" />
        </div>
      </div>
    );
  }

  if (!isAuthenticated || !hasAdminAccess) {
    return null;
  }

  return (
    <div className="flex min-h-screen">
      <AdminSidebar />
      <div className="flex flex-1 flex-col pl-56">
        <AdminHeader />
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}
