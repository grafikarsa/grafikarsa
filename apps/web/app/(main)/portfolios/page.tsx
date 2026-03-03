'use client';

import { useState, useCallback, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useRouter, useSearchParams } from 'next/navigation';
import { portfoliosApi, tagsApi, publicApi } from '@/lib/api';
import { seriesApi } from '@/lib/api/public';
import { PortfolioCard } from '@/components/portfolio/portfolio-card';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Search, X, ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

const ITEMS_PER_PAGE = 20;

function PortfoliosSkeleton() {
  return (
    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="space-y-3 rounded-lg border bg-card p-3">
          <Skeleton className="aspect-video w-full rounded-lg" />
          <Skeleton className="h-5 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
          <div className="flex gap-2">
            <Skeleton className="h-5 w-16 rounded-full" />
            <Skeleton className="h-5 w-20 rounded-full" />
          </div>
        </div>
      ))}
    </div>
  );
}

export default function PortfoliosPage() {
  const router = useRouter();
  const searchParams = useSearchParams();

  // Read from URL params
  const searchQuery = searchParams.get('search') || '';
  const jurusanKode = searchParams.get('jurusan') || 'all';
  const kelasNama = searchParams.get('kelas') || 'all';
  const seriesId = searchParams.get('series') || 'all';
  const sortBy = searchParams.get('sort') || '-published_at';
  const page = parseInt(searchParams.get('page') || '1', 10);
  const tagParam = searchParams.get('tags') || '';

  // Local state for search input
  const [searchInput, setSearchInput] = useState(searchQuery);

  // Parse selected tags from URL
  const selectedTags = useMemo(() => {
    return tagParam ? tagParam.split(',').filter(Boolean) : [];
  }, [tagParam]);

  // Fetch tags
  const { data: tagsData } = useQuery({
    queryKey: ['tags'],
    queryFn: () => tagsApi.getTags(),
  });

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

  // Fetch series for filter
  const { data: seriesData } = useQuery({
    queryKey: ['series'],
    queryFn: () => seriesApi.getSeries(),
  });

  const tags = tagsData?.data || [];
  const jurusanList = jurusanData?.data || [];
  const kelasList = kelasData?.data || [];
  const seriesList = seriesData?.data || [];

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

      if (resetPage && !('page' in updates)) {
        params.delete('page');
      }

      const queryString = params.toString();
      router.push(queryString ? `/portfolios?${queryString}` : '/portfolios', { scroll: false });
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

  // Handle tag toggle
  const toggleTag = (tagId: string) => {
    const newTags = selectedTags.includes(tagId)
      ? selectedTags.filter((id) => id !== tagId)
      : [...selectedTags, tagId];
    updateParams({ tags: newTags.length > 0 ? newTags.join(',') : null });
  };

  // Get selected series ID for API call
  const selectedSeriesId = seriesId === 'all' ? undefined : seriesId === 'none' ? 'none' : seriesId;

  // Fetch portfolios with filters and pagination
  const { data, isLoading, error } = useQuery({
    queryKey: ['portfolios', searchQuery, selectedJurusan?.id, selectedKelas?.id, selectedTags, selectedSeriesId, sortBy, page],
    queryFn: () =>
      portfoliosApi.getPortfolios({
        search: searchQuery || undefined,
        jurusan_id: selectedJurusan?.id || undefined,
        kelas_id: selectedKelas?.id || undefined,
        tag_ids: selectedTags.length > 0 ? selectedTags.join(',') : undefined,
        series_id: selectedSeriesId,
        sort: sortBy,
        page,
        limit: ITEMS_PER_PAGE,
      }),
    enabled: jurusanList.length > 0 || jurusanKode === 'all',
  });

  const portfolios = data?.data || [];
  const meta = data?.meta;

  const hasActiveFilters =
    searchQuery || jurusanKode !== 'all' || kelasNama !== 'all' || seriesId !== 'all' || selectedTags.length > 0;

  const clearFilters = () => {
    setSearchInput('');
    router.push('/portfolios', { scroll: false });
  };

  const handlePageChange = (newPage: number) => {
    updateParams({ page: newPage > 1 ? newPage.toString() : null }, false);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <div className="container mx-auto px-4 pb-24 pt-20 md:px-12 lg:px-16 md:pt-24">
      {/* Header - Centered */}
      <div className="mx-auto mb-6 max-w-2xl text-center md:mb-8">
        <h1 className="mb-2 text-2xl font-bold md:text-4xl">Katalog Portofolio</h1>
        <p className="text-sm text-muted-foreground md:text-base">
          Jelajahi karya-karya terbaik dari siswa dan alumni SMKN 4 Malang
        </p>
      </div>

      {/* Search - Centered with submit button */}
      <form onSubmit={handleSearchSubmit} className="mx-auto mb-6 max-w-md">
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Cari portofolio..."
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
      <div className="mx-auto mb-6 grid grid-cols-2 gap-2 sm:flex sm:max-w-4xl sm:justify-center sm:gap-3">
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

        <Select value={kelasNama} onValueChange={(v) => updateParams({ kelas: v === 'all' ? null : v })}>
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

        <Select value={seriesId} onValueChange={(v) => updateParams({ series: v === 'all' ? null : v })}>
          <SelectTrigger className="w-full sm:w-[180px]">
            <SelectValue placeholder="Series" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Series</SelectItem>
            <SelectItem value="none">Tanpa Series</SelectItem>
            {seriesList.map((s) => (
              <SelectItem key={s.id} value={s.id}>
                {s.nama}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={sortBy} onValueChange={(v) => updateParams({ sort: v === '-published_at' ? null : v })}>
          <SelectTrigger className="w-full sm:w-[160px]">
            <SelectValue placeholder="Urutkan" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="-published_at">Terbaru</SelectItem>
            <SelectItem value="-like_count">Terpopuler</SelectItem>
            <SelectItem value="judul">Judul A-Z</SelectItem>
          </SelectContent>
        </Select>

        {hasActiveFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters} className="col-span-2 h-10 gap-1 sm:col-span-1 sm:h-9">
            <X className="h-4 w-4" />
            Reset
          </Button>
        )}
      </div>

      {/* Tags Filter */}
      {tags.length > 0 && (
        <div className="mx-auto mb-8 flex max-w-4xl flex-wrap justify-center gap-2">
          {tags.map((tag) => (
            <Badge
              key={tag.id}
              variant={selectedTags.includes(tag.id) ? 'default' : 'outline'}
              className={cn('cursor-pointer', selectedTags.includes(tag.id) && 'pr-1')}
              onClick={() => toggleTag(tag.id)}
            >
              {tag.nama}
              {selectedTags.includes(tag.id) && <X className="ml-1 h-3 w-3" />}
            </Badge>
          ))}
        </div>
      )}

      {/* Results count */}
      {!isLoading && meta && (
        <p className="mb-6 text-center text-sm text-muted-foreground">
          Menampilkan {portfolios.length} dari {meta.total_count} portofolio
          {meta.total_pages > 1 && ` • Halaman ${meta.current_page} dari ${meta.total_pages}`}
        </p>
      )}

      {/* Portfolios Grid - 4 columns */}
      {isLoading ? (
        <PortfoliosSkeleton />
      ) : error ? (
        <p className="text-center text-muted-foreground">
          Gagal memuat data. Silakan coba lagi.
        </p>
      ) : portfolios.length === 0 ? (
        <div className="rounded-lg border border-dashed p-12 text-center">
          <p className="text-muted-foreground">
            {hasActiveFilters
              ? 'Tidak ada portofolio yang ditemukan dengan filter ini.'
              : 'Belum ada portofolio.'}
          </p>
          {hasActiveFilters && (
            <Button variant="link" onClick={clearFilters} className="mt-2">
              Reset filter
            </Button>
          )}
        </div>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
            {portfolios.map((portfolio) => (
              <PortfolioCard key={portfolio.id} portfolio={portfolio} />
            ))}
          </div>

          {/* Pagination */}
          {meta && meta.total_pages > 1 && (
            <div className="mt-10 flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(page - 1)}
                disabled={page <= 1}
              >
                <ChevronLeft className="h-4 w-4" />
                Sebelumnya
              </Button>

              <div className="flex items-center gap-1">
                {Array.from({ length: meta.total_pages }, (_, i) => i + 1)
                  .filter((p) => {
                    return (
                      p === 1 ||
                      p === meta.total_pages ||
                      Math.abs(p - page) <= 1
                    );
                  })
                  .map((p, idx, arr) => {
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
              >
                Selanjutnya
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
