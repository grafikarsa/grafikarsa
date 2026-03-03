'use client';

import { useParams, useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { ArrowLeft, Pencil } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { useAuthStore } from '@/lib/stores/auth-store';
import { portfoliosApi } from '@/lib/api';
import { PortfolioEditor } from '@/components/portfolio/portfolio-editor';

export default function EditPortfolioPage() {
  const params = useParams();
  const router = useRouter();
  const username = params.username as string;
  const slug = params.slug as string;
  const { user: currentUser, isAuthenticated, isLoading: authLoading } = useAuthStore();

  const { data, isLoading } = useQuery({
    queryKey: ['portfolio', slug, username],
    queryFn: () => portfoliosApi.getPortfolioBySlug(slug, username),
    enabled: !!slug && !!username,
  });

  const portfolio = data?.data;
  const isOwner = currentUser?.username === username;

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push('/login');
    } else if (!authLoading && isAuthenticated && !isOwner) {
      router.push(`/${username}/${slug}`);
    }
  }, [authLoading, isAuthenticated, isOwner, username, slug, router]);

  if (authLoading || isLoading) {
    return (
      <div className="mx-auto max-w-3xl space-y-6 px-6">
        <div className="flex items-center gap-4">
          <Skeleton className="h-10 w-10 rounded-full" />
          <Skeleton className="h-8 w-48" />
        </div>
        <Skeleton className="aspect-[21/9] w-full rounded-xl" />
        <Skeleton className="h-14 w-full rounded-xl" />
        <Skeleton className="h-32 w-full rounded-xl" />
      </div>
    );
  }

  if (!portfolio || !isOwner) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center">
        <div className="text-center">
          <p className="text-muted-foreground">Portfolio tidak ditemukan</p>
          <Link href={`/${username}`}>
            <Button variant="link" className="mt-2">Kembali ke profil</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl px-6">
      {/* Header */}
      <div className="mb-6 flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          className="h-10 w-10 rounded-full"
          onClick={() => router.back()}
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div className="flex items-center gap-3">
          <div className="rounded-full bg-blue-500/10 p-2">
            <Pencil className="h-5 w-5 text-blue-500" />
          </div>
          <div>
            <h1 className="text-xl font-semibold">Edit Portfolio</h1>
            <p className="text-sm text-muted-foreground">Perbarui konten portfoliomu</p>
          </div>
        </div>
      </div>

      <PortfolioEditor portfolio={portfolio} isEdit />
    </div>
  );
}
