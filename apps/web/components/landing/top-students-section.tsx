'use client';

import { useQuery } from '@tanstack/react-query';
import { Trophy } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { UserCard } from '@/components/user/user-card';
import { topApi } from '@/lib/api/public';
import { UserCard as UserCardType } from '@/lib/types';
import { cn } from '@/lib/utils';
import { RankingInfo } from '@/components/shared/ranking-info';

const rankStyles = {
  1: 'bg-amber-100 text-amber-700 border-amber-300 dark:bg-amber-900/30 dark:text-amber-400 dark:border-amber-700',
  2: 'bg-gray-100 text-gray-700 border-gray-300 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600',
  3: 'bg-orange-100 text-orange-700 border-orange-300 dark:bg-orange-900/30 dark:text-orange-400 dark:border-orange-700',
};

const rankLabels = {
  1: '# 1',
  2: '# 2',
  3: '# 3',
};

function StudentCardSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="mx-auto h-6 w-20 rounded-full" />
      <div className="rounded-xl border bg-card overflow-hidden">
        <Skeleton className="h-24" />
        <div className="p-4 pt-12 space-y-2">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-4 w-24" />
          <div className="flex gap-2 pt-2">
            <Skeleton className="h-5 w-16 rounded-full" />
            <Skeleton className="h-5 w-20 rounded-full" />
          </div>
        </div>
      </div>
    </div>
  );
}

export function TopStudentsSection() {
  const { data, isLoading } = useQuery({
    queryKey: ['top-students'],
    queryFn: () => topApi.getTopStudents(),
    staleTime: 5 * 60 * 1000,
  });

  const students = data?.data || [];

  if (isLoading) {
    return (
      <section className="py-16">
        <div className="mx-auto max-w-6xl px-6">
          <div className="mb-10 text-center">
            <div className="inline-flex items-center gap-2 rounded-full bg-primary/10 px-4 py-1.5 text-sm font-medium text-primary mb-4">
              <Trophy className="h-4 w-4" />
              Top Students
            </div>
            <h2 className="text-3xl font-semibold tracking-tight flex justify-center items-center gap-2">
              Siswa Terbaik
              <RankingInfo type="students" />
            </h2>
            <p className="mt-2 text-muted-foreground">Siswa dengan karya dan kontribusi terbaik</p>
          </div>
          <div className="grid gap-6 md:grid-cols-3">
            {[1, 2, 3].map((i) => (
              <StudentCardSkeleton key={i} />
            ))}
          </div>
        </div>
      </section>
    );
  }

  if (students.length === 0) {
    return null;
  }

  return (
    <section className="py-16">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mb-10 text-center">
          <div className="inline-flex items-center gap-2 rounded-full bg-primary/10 px-4 py-1.5 text-sm font-medium text-primary mb-4">
            <Trophy className="h-4 w-4" />
            Top Students
          </div>
          <h2 className="text-3xl font-semibold tracking-tight flex justify-center items-center gap-2">
            Siswa Terbaik
            <RankingInfo type="students" />
          </h2>
          <p className="mt-2 text-muted-foreground">Siswa dengan karya dan kontribusi terbaik</p>
        </div>
        <div className="grid gap-6 md:grid-cols-3">
          {students.map((student, index) => {
            const rank = (index + 1) as 1 | 2 | 3;
            // Convert TopStudent to UserCard type
            const userCardData: UserCardType = {
              id: student.id,
              username: student.username,
              nama: student.nama,
              avatar_url: student.avatar_url || undefined,
              banner_url: student.banner_url || undefined,
              role: 'student',
              kelas: student.kelas_nama ? { id: '', nama: student.kelas_nama } : undefined,
              jurusan: student.jurusan_nama ? { id: '', nama: student.jurusan_nama } : undefined,
            };

            return (
              <div key={student.id} className="relative">
                {/* Rank Badge */}
                <div className="flex justify-center mb-2">
                  <Badge variant="outline" className={cn('text-sm px-3 py-1 font-medium', rankStyles[rank])}>
                    {rankLabels[rank]}
                  </Badge>
                </div>
                <UserCard user={userCardData} />
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
