'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/lib/stores/auth-store';
import { profileApi } from '@/lib/api';
import { UserEditForm } from '@/components/user/user-edit-form';
import { Skeleton } from '@/components/ui/skeleton';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import Link from 'next/link';
import { useEffect } from 'react';

export default function EditProfilePage() {
  const params = useParams();
  const router = useRouter();
  const username = params.username as string;
  const { user: currentUser, isAuthenticated, isLoading: authLoading } = useAuthStore();

  // Use /me endpoint to get full profile data including email
  const { data, isLoading } = useQuery({
    queryKey: ['profile', 'me'],
    queryFn: () => profileApi.getMe(),
    enabled: isAuthenticated,
  });

  const profile = data?.data;
  const isOwner = currentUser?.username === username;

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push('/login');
    } else if (!authLoading && isAuthenticated && !isOwner && currentUser?.username) {
      router.replace(`/${currentUser.username}/edit`);
    }
  }, [authLoading, isAuthenticated, isOwner, username, router, currentUser?.username]);

  if (authLoading || isLoading) {
    return (
      <div className="mx-auto max-w-2xl space-y-6 px-4">
        <div className="flex items-center gap-4">
          <Skeleton className="h-10 w-10 rounded-full" />
          <Skeleton className="h-8 w-32" />
        </div>
        <Skeleton className="h-52 w-full rounded-xl" />
        <Skeleton className="h-64 w-full rounded-xl" />
      </div>
    );
  }

  if (!profile || !isOwner) {
    return null;
  }

  return (
    <div className="mx-auto max-w-2xl px-4 pb-12">
      {/* Header */}
      <div className="mb-6 flex items-center gap-3">
        <Link href={`/${username}`}>
          <Button variant="ghost" size="icon" className="h-9 w-9">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div>
          <h1 className="text-xl font-semibold">Edit Profil</h1>
          <p className="text-sm text-muted-foreground">Kelola informasi profil kamu</p>
        </div>
      </div>

      <UserEditForm user={profile} />
    </div>
  );
}
