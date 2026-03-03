'use client';

import { useAuthStore } from '@/lib/stores/auth-store';
import { useUIStore } from '@/lib/stores/ui-store';
import { GuestNavbar, StudentSidebar, StudentHeader, Footer } from '@/components/layout';
import { AdminSidebar } from '@/components/layout/admin-sidebar';
import { AdminHeader } from '@/components/layout/admin-header';
import { BottomNav } from '@/components/layout/bottom-nav';
import { Skeleton } from '@/components/ui/skeleton';

function LoadingScreen() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="space-y-4 text-center">
        <Skeleton className="mx-auto h-12 w-12 rounded-full" />
        <Skeleton className="mx-auto h-4 w-32" />
      </div>
    </div>
  );
}

export default function MainLayout({ children }: { children: React.ReactNode }) {
  const { user, isAuthenticated, isLoading } = useAuthStore();

  // Check if user has admin access
  const hasAdminAccess =
    user?.role === 'admin' ||
    (user?.special_roles && user.special_roles.length > 0) ||
    (user?.capabilities && user.capabilities.length > 0);

  // UI Store for view mode
  const { viewMode } = useUIStore();

  if (isLoading) {
    return <LoadingScreen />;
  }

  // Admin layout - show admin header/sidebar when browsing public pages
  // Only if viewMode is explicitly 'admin' (default)
  if (isAuthenticated && hasAdminAccess && viewMode === 'admin') {
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

  // Guest layout
  if (!isAuthenticated) {
    return (
      <div className="flex min-h-screen flex-col">
        <GuestNavbar />
        <main className="flex-1">{children}</main>
        <Footer />
      </div>
    );
  }

  // Authenticated student/alumni layout
  return (
    <div className="flex min-h-screen">
      {/* Desktop sidebar - hidden on mobile */}
      <div className="hidden md:block">
        <StudentSidebar />
      </div>
      <div className="flex flex-1 flex-col md:pl-16">
        <StudentHeader />
        {/* Add pb-20 on mobile for bottom nav clearance */}
        <main className="flex-1 p-4 pb-24 md:p-6 md:pb-6">{children}</main>
      </div>
      {/* Mobile bottom navigation */}
      <BottomNav />
    </div>
  );
}
