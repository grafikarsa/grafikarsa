'use client';

import React, { useState, useCallback, useMemo, memo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, Loader2, Tag as TagIcon, Hash, Search } from 'lucide-react';
import { useDebounce } from '@/lib/hooks/use-debounce';
import { getDebugEmptyState } from '@/lib/utils/debug';
import { DebugBanner } from '@/components/admin/debug-banner';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { DataTable, Column } from '@/components/admin/data-table';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { adminTagsApi } from '@/lib/api/admin';
import { Tag } from '@/lib/types';

// Memoized Sticky Header Component - prevents re-render during data fetching
const TagFilters = memo(({
  search,
  onSearchChange,
  onCreateClick,
}: {
  search: string;
  onSearchChange: (value: string) => void;
  onCreateClick: () => void;
}) => {
  return (
    <div className="sticky top-0 z-10 bg-background/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6 lg:px-8 border-b">
      <div className="mx-auto w-full max-w-[1600px] flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari tag..."
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button onClick={onCreateClick}>
          <Plus className="mr-2 h-4 w-4" />
          Tambah Tag
        </Button>
      </div>
    </div>
  );
});

TagFilters.displayName = 'TagFilters';

export default function AdminTagsPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [editTag, setEditTag] = useState<Tag | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteTag, setDeleteTag] = useState<Tag | null>(null);
  
  const debouncedSearch = useDebounce(search, 300);

  // Memoize callbacks to prevent unnecessary re-creation
  const handleSearchChange = useCallback((value: string) => {
    setSearch(value);
  }, []);

  const handleCreateClick = useCallback(() => {
    setIsCreateOpen(true);
  }, []);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['admin-tags', debouncedSearch],
    queryFn: () => adminTagsApi.getTags({ search: debouncedSearch || undefined }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminTagsApi.deleteTag(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-tags'] });
      toast.success('Tag berhasil dihapus');
      setDeleteTag(null);
    },
    onError: () => {
      toast.error('Gagal menghapus tag');
    },
  });

  const tags = data?.data || [];

  // Debug mode: Force empty state
  const debugMode = getDebugEmptyState();
  const displayTags = debugMode ? [] : tags;

  const columns: Column<Tag>[] = [
    {
      key: 'nama',
      header: 'Nama Tag',
      render: (tag) => (
        <div className="flex items-center gap-2">
          <Hash className="h-4 w-4 text-muted-foreground" />
          <span className="font-medium">{tag.nama}</span>
        </div>
      ),
    },
  ];

  // Custom actions for DataTable - inline buttons instead of dropdown
  const renderActions = useCallback((tag: Tag) => (
    <div className="flex justify-end gap-1">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setEditTag(tag)}
      >
        Edit
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className="text-destructive hover:text-destructive"
        onClick={() => setDeleteTag(tag)}
      >
        Hapus
      </Button>
    </div>
  ), []);

  // Memoize content area - only re-render when actual data changes
  const contentArea = useMemo(() => {
    // Show loading overlay during refetch
    const showLoadingOverlay = isFetching && data;

    // Empty state
    if (displayTags.length === 0 && !isLoading) {
      const hasSearch = debouncedSearch;
      
      return (
        <>
          {debugMode && <DebugBanner pageName="Tags" />}
          <div className="flex items-center justify-center min-h-[60vh]">
            <div className="flex flex-col items-center justify-center px-6">
              <div className="rounded-full bg-primary/10 p-4">
                <TagIcon className="h-10 w-10 text-primary" />
              </div>
              <h3 className="mt-6 text-xl font-semibold">
                {hasSearch ? 'Tidak ada tag yang sesuai' : 'Belum ada tag'}
              </h3>
              <p className="mt-2 text-sm text-muted-foreground text-center max-w-sm">
                {hasSearch 
                  ? 'Coba ubah kata kunci pencarian untuk menemukan tag yang Anda cari.'
                  : 'Tag digunakan untuk mengkategorisasi portfolio. Buat tag pertama untuk mulai mengorganisir portfolio siswa.'
                }
              </p>
              {!hasSearch && (
                <Button onClick={handleCreateClick} className="mt-6">
                  <Plus className="mr-2 h-4 w-4" />
                  Buat Tag Pertama
                </Button>
              )}
            </div>
          </div>
        </>
      );
    }

    return (
      <div className="relative">
        {debugMode && <DebugBanner pageName="Tags" />}
        <DataTable
          data={displayTags}
          columns={columns}
          isLoading={isLoading && !data}
          actions={renderActions}
        />

        {showLoadingOverlay && (
          <div className="absolute inset-0 bg-background/50 backdrop-blur-sm z-10 flex items-center justify-center rounded-lg">
            <div className="flex items-center gap-2 bg-background border rounded-lg px-4 py-2 shadow-lg">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span className="text-sm font-medium">Memuat...</span>
            </div>
          </div>
        )}
      </div>
    );
  }, [tags, columns, isLoading, data, isFetching, debouncedSearch, handleCreateClick, renderActions]);

  return (
    <div className="flex flex-col -m-4 sm:-m-6 lg:-m-8">
      {/* Sticky Header - Filters & Actions (Full Width) */}
      <TagFilters
        search={search}
        onSearchChange={handleSearchChange}
        onCreateClick={handleCreateClick}
      />

      {/* Content Area with proper padding */}
      <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
        <div className="mx-auto w-full max-w-[1600px]">
          {contentArea}
        </div>
      </div>

      {/* Modals - Outside scrollable area */}
      <TagFormDialog
        tag={editTag}
        open={isCreateOpen || !!editTag}
        onClose={() => {
          setIsCreateOpen(false);
          setEditTag(null);
        }}
      />

      <ConfirmDialog
        open={!!deleteTag}
        onOpenChange={() => setDeleteTag(null)}
        title="Hapus Tag"
        description={
          <>
            Yakin ingin menghapus tag <strong>&quot;{deleteTag?.nama}&quot;</strong>? Portfolio yang
            menggunakan tag ini tidak akan terhapus, hanya relasi tag-nya yang dihapus.
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteTag && deleteMutation.mutate(deleteTag.id)}
      />
    </div>
  );
}

