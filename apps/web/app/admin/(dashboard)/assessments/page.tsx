'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Search,
  Star,
  CheckCircle2,
  Clock,
  Loader2,
  Eye,
  ClipboardList,
  AlertCircle,
  X,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Slider } from '@/components/ui/slider';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { DebugBanner } from '@/components/admin/debug-banner';
import { adminAssessmentsApi, adminAssessmentMetricsApi } from '@/lib/api/admin';
import {
  PortfolioForAssessment,
  AssessmentMetric,
  AssessmentResponse,
  ScoreInput,
} from '@/lib/types';
import { useDebounce } from '@/lib/hooks/use-debounce';
import { getDebugEmptyState } from '@/lib/utils/debug';
import { cn } from '@/lib/utils';

type FilterType = 'all' | 'pending' | 'assessed';

const filterOptions = [
  { value: 'all', label: 'Semua Portfolio' },
  { value: 'pending', label: 'Belum Dinilai' },
  { value: 'assessed', label: 'Sudah Dinilai' },
];

export default function AdminAssessmentsPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<FilterType>('all');
  const [page, setPage] = useState(1);
  const [selectedPortfolio, setSelectedPortfolio] = useState<PortfolioForAssessment | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<PortfolioForAssessment | null>(null);
  const debouncedSearch = useDebounce(search, 300);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin-assessments', debouncedSearch, filter, page],
    queryFn: () =>
      adminAssessmentsApi.getPortfolios({
        search: debouncedSearch || undefined,
        filter: filter === 'all' ? undefined : filter,
        page,
        limit: 20,
      }),
  });

  // Fetch stats for accurate counts
  const { data: statsData } = useQuery({
    queryKey: ['admin-assessment-stats'],
    queryFn: () => adminAssessmentsApi.getStats(),
  });

  const deleteMutation = useMutation({
    mutationFn: (portfolioId: string) => adminAssessmentsApi.deleteAssessment(portfolioId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-assessments'] });
      toast.success('Penilaian berhasil dihapus');
      setDeleteTarget(null);
    },
    onError: () => toast.error('Gagal menghapus penilaian'),
  });

  const portfolios = data?.data || [];
  const pagination = data?.meta;
  const stats = statsData?.data;

  // Stats - use stats endpoint for accurate totals
  const totalCount = stats?.total_published || pagination?.total_count || 0;
  const assessedCount = stats?.assessed || 0;
  const pendingCount = stats?.pending || 0;

  // Debug mode: Force empty state
  const debugMode = getDebugEmptyState();
  const displayPortfolios = debugMode ? [] : portfolios;

  if (isLoading) {
    return (
      <div className="flex flex-col -m-4 sm:-m-6 lg:-m-8">
        {/* Sticky Header Skeleton */}
        <div className="sticky top-0 z-10 bg-background/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6 lg:px-8 border-b">
          <div className="mx-auto w-full max-w-[1600px] space-y-4">
            <div className="grid gap-4 sm:grid-cols-3">
              {[...Array(3)].map((_, i) => (
                <Skeleton key={i} className="h-24" />
              ))}
            </div>
            <div className="flex flex-col gap-4 sm:flex-row">
              <Skeleton className="h-10 flex-1" />
              <Skeleton className="h-10 w-full sm:w-48" />
            </div>
          </div>
        </div>
        <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-[1600px]">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {[...Array(6)].map((_, i) => (
                <Skeleton key={i} className="h-80" />
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-6">
        <Card className="border-destructive/50 bg-destructive/5 p-8">
          <div className="flex flex-col items-center justify-center text-center">
            <AlertCircle className="h-12 w-12 text-destructive" />
            <h3 className="mt-4 text-lg font-semibold">Gagal Memuat Data</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Terjadi kesalahan saat mengambil data portfolio
            </p>
            <Button onClick={() => refetch()} className="mt-4">
              Coba Lagi
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col -m-4 sm:-m-6 lg:-m-8">
      {/* Sticky Header - Stats & Filters (Full Width) */}
      <div className="sticky top-0 z-10 bg-background/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6 lg:px-8 border-b">
        <div className="mx-auto w-full max-w-[1600px] space-y-4">
          {/* Stats Cards */}
          <div className="grid gap-4 sm:grid-cols-3">
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-muted p-2">
                  <ClipboardList className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{totalCount}</p>
                  <p className="text-sm text-muted-foreground">Total Portfolio</p>
                </div>
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
                  <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{assessedCount}</p>
                  <p className="text-sm text-muted-foreground">Sudah Dinilai</p>
                </div>
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-yellow-100 p-2 dark:bg-yellow-900/30">
                  <Clock className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{pendingCount}</p>
                  <p className="text-sm text-muted-foreground">Belum Dinilai</p>
                </div>
              </div>
            </Card>
          </div>

          {/* Filters */}
          <div className="flex flex-col gap-4 sm:flex-row">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Cari portfolio..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={filter} onValueChange={(v) => setFilter(v as FilterType)}>
              <SelectTrigger className="w-full sm:w-48">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {filterOptions.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      {/* Content Area with proper padding */}
      <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
        <div className="mx-auto w-full max-w-[1600px]">
          {/* Debug Banner */}
          {debugMode && <DebugBanner pageName="Penilaian Portfolio" />}

          {/* Portfolio List */}
          {displayPortfolios.length === 0 ? (
            <div className="flex items-center justify-center min-h-[60vh]">
              <div className="flex flex-col items-center justify-center px-6">
                <div className="rounded-full bg-primary/10 p-4">
                  <ClipboardList className="h-10 w-10 text-primary" />
                </div>
                <h3 className="mt-6 text-xl font-semibold">
                  {search || filter !== 'all' ? 'Tidak ada portfolio yang sesuai' : 'Belum ada portfolio untuk dinilai'}
                </h3>
                <p className="mt-2 text-sm text-muted-foreground text-center max-w-md">
                  {search || filter !== 'all'
                    ? 'Coba ubah filter atau kata kunci pencarian untuk menemukan portfolio yang Anda cari.'
                    : 'Portfolio yang sudah dipublish akan muncul di sini untuk dinilai. Tunggu hingga ada portfolio baru yang dipublish oleh siswa.'}
                </p>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 md:gap-6">
              {displayPortfolios.map((portfolio) => (
                <PortfolioCard
                  key={portfolio.id}
                  portfolio={portfolio}
                  onAssess={() => setSelectedPortfolio(portfolio)}
                  onDelete={() => setDeleteTarget(portfolio)}
                />
              ))}
            </div>
          )}

          {/* Pagination */}
          {pagination && pagination.total_pages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-8">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                Previous
              </Button>
              <span className="text-sm text-muted-foreground">
                Halaman {page} dari {pagination.total_pages}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
                disabled={page === pagination.total_pages}
              >
                Next
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Assessment Sheet */}
      <AssessmentSheet
        portfolio={selectedPortfolio}
        onClose={() => setSelectedPortfolio(null)}
      />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Hapus Penilaian"
        description={
          <>
            Yakin ingin menghapus penilaian untuk portfolio{' '}
            <strong>&quot;{deleteTarget?.judul}&quot;</strong>?
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  );
}


// Portfolio Card Component
function PortfolioCard({
  portfolio,
  onAssess,
  onDelete,
}: {
  portfolio: PortfolioForAssessment;
  onAssess: () => void;
  onDelete: () => void;
}) {
  const hasAssessment = !!portfolio.assessment;

  return (
    <Card className="group w-[320px] gap-0 overflow-hidden border py-0 transition-shadow hover:shadow-lg">
      <div className="p-3 pb-4">
        {/* Thumbnail */}
        <div className="relative h-[240px] w-full overflow-hidden rounded-xl bg-muted">
          {portfolio.thumbnail_url ? (
            <img
              src={portfolio.thumbnail_url}
              alt={portfolio.judul}
              className="h-full w-full object-cover transition-transform group-hover:scale-105"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center">
              <ClipboardList className="h-12 w-12 text-muted-foreground/50" />
            </div>
          )}
          {/* Score Ribbon - Bookmark style with color based on score */}
          {hasAssessment && (
            <div className="absolute right-3 -top-0.5">
              <div className="relative">
                {/* Ribbon body */}
                <div 
                  className={cn(
                    "px-2.5 pt-2 pb-4 shadow-lg flex flex-col items-center",
                    (portfolio.assessment?.total_score ?? 0) < 4 
                      ? "bg-red-500" 
                      : (portfolio.assessment?.total_score ?? 0) <= 7 
                        ? "bg-amber-500" 
                        : "bg-green-500"
                  )}
                  style={{
                    clipPath: 'polygon(0 0, 100% 0, 100% 100%, 50% 85%, 0 100%)',
                  }}
                >
                  <Star className="h-4 w-4 text-white fill-white" />
                  <span className="text-white text-sm font-bold mt-0.5">
                    {portfolio.assessment?.total_score?.toFixed(1) || '-'}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Assessment Status Badge */}
        <div className="mt-3 flex items-center justify-between gap-2">
          <div />
          {hasAssessment ? (
            <Badge 
              variant="secondary" 
              className="rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-normal text-green-700 dark:bg-green-900/30 dark:text-green-400"
            >
              <CheckCircle2 className="mr-1 h-3 w-3" />
              Sudah Dinilai
            </Badge>
          ) : (
            <Badge 
              variant="secondary" 
              className="rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs font-normal text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400"
            >
              <Clock className="mr-1 h-3 w-3" />
              Belum Dinilai
            </Badge>
          )}
        </div>

        {/* Title */}
        <h3 className="mt-2 line-clamp-2 font-semibold leading-tight">{portfolio.judul}</h3>

        {/* User Info */}
        {portfolio.user && (
          <div className="mt-3 flex items-center gap-2">
            <Avatar className="h-8 w-8">
              <AvatarImage src={portfolio.user.avatar_url} alt={portfolio.user.nama} />
              <AvatarFallback className="text-xs">{portfolio.user.nama?.charAt(0)}</AvatarFallback>
            </Avatar>
            <div className="flex flex-col">
              <span className="text-sm font-medium leading-tight">{portfolio.user.nama}</span>
              <span className="text-xs text-muted-foreground">@{portfolio.user.username}</span>
            </div>
          </div>
        )}

        {/* Actions */}
        <div className="mt-4 flex gap-2">
          <Button size="sm" className="flex-1" onClick={onAssess}>
            <Star className="mr-1.5 h-4 w-4" />
            {hasAssessment ? 'Edit Penilaian' : 'Nilai Portfolio'}
          </Button>
          {hasAssessment && (
            <Button size="sm" variant="destructive" onClick={onDelete}>
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
    </Card>
  );
}


// Assessment Sheet Component
function AssessmentSheet({
  portfolio,
  onClose,
}: {
  portfolio: PortfolioForAssessment | null;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [scores, setScores] = useState<Record<string, { score: number; comment: string }>>({});
  const [finalComment, setFinalComment] = useState('');

  // Fetch metrics
  const { data: metricsData } = useQuery({
    queryKey: ['admin-assessment-metrics-active'],
    queryFn: () => adminAssessmentMetricsApi.getMetrics({ active_only: true }),
    enabled: !!portfolio,
  });

  // Fetch existing assessment
  const { data: assessmentData, isLoading: isLoadingAssessment } = useQuery({
    queryKey: ['admin-assessment', portfolio?.id],
    queryFn: () => adminAssessmentsApi.getAssessment(portfolio!.id),
    enabled: !!portfolio,
  });

  const metrics = metricsData?.data || [];
  const existingAssessment = assessmentData?.data?.assessment;

  // Initialize scores when data loads
  React.useEffect(() => {
    if (metrics.length > 0) {
      const initialScores: Record<string, { score: number; comment: string }> = {};
      metrics.forEach((metric) => {
        const existingScore = existingAssessment?.scores?.find(
          (s) => s.metric_id === metric.id
        );
        initialScores[metric.id] = {
          score: existingScore?.score || 5,
          comment: existingScore?.comment || '',
        };
      });
      setScores(initialScores);
      setFinalComment(existingAssessment?.final_comment || '');
    }
  }, [metrics, existingAssessment]);

  const submitMutation = useMutation({
    mutationFn: () => {
      const scoreInputs: ScoreInput[] = Object.entries(scores).map(([metricId, data]) => ({
        metric_id: metricId,
        score: data.score,
        comment: data.comment || undefined,
      }));
      return adminAssessmentsApi.createOrUpdateAssessment(portfolio!.id, {
        scores: scoreInputs,
        final_comment: finalComment || undefined,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-assessments'] });
      queryClient.invalidateQueries({ queryKey: ['admin-assessment', portfolio?.id] });
      queryClient.invalidateQueries({ queryKey: ['admin-assessment-stats'] });
      toast.success('Penilaian berhasil disimpan');
      onClose();
    },
    onError: () => toast.error('Gagal menyimpan penilaian'),
  });

  const handleScoreChange = (metricId: string, score: number) => {
    setScores((prev) => ({
      ...prev,
      [metricId]: { ...prev[metricId], score },
    }));
  };

  const handleCommentChange = (metricId: string, comment: string) => {
    setScores((prev) => ({
      ...prev,
      [metricId]: { ...prev[metricId], comment },
    }));
  };

  const totalScore =
    Object.values(scores).length > 0
      ? Object.values(scores).reduce((sum, s) => sum + s.score, 0) / Object.values(scores).length
      : 0;

  const getScoreColor = (score: number) => {
    if (score < 4) return 'text-red-500';
    if (score <= 7) return 'text-amber-500';
    return 'text-green-500';
  };

  const getScoreBgColor = (score: number) => {
    if (score < 4) return 'bg-red-500/10 border-red-500/20';
    if (score <= 7) return 'bg-amber-500/10 border-amber-500/20';
    return 'bg-green-500/10 border-green-500/20';
  };

  if (!portfolio) return null;

  return (
    <Sheet open={!!portfolio} onOpenChange={() => onClose()}>
      <SheetContent className="w-full sm:max-w-xl p-0 flex flex-col gap-0">
        {/* Fixed Header */}
        <div className="shrink-0 border-b bg-background">
          {/* Portfolio Preview Header */}
          <div className="p-4 pb-3">
            <div className="flex items-start gap-3">
              {portfolio.thumbnail_url ? (
                <img
                  src={portfolio.thumbnail_url}
                  alt={portfolio.judul}
                  className="w-16 h-12 object-cover rounded-md shrink-0"
                />
              ) : (
                <div className="w-16 h-12 bg-muted rounded-md flex items-center justify-center shrink-0">
                  <ClipboardList className="h-5 w-5 text-muted-foreground" />
                </div>
              )}
              <div className="flex-1 min-w-0">
                <h3 className="font-semibold text-base line-clamp-1">{portfolio.judul}</h3>
                {portfolio.user && (
                  <p className="text-sm text-muted-foreground">oleh {portfolio.user.nama}</p>
                )}
              </div>
              <Link
                href={`/${portfolio.user?.username}/${portfolio.slug}`}
                className="shrink-0"
              >
                <Button variant="outline" size="sm" className="h-8">
                  <Eye className="h-3.5 w-3.5 mr-1.5" />
                  Lihat
                </Button>
              </Link>
            </div>
          </div>

          {/* Total Score - Sticky */}
          <div className={cn('px-4 py-3 border-t', getScoreBgColor(totalScore))}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className={cn('rounded-full p-1.5', getScoreBgColor(totalScore))}>
                  <Star className={cn('h-4 w-4', getScoreColor(totalScore), totalScore > 0 && 'fill-current')} />
                </div>
                <span className="font-medium text-sm">Total Nilai</span>
              </div>
              <div className="flex items-baseline gap-1">
                <span className={cn('text-3xl font-bold tabular-nums', getScoreColor(totalScore))}>
                  {totalScore.toFixed(1)}
                </span>
                <span className="text-muted-foreground text-sm">/ 10</span>
              </div>
            </div>
          </div>
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto">
          {isLoadingAssessment ? (
            <div className="py-12 flex flex-col items-center justify-center gap-3">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              <p className="text-sm text-muted-foreground">Memuat data penilaian...</p>
            </div>
          ) : (
            <div className="p-4 space-y-4">
              {/* Section: Metrics */}
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <ClipboardList className="h-4 w-4 text-muted-foreground" />
                  <h4 className="font-medium text-sm text-muted-foreground uppercase tracking-wide">
                    Penilaian per Metrik
                  </h4>
                </div>
                <div className="space-y-3">
                  {metrics.map((metric) => (
                    <MetricScoreInput
                      key={metric.id}
                      metric={metric}
                      score={scores[metric.id]?.score || 5}
                      comment={scores[metric.id]?.comment || ''}
                      onScoreChange={(score) => handleScoreChange(metric.id, score)}
                      onCommentChange={(comment) => handleCommentChange(metric.id, comment)}
                    />
                  ))}
                </div>
              </div>

              {/* Section: Final Comment */}
              <div className="space-y-3 pt-2">
                <div className="flex items-center gap-2">
                  <Star className="h-4 w-4 text-muted-foreground" />
                  <h4 className="font-medium text-sm text-muted-foreground uppercase tracking-wide">
                    Komentar Akhir
                  </h4>
                  <span className="text-xs text-muted-foreground">(Opsional)</span>
                </div>
                <Textarea
                  value={finalComment}
                  onChange={(e) => setFinalComment(e.target.value)}
                  placeholder="Berikan komentar atau feedback keseluruhan untuk portfolio ini..."
                  rows={3}
                  className="resize-none"
                />
              </div>
            </div>
          )}
        </div>

        {/* Fixed Footer */}
        <div className="shrink-0 border-t bg-background p-4">
          <div className="flex gap-3">
            <Button variant="outline" className="flex-1" onClick={onClose}>
              Batal
            </Button>
            <Button
              className="flex-1"
              onClick={() => submitMutation.mutate()}
              disabled={submitMutation.isPending || isLoadingAssessment}
            >
              {submitMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Simpan Penilaian
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}


// Metric Score Input Component
function MetricScoreInput({
  metric,
  score,
  comment,
  onScoreChange,
  onCommentChange,
}: {
  metric: AssessmentMetric;
  score: number;
  comment: string;
  onScoreChange: (score: number) => void;
  onCommentChange: (comment: string) => void;
}) {
  const [showComment, setShowComment] = useState(!!comment);

  return (
    <Card className="p-4 space-y-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <h5 className="font-medium">{metric.nama}</h5>
          {metric.deskripsi && (
            <p className="text-sm text-muted-foreground mt-1">{metric.deskripsi}</p>
          )}
        </div>
        <div className="text-right shrink-0">
          <span className="text-2xl font-bold text-primary">{score}</span>
          <span className="text-muted-foreground">/10</span>
        </div>
      </div>

      {/* Slider */}
      <div className="space-y-2">
        <Slider
          value={[score]}
          onValueChange={([value]) => onScoreChange(value)}
          min={1}
          max={10}
          step={1}
          className="w-full"
        />
        <div className="flex justify-between text-xs text-muted-foreground">
          <span>1</span>
          <span>5</span>
          <span>10</span>
        </div>
      </div>

      {/* Comment Toggle */}
      {!showComment ? (
        <Button
          variant="ghost"
          size="sm"
          className="text-muted-foreground"
          onClick={() => setShowComment(true)}
        >
          + Tambah Komentar
        </Button>
      ) : (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <Label className="text-sm">Komentar</Label>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-muted-foreground"
              onClick={() => {
                setShowComment(false);
                onCommentChange('');
              }}
            >
              <X className="h-3 w-3" />
            </Button>
          </div>
          <Textarea
            value={comment}
            onChange={(e) => onCommentChange(e.target.value)}
            placeholder="Komentar untuk metrik ini..."
            rows={2}
            className="text-sm"
          />
        </div>
      )}
    </Card>
  );
}
