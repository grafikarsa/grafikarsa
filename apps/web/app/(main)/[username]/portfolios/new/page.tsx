'use client';

import { useParams, useRouter } from 'next/navigation';
import { useEffect } from 'react';
import Link from 'next/link';
import { ArrowLeft, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { useAuthStore } from '@/lib/stores/auth-store';
import { PortfolioEditor } from '@/components/portfolio/portfolio-editor';

export default function CreatePortfolioPage() {
  const params = useParams();
  const router = useRouter();
  const username = params.username as string;
  const { user: currentUser, isAuthenticated, isLoading: authLoading } = useAuthStore();

  const isOwner = currentUser?.username === username;

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push('/login');
    } else if (!authLoading && isAuthenticated && !isOwner) {
      router.push(`/${username}`);
    }
  }, [authLoading, isAuthenticated, isOwner, username, router]);

  if (authLoading) {
    return (
      <div className="mx-auto max-w-3xl space-y-6 px-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="aspect-[21/9] w-full rounded-xl" />
        <Skeleton className="h-14 w-full rounded-xl" />
        <Skeleton className="h-32 w-full rounded-xl" />
      </div>
    );
  }

  if (!isOwner) {
    return null;
  }

  return (
    <div className="mx-auto max-w-3xl px-6">
      {/* Header */}
      <div className="mb-6 flex items-center gap-4">
        <Link href={`/${username}`}>
          <Button variant="ghost" size="icon" className="h-10 w-10 rounded-full">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div className="flex items-center gap-3">
          <div className="rounded-full bg-primary/10 p-2">
            <Plus className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h1 className="text-xl font-semibold">Buat Portfolio Baru</h1>
            <p className="text-sm text-muted-foreground">Tunjukkan karya terbaikmu</p>
          </div>
        </div>
      </div>

      <PortfolioEditor />
    </div>
  );
}
