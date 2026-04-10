'use client';

import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/lib/stores/auth-store';
import { profileApi } from '@/lib/api';
import { Skeleton } from '@/components/ui/skeleton';
import { SocialLinksEditForm } from '@/components/user/edit/social-links-edit-form';

export default function EditSocialPage() {
  const params = useParams();
  const username = params.username as string;
  const { user: currentUser, isAuthenticated } = useAuthStore();

  const { data, isLoading } = useQuery({
    queryKey: ['profile', 'me'],
    queryFn: () => profileApi.getMe(),
    enabled: isAuthenticated,
  });

  const profile = data?.data;
  const isOwner = currentUser?.username === username;

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-64 w-full rounded-xl" />
      </div>
    );
  }

  if (!profile || !isOwner) {
    return null;
  }

  return <SocialLinksEditForm user={profile} />;
}
