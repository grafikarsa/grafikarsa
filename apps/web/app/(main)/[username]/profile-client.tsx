'use client';

import Link from 'next/link';
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
        refetchOnMount: 'always', // Always refetch to get latest follow state
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
                    <div className="grid grid-cols-[repeat(auto-fill,minmax(320px,320px))] gap-6">
                        {Array.from({ length: 6 }).map((_, i) => (
                            <div key={i} className="w-[320px] space-y-3 rounded-lg border p-3">
                                <Skeleton className="h-[240px] w-full rounded-lg" />
                                <Skeleton className="h-4 w-3/4" />
                                <Skeleton className="h-4 w-1/2" />
                            </div>
                        ))}
                    </div>
                ) : portfolios.length === 0 ? (
                    <div className="rounded-lg border border-dashed bg-muted/30 p-12 text-center">
                        {isOwner ? (
                            <>
                                <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
                                    <svg
                                        className="h-8 w-8 text-primary"
                                        fill="none"
                                        viewBox="0 0 24 24"
                                        stroke="currentColor"
                                        strokeWidth={2}
                                    >
                                        <path
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                            d="M12 4v16m8-8H4"
                                        />
                                    </svg>
                                </div>
                                <h3 className="mb-2 text-lg font-semibold">Belum ada portofolio</h3>
                                <p className="mb-6 text-sm text-muted-foreground">
                                    Mulai showcase karya terbaikmu dengan membuat portofolio pertama
                                </p>
                                <a
                                    href="/portfolios/new"
                                    className="inline-flex items-center justify-center rounded-md bg-primary px-6 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
                                >
                                    <svg
                                        className="mr-2 h-4 w-4"
                                        fill="none"
                                        viewBox="0 0 24 24"
                                        stroke="currentColor"
                                        strokeWidth={2}
                                    >
                                        <path
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                            d="M12 4v16m8-8H4"
                                        />
                                    </svg>
                                    Buat Portofolio
                                </a>
                            </>
                        ) : (
                            <>
                                <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                                    <svg
                                        className="h-8 w-8 text-muted-foreground"
                                        fill="none"
                                        viewBox="0 0 24 24"
                                        stroke="currentColor"
                                        strokeWidth={2}
                                    >
                                        <path
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                            d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
                                        />
                                    </svg>
                                </div>
                                <h3 className="mb-2 text-lg font-semibold">Belum ada portofolio</h3>
                                <p className="text-sm text-muted-foreground">
                                    {profile.nama} belum membuat portofolio
                                </p>
                            </>
                        )}
                    </div>
                ) : (
                    <div className="grid grid-cols-[repeat(auto-fill,minmax(320px,320px))] gap-6">
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
