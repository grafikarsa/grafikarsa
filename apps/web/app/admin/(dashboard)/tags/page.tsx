'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, Loader2, Tag as TagIcon, Hash, Search } from 'lucide-react';
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

export default function AdminTagsPage() {
  const queryClient = useQueryClient();
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [editTag, setEditTag] = useState<Tag | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteTag, setDeleteTag] = useState<Tag | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-tags', searchQuery],
    queryFn: () => adminTagsApi.getTags({ search: searchQuery || undefined }),
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

  const handleSearch = () => {
    setSearchQuery(searchInput);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  const tags = data?.data || [];

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

  return (
    <div className="space-y-6">
      {/* Search and Action Button in same row */}
      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Cari tag..."
            className="pl-9"
          />
        </div>
        <Button variant="secondary" onClick={handleSearch}>
          <Search className="mr-2 h-4 w-4" />
          Cari
        </Button>
        <Button onClick={() => setIsCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Tambah Tag
        </Button>
      </div>

      <DataTable
        data={tags}
        columns={columns}
        isLoading={isLoading}
        onEdit={setEditTag}
        onDelete={setDeleteTag}
      />

      {/* Form Dialog */}
      <TagFormDialog
        tag={editTag}
        open={isCreateOpen || !!editTag}
        onClose={() => {
          setIsCreateOpen(false);
          setEditTag(null);
        }}
      />

      {/* Delete Confirmation */}
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
