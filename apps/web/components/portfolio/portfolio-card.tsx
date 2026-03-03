'use client';

import { useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Pencil,
  Trash2,
  Send,
  Archive,
  ArchiveRestore,
  Eye,
  Loader2,
} from 'lucide-react';
import confetti from 'canvas-confetti';
import { Card } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { portfoliosApi } from '@/lib/api';
import { PortfolioCard as PortfolioCardType, PortfolioStatus } from '@/lib/types';
import { formatDate } from '@/lib/utils/format';
import { cn } from '@/lib/utils';

interface PortfolioCardProps {
  portfolio: PortfolioCardType;
  showStatus?: boolean;
  showActions?: boolean;
  username?: string;
}

// Status styles matching admin dashboard
const statusStyles: Record<PortfolioStatus, string> = {
  draft: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  pending_review: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  published: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  rejected: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  archived: 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400',
};

const statusLabels: Record<PortfolioStatus, string> = {
  draft: 'Draft',
  pending_review: 'Menunggu Review',
  published: 'Published',
  rejected: 'Ditolak',
  archived: 'Diarsipkan',
};

export function PortfolioCard({ portfolio, showStatus = false, showActions = false, username }: PortfolioCardProps) {
  const { id, judul, slug, thumbnail_url, published_at, created_at, user, tags, status } = portfolio;
  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const resolvedUsername = user?.username || username;
  const firstTag = tags && tags.length > 0 ? tags[0] : null;

  // Mutations
  const submitMutation = useMutation({
    mutationFn: () => portfoliosApi.submitPortfolio(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-portfolios'] });
      queryClient.invalidateQueries({ queryKey: ['my-portfolios'] });
      toast.success('Portfolio dikirim untuk review');
      confetti({
        particleCount: 150,
        spread: 70,
        origin: { y: 0.6 },
        colors: ['#22c55e', '#10b981', '#ffffff'] // Primary green colors
      });
    },
    onError: () => toast.error('Gagal mengirim portfolio'),
  });

  const archiveMutation = useMutation({
    mutationFn: () =>
      status === 'archived' ? portfoliosApi.unarchivePortfolio(id) : portfoliosApi.archivePortfolio(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-portfolios'] });
      queryClient.invalidateQueries({ queryKey: ['my-portfolios'] });
      toast.success(status === 'archived' ? 'Portfolio diaktifkan kembali' : 'Portfolio diarsipkan');
    },
    onError: () => toast.error('Gagal mengubah status'),
  });

  const deleteMutation = useMutation({
    mutationFn: () => portfoliosApi.deletePortfolio(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-portfolios'] });
      queryClient.invalidateQueries({ queryKey: ['my-portfolios'] });
      toast.success('Portfolio dihapus');
      setDeleteDialogOpen(false);
    },
    onError: () => toast.error('Gagal menghapus portfolio'),
  });

  const canSubmit = status === 'draft' || status === 'rejected';
  const canArchive = status === 'published';
  const canUnarchive = status === 'archived';
  const isLoading = submitMutation.isPending || archiveMutation.isPending;

  return (
    <TooltipProvider>
      <Card className="group gap-0 overflow-hidden border py-0 transition-shadow hover:shadow-lg">
        <Link href={`/${resolvedUsername}/${slug}`} className="block p-3">
          {/* Thumbnail */}
          <div className="relative aspect-video w-full overflow-hidden rounded-lg bg-muted sm:aspect-[4/3] sm:rounded-xl">
            {thumbnail_url ? (
              <Image
                src={thumbnail_url}
                alt={judul}
                fill
                className="object-cover transition-transform group-hover:scale-105"
              />
            ) : (
              <div className="flex h-full items-center justify-center text-xs text-muted-foreground sm:text-sm">
                No Image
              </div>
            )}
          </div>

          {/* Tag & Status */}
          <div className="mt-2.5 flex items-center justify-between gap-2 sm:mt-3">
            {firstTag ? (
              <Badge variant="secondary" className="max-w-[120px] truncate rounded-full px-2 py-0.5 text-[10px] font-normal sm:max-w-none sm:px-2.5 sm:text-xs">
                {firstTag.nama}
              </Badge>
            ) : (
              <div />
            )}
            {showStatus && status && (
              <span className={cn('rounded-full px-2 py-0.5 text-[10px] font-medium sm:px-2.5 sm:py-1 sm:text-xs', statusStyles[status])}>
                {statusLabels[status]}
              </span>
            )}
          </div>

          {/* Title */}
          <h3 className="mt-1.5 line-clamp-2 text-sm font-semibold leading-tight group-hover:text-primary sm:mt-2 sm:text-base">{judul}</h3>

          {/* User Info & Date */}
          {user ? (
            <div className="mt-2.5 flex items-center gap-2 sm:mt-3">
              <Avatar className="h-6 w-6 sm:h-8 sm:w-8">
                <AvatarImage src={user.avatar_url} alt={user.nama} />
                <AvatarFallback className="text-[10px] sm:text-xs">{user.nama?.charAt(0)}</AvatarFallback>
              </Avatar>
              <div className="flex flex-col overflow-hidden">
                <span className="truncate text-xs font-medium leading-tight sm:text-sm">{user.nama}</span>
                <span className="truncate text-[10px] text-muted-foreground sm:text-xs">
                  {published_at ? `Posted ${formatDate(published_at)}` : `Dibuat ${formatDate(created_at)}`}
                </span>
              </div>
            </div>
          ) : (
            <p className="mt-3 text-xs text-muted-foreground">Dibuat {formatDate(created_at)}</p>
          )}
        </Link>

        {/* Action Buttons */}
        {showActions && (
          <div className="flex items-center gap-1.5 border-t px-3 py-2.5">
            {/* View */}
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
                  <Link href={`/${resolvedUsername}/${slug}`}>
                    <Eye className="h-4 w-4" />
                  </Link>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Lihat</TooltipContent>
            </Tooltip>

            {/* Edit */}
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
                  <Link href={`/${resolvedUsername}/${slug}/edit`}>
                    <Pencil className="h-4 w-4" />
                  </Link>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Edit</TooltipContent>
            </Tooltip>

            {/* Submit for Review */}
            {canSubmit && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-amber-600 hover:bg-amber-100 hover:text-amber-700 dark:hover:bg-amber-900/30"
                    onClick={() => submitMutation.mutate()}
                    disabled={isLoading}
                  >
                    {submitMutation.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Send className="h-4 w-4" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Kirim Review</TooltipContent>
              </Tooltip>
            )}

            {/* Archive */}
            {canArchive && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => archiveMutation.mutate()}
                    disabled={isLoading}
                  >
                    {archiveMutation.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Archive className="h-4 w-4" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Arsipkan</TooltipContent>
              </Tooltip>
            )}

            {/* Unarchive */}
            {canUnarchive && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-green-600 hover:bg-green-100 hover:text-green-700 dark:hover:bg-green-900/30"
                    onClick={() => archiveMutation.mutate()}
                    disabled={isLoading}
                  >
                    {archiveMutation.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <ArchiveRestore className="h-4 w-4" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Aktifkan Kembali</TooltipContent>
              </Tooltip>
            )}

            {/* Spacer */}
            <div className="flex-1" />

            {/* Delete */}
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-destructive hover:bg-destructive/10 hover:text-destructive"
                  onClick={() => setDeleteDialogOpen(true)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Hapus</TooltipContent>
            </Tooltip>
          </div>
        )}
      </Card>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Hapus Portfolio"
        description={
          <>
            Yakin ingin menghapus <strong>{judul}</strong>? Tindakan ini tidak dapat dibatalkan.
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </TooltipProvider>
  );
}
