'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Search,
  Plus,
  Pencil,
  Trash2,
  FileText,
  Eye,
  EyeOff,
  Calendar,
} from 'lucide-react';
import { getDebugEmptyState } from '@/lib/utils/debug';
import { DebugBanner } from '@/components/admin/debug-banner';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import {
  getAdminChangelogs,
  deleteChangelog,
  publishChangelog,
  unpublishChangelog,
} from '@/lib/api/changelog';
import type { ChangelogListItem } from '@/lib/types/changelog';
import { formatDate } from '@/lib/utils/format';
import { cn } from '@/lib/utils';
import { ChangelogFormModal } from '@/components/changelog/changelog-form-modal';

const categoryColors: Record<string, string> = {
  added: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  updated: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  removed: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  fixed: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
};

export default function AdminChangelogsPage() {
  const queryClient = useQueryClient();
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [page, setPage] = useState(1);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ChangelogListItem | null>(null);

  const handleSearch = () => {
    setSearchQuery(searchInput);
    setPage(1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  const { data, isLoading } = useQuery({
    queryKey: ['admin-changelogs', searchQuery, page],
    queryFn: () => getAdminChangelogs(page, 20, searchQuery),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteChangelog(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-changelogs'] });
      toast.success('Changelog dihapus');
      setDeleteTarget(null);
    },
    onError: () => toast.error('Gagal menghapus changelog'),
  });

  const publishMutation = useMutation({
    mutationFn: (id: string) => publishChangelog(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-changelogs'] });
      toast.success('Changelog dipublish');
    },
    onError: () => toast.error('Gagal mempublish changelog'),
  });

  const unpublishMutation = useMutation({
    mutationFn: (id: string) => unpublishChangelog(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-changelogs'] });
      toast.success('Changelog di-unpublish');
    },
    onError: () => toast.error('Gagal meng-unpublish changelog'),
  });

  // axios wraps response in .data, then our API wraps in .data again
  const changelogs = data?.data?.data || [];
  const pagination = data?.data?.meta;

  // Debug mode: Force empty state
  const debugMode = getDebugEmptyState();
  const displayChangelogs = debugMode ? [] : changelogs;

  if (isLoading) {
    return (
      <div className="flex flex-col -m-4 sm:-m-6 lg:-m-8">
        {/* Sticky Header Skeleton */}
        <div className="sticky top-0 z-50 bg-background/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6 lg:px-8 border-b">
          <div className="mx-auto w-full max-w-[1600px] flex gap-3">
            <Skeleton className="h-10 flex-1" />
            <Skeleton className="h-10 w-24" />
            <Skeleton className="h-10 w-40" />
          </div>
        </div>
        <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-[1600px] space-y-3">
            {[...Array(5)].map((_, i) => (
              <Skeleton key={i} className="h-24" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col -m-4 sm:-m-6 lg:-m-8">
      {/* Sticky Header - Search & Actions */}
      <div className="sticky top-0 z-50 bg-background/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6 lg:px-8 border-b">
        <div className="mx-auto w-full max-w-[1600px]">
          <div className="flex gap-3">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Cari changelog..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={handleKeyDown}
                className="pl-9"
              />
            </div>
            <Button variant="secondary" onClick={handleSearch}>
              <Search className="mr-2 h-4 w-4" />
              Cari
            </Button>
            <Button onClick={() => setIsCreating(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Buat Changelog
            </Button>
          </div>
        </div>
      </div>

      {/* Content Area with proper padding */}
      <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
        <div className="mx-auto w-full max-w-[1600px] space-y-6">
          {debugMode && <DebugBanner pageName="Changelog" />}

          {/* Changelog List */}
          {displayChangelogs.length === 0 ? (
            <div className="flex items-center justify-center min-h-[60vh]">
              <div className="flex flex-col items-center justify-center px-6">
                <div className="rounded-full bg-primary/10 p-4">
                  <FileText className="h-10 w-10 text-primary" />
                </div>
                <h3 className="mt-6 text-xl font-semibold">
                  {searchQuery ? 'Tidak ada changelog yang sesuai' : 'Belum ada changelog'}
                </h3>
                <p className="mt-2 text-sm text-muted-foreground text-center max-w-sm">
                  {searchQuery
                    ? 'Coba ubah kata kunci pencarian untuk menemukan changelog yang Anda cari.'
                    : 'Changelog membantu pengguna mengetahui fitur baru, perbaikan, dan perubahan pada platform.'}
                </p>
                {!searchQuery && (
                  <Button className="mt-4" onClick={() => setIsCreating(true)}>
                    <Plus className="mr-2 h-4 w-4" />
                    Buat Changelog
                  </Button>
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              {displayChangelogs.map((changelog: ChangelogListItem) => (
                <Card key={changelog.id} className="p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <Badge variant="outline" className="font-mono">
                          v{changelog.version}
                        </Badge>
                        <Badge
                          variant={changelog.is_published ? 'default' : 'secondary'}
                          className={cn(
                            changelog.is_published
                              ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                              : ''
                          )}
                        >
                          {changelog.is_published ? 'Published' : 'Draft'}
                        </Badge>
                        {changelog.categories.map((cat: string) => (
                          <span
                            key={cat}
                            className={cn(
                              'rounded-full px-2 py-0.5 text-xs font-medium capitalize',
                              categoryColors[cat] || ''
                            )}
                          >
                            {cat}
                          </span>
                        ))}
                      </div>
                      <h3 className="mt-2 font-semibold">{changelog.title}</h3>
                      {changelog.description && (
                        <p className="mt-1 text-sm text-muted-foreground line-clamp-2">
                          {changelog.description}
                        </p>
                      )}
                      <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
                        <Calendar className="h-3 w-3" />
                        <span>{formatDate(changelog.release_date)}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {changelog.is_published ? (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => unpublishMutation.mutate(changelog.id)}
                          disabled={unpublishMutation.isPending}
                        >
                          <EyeOff className="mr-1 h-4 w-4" />
                          Unpublish
                        </Button>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => publishMutation.mutate(changelog.id)}
                          disabled={publishMutation.isPending}
                        >
                          <Eye className="mr-1 h-4 w-4" />
                          Publish
                        </Button>
                      )}
                      <Button
                        variant="outline"
                        size="icon"
                        onClick={() => setEditingId(changelog.id)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        onClick={() => setDeleteTarget(changelog)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          )}

          {/* Pagination */}
          {pagination && pagination.total_pages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-4">
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

      {/* Create/Edit Modal */}
      <ChangelogFormModal
        open={isCreating || !!editingId}
        onClose={() => {
          setIsCreating(false);
          setEditingId(null);
        }}
        editId={editingId}
      />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Hapus Changelog"
        description={`Yakin ingin menghapus changelog v${deleteTarget?.version}?`}
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  );
}
