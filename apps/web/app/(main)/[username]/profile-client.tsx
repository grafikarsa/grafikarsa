'use client';

import { useQuery } from '@tanstack/react-query';
import { usersApi, portfoliosApi } from '@/lib/api';
import { useAuthStore } from '@/lib/stores/auth-store';
import { UserProfile } from '@/components/user/user-profile';
import { PortfolioCard } from '@/components/portfolio/portfolio-card';
import { Skeleton } from '@/components/ui/skeleton';
import { User } from '@/lib/types';
import { notFound } from 'next/navigation';

interface ProfileClientProps {
    username: string;
    initialData: User | null;
}

function ProfileSkeleton() {
    return (
        <div className="space-y-6">
            <Skeleton className="h-48 w-full rounded-lg md:h-64" />
            <div className="px-4">
                <Skeleton className="h-32 w-32 rounded-full" />
                <div className="mt-8 space-y-2">
                    <Skeleton className="h-8 w-48" />
                    <Skeleton className="h-4 w-32" />
                </div>
            </div>
        </div>
    );
}

export function ProfileClient({ username, initialData }: ProfileClientProps) {
    const { user: currentUser } = useAuthStore();
    const isOwner = currentUser?.username === username;

    const { data: userData, isLoading: userLoading } = useQuery({
        queryKey: ['user', username],
        queryFn: () => usersApi.getUserByUsername(username),
        initialData: initialData ? { data: initialData, message: '', success: true } : undefined,
    });

    const { data: portfoliosData, isLoading: portfoliosLoading } = useQuery({
        queryKey: ['user-portfolios', username, isOwner],
        queryFn: () =>
            isOwner
                ? portfoliosApi.getMyPortfolios({ limit: 50 })
                : portfoliosApi.getPortfolios({ user_id: userData?.data?.id, limit: 50 }),
        enabled: !!userData?.data,
    });

    if (userLoading && !(userData as any)?.data) {
        return <ProfileSkeleton />;
    }

    const profile = userData?.data || initialData;

    if (!profile) {
        notFound();
        return null;
    }

    const portfolios = portfoliosData?.data || [];

    return (
        <div>
            <UserProfile profile={profile} />

            {/* Portfolios Section */}
            <div className="container mx-auto max-w-5xl px-6 pb-12 md:px-12 lg:px-16" id="portfolios">
                <h2 className="mb-6 text-xl font-semibold">Portofolio</h2>

                {portfoliosLoading ? (
                    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
                        {Array.from({ length: 6 }).map((_, i) => (
                            <div key={i} className="space-y-3 rounded-lg border p-3">
                                <Skeleton className="aspect-[4/3] w-full rounded-lg" />
                                <Skeleton className="h-4 w-3/4" />
                                <Skeleton className="h-4 w-1/2" />
                            </div>
                        ))}
                    </div>
                ) : portfolios.length === 0 ? (
                    <div className="rounded-lg border border-dashed p-12 text-center">
                        <p className="text-muted-foreground">Belum ada portofolio.</p>
                    </div>
                ) : (
                    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
                        {portfolios.map((portfolio) => (
                            <PortfolioCard
                                key={portfolio.id}
                                portfolio={portfolio}
                                showStatus={isOwner}
                                showActions={isOwner}
                                username={username}
                            />
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}
