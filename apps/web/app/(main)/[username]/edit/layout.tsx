'use client';

import { useParams, usePathname, useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/stores/auth-store';
import { Button } from '@/components/ui/button';
import { ArrowLeft, User, Lock, Link2, Settings } from 'lucide-react';
import Link from 'next/link';
import { useEffect } from 'react';
import { cn } from '@/lib/utils';

const settingsNav = [
  {
    title: 'Profil',
    href: '/edit',
    icon: User,
    description: 'Foto, nama, bio, dan informasi dasar',
  },
  {
    title: 'Akun',
    href: '/edit/account',
    icon: Settings,
    description: 'Username, email, dan pengaturan akun',
  },
  {
    title: 'Social Links',
    href: '/edit/social',
    icon: Link2,
    description: 'Link ke profil sosial media',
  },
  {
    title: 'Password',
    href: '/edit/password',
    icon: Lock,
    description: 'Ubah password akun',
  },
];

export default function EditProfileLayout({ children }: { children: React.ReactNode }) {
  const params = useParams();
  const pathname = usePathname();
  const router = useRouter();
  const username = params.username as string;
  const { user: currentUser, isAuthenticated, isLoading: authLoading } = useAuthStore();

  const isOwner = currentUser?.username === username;

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push('/login');
    } else if (!authLoading && isAuthenticated && !isOwner && currentUser?.username) {
      router.replace(`/${currentUser.username}/edit`);
    }
  }, [authLoading, isAuthenticated, isOwner, username, router, currentUser?.username]);

  if (authLoading || !isOwner) {
    return null;
  }

  return (
    <div className="container mx-auto max-w-7xl px-4 py-4 md:px-6 md:py-6 lg:px-8">
      {/* Header */}
      <div className="mb-4 flex items-center gap-3 md:mb-6">
        <Link href={`/${username}`}>
          <Button variant="ghost" size="icon" className="h-9 w-9">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div>
          <h1 className="text-xl font-semibold md:text-2xl">Pengaturan</h1>
        </div>
      </div>

      {/* Layout with Sidebar */}
      <div className="flex flex-col gap-4 lg:flex-row lg:gap-6">
        {/* Sidebar Navigation */}
        <aside className="w-full lg:w-64 lg:flex-shrink-0">
          <nav className="space-y-1">
            {settingsNav.map((item) => {
              const isActive = pathname === `/${username}${item.href}`;
              const Icon = item.icon;
              
              return (
                <Link
                  key={item.href}
                  href={`/${username}${item.href}`}
                  className={cn(
                    'flex items-start gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors',
                    isActive
                      ? 'bg-primary/10 text-primary font-medium'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  )}
                >
                  <Icon className="h-5 w-5 flex-shrink-0 mt-0.5" />
                  <div className="flex-1 min-w-0">
                    <div className="font-medium">{item.title}</div>
                    <div className="text-xs text-muted-foreground mt-0.5 hidden sm:block">
                      {item.description}
                    </div>
                  </div>
                </Link>
              );
            })}
          </nav>
        </aside>

        {/* Main Content */}
        <main className="flex-1 min-w-0">
          <div className="mx-auto max-w-2xl">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