function TagFormDialog({
  tag,
  open,
  onClose,
}: {
  tag: Tag | null;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const isEdit = !!tag;
  const [nama, setNama] = useState('');

  React.useEffect(() => {
    if (open) {
      if (tag) {
        setNama(tag.nama);
      } else {
        setNama('');
      }
    }
  }, [tag, open]);

  const createMutation = useMutation({
    mutationFn: () => adminTagsApi.createTag({ nama }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-tags'] });
      toast.success('Tag berhasil dibuat');
      onClose();
      setNama('');
    },
    onError: () => {
      toast.error('Gagal membuat tag');
    },
  });

  const updateMutation = useMutation({
    mutationFn: () => adminTagsApi.updateTag(tag!.id, { nama }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-tags'] });
      toast.success('Tag berhasil diperbarui');
      onClose();
    },
    onError: () => {
      toast.error('Gagal memperbarui tag');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isEdit) {
      updateMutation.mutate();
    } else {
      createMutation.mutate();
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-primary/10 p-2">
              <TagIcon className="h-5 w-5 text-primary" />
            </div>
            <div>
              <DialogTitle>{isEdit ? 'Edit Tag' : 'Tambah Tag Baru'}</DialogTitle>
              <DialogDescription>
                {isEdit
                  ? 'Ubah nama tag yang sudah ada'
                  : 'Buat tag baru untuk kategorisasi portfolio'}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 pt-4">
          <div className="space-y-2">
            <Label htmlFor="nama">
              Nama Tag <span className="text-destructive">*</span>
            </Label>
            <Input
              id="nama"
              value={nama}
              onChange={(e) => setNama(e.target.value)}
              placeholder="Contoh: Web Development"
              required
            />
            <p className="text-xs text-muted-foreground">
              Gunakan nama yang deskriptif dan mudah dipahami
            </p>
          </div>

          <DialogFooter className="pt-4">
            <Button type="button" variant="outline" onClick={onClose}>
              Batal
            </Button>
            <Button type="submit" disabled={isPending || !nama.trim()}>
              {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isEdit ? 'Simpan Perubahan' : 'Buat Tag'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
