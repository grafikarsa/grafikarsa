'use client';

import { useState, useCallback, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useRouter, useSearchParams } from 'next/navigation';
import { usersApi, publicApi } from '@/lib/api';
import { UserCard } from '@/components/user/user-card';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Search, X, ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';

const ITEMS_PER_PAGE = 20;

function UsersSkeleton() {
  return (
    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="rounded-3xl border bg-card p-4">
          <Skeleton className="aspect-square w-full rounded-2xl" />
          <div className="mt-4 space-y-2">
            <Skeleton className="h-5 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
            <div className="flex gap-2 pt-2">
              <Skeleton className="h-5 w-16 rounded-full" />
              <Skeleton className="h-5 w-20 rounded-full" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

export default function UsersPage() {
  const router = useRouter();
  const searchParams = useSearchParams();

  // Read from URL params
  const searchQuery = searchParams.get('search') || '';
  const role = searchParams.get('role') || 'all';
  const jurusanKode = searchParams.get('jurusan') || 'all';
  const kelasNama = searchParams.get('kelas') || 'all';
  const page = parseInt(searchParams.get('page') || '1', 10);

  // Local state for search input (not auto-submit)
  const [searchInput, setSearchInput] = useState(searchQuery);

  // Fetch jurusan for filter
  const { data: jurusanData } = useQuery({
    queryKey: ['jurusan'],
    queryFn: () => publicApi.getJurusan(),
  });

  // Fetch kelas for filter
  const { data: kelasData } = useQuery({
    queryKey: ['kelas'],
    queryFn: () => publicApi.getKelas(),
  });

  const jurusanList = jurusanData?.data || [];
  const kelasList = kelasData?.data || [];

  // Find jurusan ID from kode for API call
  const selectedJurusan = useMemo(() => {
    if (jurusanKode === 'all') return null;
    return jurusanList.find((j) => j.kode === jurusanKode);
  }, [jurusanKode, jurusanList]);

  // Find kelas ID from nama for API call
  const selectedKelas = useMemo(() => {
    if (kelasNama === 'all') return null;
    return kelasList.find((k) => k.nama === kelasNama);
  }, [kelasNama, kelasList]);

  // Filter kelas list by selected jurusan
  const filteredKelasList = useMemo(() => {
    if (!selectedJurusan) return kelasList;
    return kelasList.filter((k) => k.jurusan.id === selectedJurusan.id);
  }, [kelasList, selectedJurusan]);

  // Update URL with new params
  const updateParams = useCallback(
    (updates: Record<string, string | null>, resetPage = true) => {
      const params = new URLSearchParams(searchParams.toString());

      Object.entries(updates).forEach(([key, value]) => {
        if (value === null || value === '' || value === 'all') {
          params.delete(key);
        } else {
          params.set(key, value);
        }
      });

      // Reset to page 1 when filters change (except when changing page itself)
      if (resetPage && !('page' in updates)) {
        params.delete('page');
      }

      const queryString = params.toString();
      router.push(queryString ? `/users?${queryString}` : '/users', { scroll: false });
    },
    [router, searchParams]
  );

  // Handle search submit
  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    updateParams({ search: searchInput || null });
  };

  // Handle jurusan change - also reset kelas
  const handleJurusanChange = (value: string) => {
    updateParams({ jurusan: value === 'all' ? null : value, kelas: null });
  };

  // Handle kelas change
  const handleKelasChange = (value: string) => {
    updateParams({ kelas: value === 'all' ? null : value });
  };

  // Handle role change
  const handleRoleChange = (value: string) => {
    updateParams({ role: value === 'all' ? null : value });
  };

  // Fetch users with filters and pagination
  const { data, isLoading, error } = useQuery({
    queryKey: ['users', searchQuery, role, selectedJurusan?.id, selectedKelas?.id, page],
    queryFn: () =>
      usersApi.getUsers({
        search: searchQuery || undefined,
        role: role !== 'all' ? role : undefined,
        jurusan_id: selectedJurusan?.id || undefined,
        kelas_id: selectedKelas?.id || undefined,
        page,
        limit: ITEMS_PER_PAGE,
      }),
    enabled: jurusanList.length > 0 || jurusanKode === 'all', // Wait for jurusan data if filtering
  });

  const users = data?.data || [];
  const meta = data?.meta;

  const hasActiveFilters =
    searchQuery || role !== 'all' || jurusanKode !== 'all' || kelasNama !== 'all';

  const clearFilters = () => {
    setSearchInput('');
    router.push('/users', { scroll: false });
  };

  const handlePageChange = (newPage: number) => {
    updateParams({ page: newPage > 1 ? newPage.toString() : null }, false);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <div className="container mx-auto px-4 pb-24 pt-20 md:px-12 lg:px-16 md:pt-24">
      {/* Header - Centered */}
      <div className="mx-auto mb-6 max-w-2xl text-center md:mb-8">
        <h1 className="mb-2 text-2xl font-bold md:text-4xl">Siswa & Alumni</h1>
        <p className="text-sm text-muted-foreground md:text-base">
          Temukan dan terhubung dengan warga SMKN 4 Malang
        </p>
      </div>

      {/* Search - Centered with submit button */}
      <form onSubmit={handleSearchSubmit} className="mx-auto mb-6 max-w-md">
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Cari nama atau username..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="rounded-full pl-11"
            />
          </div>
          <Button type="submit" className="rounded-full px-6">
            Cari
          </Button>
        </div>
      </form>

      {/* Filters - Grid on mobile, Flex on desktop */}
      <div className="mx-auto mb-8 grid grid-cols-2 gap-2 sm:flex sm:max-w-3xl sm:justify-center sm:gap-3">
        <div className="col-span-2 sm:col-span-1">
          <Select value={role} onValueChange={handleRoleChange}>
            <SelectTrigger className="w-full sm:w-[140px]">
              <SelectValue placeholder="Role" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua Role</SelectItem>
              <SelectItem value="student">Siswa</SelectItem>
              <SelectItem value="alumni">Alumni</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Select value={jurusanKode} onValueChange={handleJurusanChange}>
          <SelectTrigger className="w-full sm:w-[180px]">
            <SelectValue placeholder="Jurusan" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Jurusan</SelectItem>
            {jurusanList.map((j) => (
              <SelectItem key={j.id} value={j.kode}>
                {j.nama}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={kelasNama} onValueChange={handleKelasChange}>
          <SelectTrigger className="w-full sm:w-[160px]">
            <SelectValue placeholder="Kelas" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Kelas</SelectItem>
            {filteredKelasList.map((k) => (
              <SelectItem key={k.id} value={k.nama}>
                {k.nama}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {hasActiveFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters} className="col-span-2 h-10 gap-1 sm:col-span-1 sm:h-9">
            <X className="h-4 w-4" />
            Reset Filter
          </Button>
        )}
      </div>

      {/* Results count */}
      {!isLoading && meta && (
        <p className="mb-6 text-center text-sm text-muted-foreground">
          Menampilkan {users.length} dari {meta.total_count} user
          {meta.total_pages > 1 && ` • Halaman ${meta.current_page} dari ${meta.total_pages}`}
        </p>
      )}

      {/* Users Grid - 4 columns */}
      {isLoading ? (
        <UsersSkeleton />
      ) : error ? (
        <p className="text-center text-muted-foreground">
          Gagal memuat data. Silakan coba lagi.
        </p>
      ) : users.length === 0 ? (
        <div className="rounded-lg border border-dashed p-12 text-center">
          <p className="text-muted-foreground">
            {hasActiveFilters
              ? 'Tidak ada user yang ditemukan dengan filter ini.'
              : 'Belum ada user.'}
          </p>
          {hasActiveFilters && (
            <Button variant="link" onClick={clearFilters} className="mt-2">
              Reset filter
            </Button>
          )}
        </div>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3 xl:grid-cols-4">
            {users.map((user) => (
              <UserCard key={user.id} user={user} />
            ))}
          </div>

          {/* Pagination */}
          {meta && meta.total_pages > 1 && (
            <div className="mt-8 flex flex-wrap items-center justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(page - 1)}
                disabled={page <= 1}
                className="h-9 px-3"
              >
                <ChevronLeft className="mr-1 h-4 w-4" />
                <span className="hidden sm:inline">Sebelumnya</span>
                <span className="sm:hidden">Prev</span>
              </Button>

              {/* Mobile Page Indicator */}
              <span className="text-sm font-medium text-muted-foreground sm:hidden">
                {page} / {meta.total_pages}
              </span>

              {/* Desktop Pagination Numbers */}
              <div className="hidden items-center gap-1 sm:flex">
                {Array.from({ length: meta.total_pages }, (_, i) => i + 1)
                  .filter((p) => {
                    // Show first, last, current, and adjacent pages
                    return (
                      p === 1 ||
                      p === meta.total_pages ||
                      Math.abs(p - page) <= 1
                    );
                  })
                  .map((p, idx, arr) => {
                    // Add ellipsis if there's a gap
                    const showEllipsisBefore = idx > 0 && p - arr[idx - 1] > 1;
                    return (
                      <span key={p} className="flex items-center">
                        {showEllipsisBefore && (
                          <span className="px-2 text-muted-foreground">...</span>
                        )}
                        <Button
                          variant={p === page ? 'default' : 'outline'}
                          size="sm"
                          className="h-8 w-8 p-0"
                          onClick={() => handlePageChange(p)}
                        >
                          {p}
                        </Button>
                      </span>
                    );
                  })}
              </div>

              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(page + 1)}
                disabled={page >= (meta?.total_pages || 1)}
                className="h-9 px-3"
              >
                <span className="hidden sm:inline">Selanjutnya</span>
                <span className="sm:hidden">Next</span>
                <ChevronRight className="ml-1 h-4 w-4" />
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
