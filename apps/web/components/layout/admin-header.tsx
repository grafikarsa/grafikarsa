'use client';

import { usePathname } from 'next/navigation';
import { LogOut } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useAuth } from '@/lib/hooks/use-auth';
import { ThemeToggle } from './theme-toggle';

// Map pathname to page title
const pageTitles: Record<string, string> = {
  '/admin': 'Dashboard',
  '/admin/users': 'Kelola Users',
  '/admin/portfolios': 'Kelola Portfolio',
  '/admin/moderation': 'Moderasi Portfolio',
  '/admin/assessments': 'Penilaian Portfolio',
  '/admin/assessment-metrics': 'Metrik Penilaian',
  '/admin/majors': 'Kelola Jurusan',
  '/admin/classes': 'Kelola Kelas',
  '/admin/academic-years': 'Kelola Tahun Ajaran',
  '/admin/tags': 'Kelola Tags',
  '/admin/series': 'Kelola Series',
  '/admin/special-roles': 'Special Roles',
  '/admin/changelogs': 'Kelola Changelog',
  '/admin/feedback': 'Feedback',
  '/admin/import': 'Import Siswa',
};

export function AdminHeader() {
  const pathname = usePathname();
  const { logout, isLogoutPending } = useAuth();

  // Get page title from pathname
  const getPageTitle = () => {
    // Exact match first
    if (pageTitles[pathname]) return pageTitles[pathname];
    
    // Check for partial matches (for nested routes)
    for (const [path, title] of Object.entries(pageTitles)) {
      if (pathname.startsWith(path + '/')) return title;
    }
    
    return 'Admin Panel';
  };

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b bg-background px-6">
      <h1 className="text-lg font-semibold">{getPageTitle()}</h1>
      
      <div className="flex items-center gap-2">
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
