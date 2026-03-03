'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Check, X, Eye, Loader2, Clock, FileText, AlertCircle } from 'lucide-react';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { adminPortfoliosApi } from '@/lib/api/admin';
import { PortfolioCard } from '@/lib/types';
import { BlockRenderer } from '@/components/portfolio/block-renderer';
import { portfoliosApi } from '@/lib/api';
import { formatDate } from '@/lib/utils/format';

type ReviewAction = 'approve' | 'reject';

export default function ModerationPage() {
  const queryClient = useQueryClient();
  const [selectedPortfolio, setSelectedPortfolio] = useState<PortfolioCard | null>(null);
  const [reviewNote, setReviewNote] = useState('');
  const [showReviewDialog, setShowReviewDialog] = useState(false);
  const [reviewAction, setReviewAction] = useState<ReviewAction>('approve');
  const [portfolioToReview, setPortfolioToReview] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-moderation'],
    queryFn: () => adminPortfoliosApi.getPortfolios({ status: 'pending_review', limit: 100 }),
  });

  const { data: detailData, isLoading: detailLoading } = useQuery({
    queryKey: ['portfolio-detail', selectedPortfolio?.id],
    queryFn: () => portfoliosApi.getPortfolioById(selectedPortfolio!.id),
    enabled: !!selectedPortfolio,
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, note }: { id: string; note?: string }) =>
      adminPortfoliosApi.approvePortfolio(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-moderation'] });
      queryClient.invalidateQueries({ queryKey: ['admin-sidebar-stats'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard-stats'] });
      toast.success('Portfolio berhasil disetujui');
      closeReviewDialog();
      setSelectedPortfolio(null);
    },
    onError: () => {
      toast.error('Gagal menyetujui portfolio');
    },
  });

  const rejectMutation = useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) =>
      adminPortfoliosApi.rejectPortfolio(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-moderation'] });
      queryClient.invalidateQueries({ queryKey: ['admin-sidebar-stats'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard-stats'] });
      toast.success('Portfolio berhasil ditolak');
      closeReviewDialog();
      setSelectedPortfolio(null);
    },
    onError: () => {
      toast.error('Gagal menolak portfolio');
    },
  });

  const portfolios = data?.data || [];
  const totalPending = (data?.meta as { total_count?: number })?.total_count ?? portfolios.length;

  const openReviewDialog = (id: string, action: ReviewAction) => {
    setPortfolioToReview(id);
    setReviewAction(action);
    setReviewNote('');
    setShowReviewDialog(true);
  };

  const closeReviewDialog = () => {
    setShowReviewDialog(false);
    setReviewNote('');
    setPortfolioToReview(null);
  };

  const confirmReview = () => {
    if (!portfolioToReview) return;

    if (reviewAction === 'approve') {
      approveMutation.mutate({
        id: portfolioToReview,
        note: reviewNote.trim() || undefined,
      });
    } else {
      if (!reviewNote.trim()) {
        toast.error('Catatan penolakan wajib diisi');
        return;
      }
      rejectMutation.mutate({ id: portfolioToReview, note: reviewNote });
    }
  };

  const isPending = approveMutation.isPending || rejectMutation.isPending;

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-9 w-48" />
          <Skeleton className="h-6 w-24" />
        </div>
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {[...Array(6)].map((_, i) => (
            <Skeleton key={i} className="h-72" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header Badge - Centered */}
      <div className="flex justify-center">
        <Badge
          variant={totalPending > 0 ? 'default' : 'secondary'}
          className="gap-1.5 px-4 py-2 text-sm"
        >
          <Clock className="h-4 w-4" />
          {totalPending} menunggu review
        </Badge>
      </div>

      {/* Empty State */}
      {portfolios.length === 0 ? (
        <Card className="border-dashed py-16">
          <div className="flex flex-col items-center justify-center">
            <div className="rounded-full bg-muted p-4">
              <FileText className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-lg font-semibold">Tidak ada portfolio pending</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Semua portfolio sudah direview. Cek kembali nanti.
            </p>
          </div>
        </Card>
      ) : (
        /* Portfolio Grid - same as portfolios page */
        <div className="grid gap-6 sm:grid-cols-2 xl:grid-cols-3">
          {portfolios.map((portfolio) => {
            const firstTag = portfolio.tags && portfolio.tags.length > 0 ? portfolio.tags[0] : null;

            return (
              <Card
                key={portfolio.id}
                className="group gap-0 overflow-hidden border py-0 transition-shadow hover:shadow-lg"
              >
                <div className="p-3 pb-4">
                  {/* Thumbnail - same aspect ratio as portfolio-card */}
                  <div className="relative aspect-[4/3] w-full overflow-hidden rounded-xl bg-muted">
                    {portfolio.thumbnail_url ? (
                      <Image
                        src={portfolio.thumbnail_url}
                        alt={portfolio.judul}
                        fill
                        className="object-cover transition-transform group-hover:scale-105"
                      />
                    ) : (
                      <div className="flex h-full items-center justify-center text-muted-foreground">
                        No Image
                      </div>
                    )}
                    {/* Preview overlay on hover */}
                    <div className="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => setSelectedPortfolio(portfolio)}
                      >
                        <Eye className="mr-2 h-4 w-4" />
                        Preview
                      </Button>
                    </div>
                  </div>

                  {/* Tag Badge */}
                  {firstTag && (
                    <Badge
                      variant="secondary"
                      className="mt-3 rounded-full px-2.5 py-0.5 text-xs font-normal"
                    >
                      {firstTag.nama}
                    </Badge>
                  )}

                  {/* Title */}
                  <h3 className="mt-2 line-clamp-2 font-semibold leading-tight">
                    {portfolio.judul}
                  </h3>

                  {/* User Info & Date */}
                  {portfolio.user && (
                    <div className="mt-3 flex items-center gap-2">
                      <Avatar className="h-8 w-8">
                        <AvatarImage src={portfolio.user.avatar_url} alt={portfolio.user.nama} />
                        <AvatarFallback className="text-xs">
                          {portfolio.user.nama?.charAt(0)}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex flex-col">
                        <span className="text-sm font-medium leading-tight">
                          {portfolio.user.nama}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          Posted on {formatDate(portfolio.created_at)}
                        </span>
                      </div>
                    </div>
                  )}

                  {/* Action Buttons */}
                  <div className="mt-4 flex gap-2">
                    <Button
                      size="sm"
                      variant="ghost"
                      className="flex-1 bg-green-500/10 text-green-600 hover:bg-green-500/20 hover:text-green-700 dark:text-green-400 dark:hover:text-green-300"
                      onClick={() => openReviewDialog(portfolio.id, 'approve')}
                    >
                      <Check className="mr-1.5 h-4 w-4" />
                      Setujui
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="flex-1 bg-red-500/10 text-red-600 hover:bg-red-500/20 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                      onClick={() => openReviewDialog(portfolio.id, 'reject')}
                    >
                      <X className="mr-1.5 h-4 w-4" />
                      Tolak
                    </Button>
                  </div>
                </div>
              </Card>
            );
          })}
        </div>
      )}

      {/* Preview Dialog */}
      <Dialog open={!!selectedPortfolio} onOpenChange={() => setSelectedPortfolio(null)}>
        <DialogContent className="flex max-h-[90vh] max-w-4xl flex-col gap-0 overflow-hidden p-0">
          {/* Sticky Header */}
          <div className="sticky top-0 z-10 border-b bg-background px-6 py-4">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0 flex-1">
                <DialogTitle className="line-clamp-1 text-lg font-semibold">
                  {selectedPortfolio?.judul}
                </DialogTitle>
                <DialogDescription className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
                  <div className="flex items-center gap-1.5">
                    <Avatar className="h-5 w-5">
                      <AvatarImage src={selectedPortfolio?.user?.avatar_url} />
                      <AvatarFallback className="text-[10px]">
                        {selectedPortfolio?.user?.nama?.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                    <span className="font-medium text-foreground">
                      {selectedPortfolio?.user?.nama}
                    </span>
                    <span className="text-muted-foreground">
                      @{selectedPortfolio?.user?.username}
                    </span>
                  </div>
                  {selectedPortfolio?.user?.kelas_nama && (
                    <>
                      <span className="text-muted-foreground">•</span>
                      <Badge variant="secondary" className="text-xs">
                        {selectedPortfolio.user.kelas_nama}
                      </Badge>
                    </>
                  )}
                  {selectedPortfolio?.created_at && (
                    <>
                      <span className="text-muted-foreground">•</span>
                      <span className="text-muted-foreground">
                        {formatDate(selectedPortfolio.created_at)}
                      </span>
                    </>
                  )}
                </DialogDescription>
              </div>
            </div>

            {/* Tags */}
            {selectedPortfolio?.tags && selectedPortfolio.tags.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-1.5">
                {selectedPortfolio.tags.map((tag) => (
                  <Badge key={tag.id} variant="outline" className="text-xs">
                    {tag.nama}
                  </Badge>
                ))}
              </div>
            )}
          </div>

          {/* Scrollable Content */}
          <div className="flex-1 overflow-y-auto px-6 py-6">
            {/* Thumbnail */}
            {selectedPortfolio?.thumbnail_url && (
              <div className="relative mb-6 aspect-video w-full overflow-hidden rounded-lg bg-muted">
                <Image
                  src={selectedPortfolio.thumbnail_url}
                  alt={selectedPortfolio.judul}
                  fill
                  className="object-cover"
                />
              </div>
            )}

            {/* Content Blocks */}
            {detailLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-48 w-full rounded-lg" />
                <Skeleton className="h-6 w-3/4" />
                <Skeleton className="h-6 w-1/2" />
                <Skeleton className="h-24 w-full rounded-lg" />
              </div>
            ) : detailData?.data?.content_blocks &&
              detailData.data.content_blocks.length > 0 ? (
              <div className="prose prose-sm max-w-none dark:prose-invert">
                <BlockRenderer blocks={detailData.data.content_blocks} />
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-16 text-center">
                <div className="rounded-full bg-muted p-4">
                  <AlertCircle className="h-8 w-8 text-muted-foreground" />
                </div>
                <p className="mt-3 font-medium">Tidak ada konten</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Portfolio ini belum memiliki content blocks
                </p>
              </div>
            )}
          </div>

          {/* Sticky Footer */}
          <div className="sticky bottom-0 flex items-center justify-between gap-4 border-t bg-background px-6 py-4">
            <Button variant="ghost" onClick={() => setSelectedPortfolio(null)}>
              Tutup
            </Button>
            <div className="flex gap-2">
              <Button
                variant="destructive"
                onClick={() => openReviewDialog(selectedPortfolio!.id, 'reject')}
              >
                <X className="mr-1.5 h-4 w-4" />
                Tolak
              </Button>
              <Button onClick={() => openReviewDialog(selectedPortfolio!.id, 'approve')}>
                <Check className="mr-1.5 h-4 w-4" />
                Setujui
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Review Confirmation Dialog */}
      <Dialog open={showReviewDialog} onOpenChange={closeReviewDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              {reviewAction === 'approve' ? (
                <>
                  <div className="rounded-full bg-green-100 p-1.5 dark:bg-green-900/30">
                    <Check className="h-4 w-4 text-green-600 dark:text-green-400" />
                  </div>
                  Setujui Portfolio
                </>
              ) : (
                <>
                  <div className="rounded-full bg-red-100 p-1.5 dark:bg-red-900/30">
                    <X className="h-4 w-4 text-red-600 dark:text-red-400" />
                  </div>
                  Tolak Portfolio
                </>
              )}
            </DialogTitle>
            <DialogDescription>
              {reviewAction === 'approve'
                ? 'Portfolio akan dipublish dan dapat dilihat oleh semua orang.'
                : 'Portfolio akan dikembalikan ke pemilik untuk diperbaiki.'}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="review-note">
                Catatan Review{' '}
                {reviewAction === 'reject' ? (
                  <span className="text-destructive">*</span>
                ) : (
                  <span className="text-muted-foreground">(opsional)</span>
                )}
              </Label>
              <Textarea
                id="review-note"
                value={reviewNote}
                onChange={(e) => setReviewNote(e.target.value)}
                placeholder={
                  reviewAction === 'approve'
                    ? 'Berikan komentar atau apresiasi (opsional)...'
                    : 'Jelaskan alasan penolakan dan saran perbaikan...'
                }
                rows={4}
                className="resize-none"
              />
              {reviewAction === 'reject' && (
                <p className="text-xs text-muted-foreground">
                  Catatan ini akan ditampilkan kepada pemilik portfolio
                </p>
              )}
            </div>
          </div>

          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={closeReviewDialog} disabled={isPending}>
              Batal
            </Button>
            {reviewAction === 'approve' ? (
              <Button onClick={confirmReview} disabled={isPending}>
                {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Setujui & Publish
              </Button>
            ) : (
              <Button
                variant="destructive"
                onClick={confirmReview}
                disabled={!reviewNote.trim() || isPending}
              >
                {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Tolak Portfolio
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
