'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, Loader2, GraduationCap, Search } from 'lucide-react';
import { getDebugEmptyState } from '@/lib/utils/debug';
import { DebugBanner } from '@/components/admin/debug-banner';
import { useDebounce } from '@/lib/hooks/use-debounce';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
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
import { adminMajorsApi, Major } from '@/lib/api/admin';

export default function AdminMajorsPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [editMajor, setEditMajor] = useState<Major | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteMajor, setDeleteMajor] = useState<Major | null>(null);
  
  const debouncedSearch = useDebounce(search, 300);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-majors'],
    queryFn: () => adminMajorsApi.getMajors(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminMajorsApi.deleteMajor(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-majors'] });
      toast.success('Jurusan berhasil dihapus');
      setDeleteMajor(null);
    },
    onError: () => {
      toast.error('Gagal menghapus jurusan. Pastikan tidak ada kelas yang menggunakan jurusan ini.');
    },
  });

  const majors = data?.data || [];
  
  // Filter by search
  const filteredMajors = debouncedSearch
    ? majors.filter((m) =>
        m.nama.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
        m.kode.toLowerCase().includes(debouncedSearch.toLowerCase())
      )
    : majors;

  // Debug mode: Force empty state
  const debugMode = getDebugEmptyState();
  const displayMajors = debugMode ? [] : filteredMajors;

  const columns: Column<Major>[] = [
    {
      key: 'kode',
      header: 'Kode',
      render: (m) => (
        <Badge variant="secondary" className="font-mono uppercase">
          {m.kode}
        </Badge>
      ),
    },
    {
      key: 'nama',
      header: 'Nama Jurusan',
      render: (m) => <span className="font-medium">{m.nama}</span>,
    },
  ];

  return (
    <div className="flex flex-col -m-4 sm:-m-6 lg:-m-8">
      {/* Sticky Header - Search & Actions (Full Width) */}
      <div className="sticky top-0 z-10 bg-background/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6 lg:px-8 border-b">
        <div className="mx-auto w-full max-w-[1600px] flex gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Cari jurusan..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
          <Button onClick={() => setIsCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Tambah Jurusan
          </Button>
        </div>
      </div>

      {/* Content Area with proper padding */}
      <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
        <div className="mx-auto w-full max-w-[1600px]">
          {debugMode && <DebugBanner pageName="Jurusan" />}

          {displayMajors.length === 0 && !isLoading ? (
            <div className="flex items-center justify-center min-h-[60vh]">
              <div className="flex flex-col items-center justify-center px-6">
                <div className="rounded-full bg-primary/10 p-4">
                  <GraduationCap className="h-10 w-10 text-primary" />
                </div>
                <h3 className="mt-6 text-xl font-semibold">
                  {search ? 'Tidak ada jurusan yang sesuai' : 'Belum ada jurusan'}
                </h3>
                <p className="mt-2 text-sm text-muted-foreground text-center max-w-sm">
                  {search
                    ? 'Coba ubah kata kunci pencarian untuk menemukan jurusan yang Anda cari.'
                    : 'Jurusan/kompetensi keahlian digunakan untuk mengorganisir kelas. Buat jurusan pertama untuk mulai mengelola kelas siswa.'}
                </p>
                {!search && (
                  <Button onClick={() => setIsCreateOpen(true)} className="mt-6">
                    <Plus className="mr-2 h-4 w-4" />
                    Buat Jurusan Pertama
                  </Button>
                )}
              </div>
            </div>
          ) : (
            <DataTable
              data={displayMajors}
              columns={columns}
              isLoading={isLoading}
              onEdit={setEditMajor}
              onDelete={setDeleteMajor}
            />
          )}
        </div>
      </div>

      {/* Modals - Outside scrollable area */}
      {/* Form Dialog */}
      <MajorFormDialog
        major={editMajor}
        open={isCreateOpen || !!editMajor}
        onClose={() => {
          setIsCreateOpen(false);
          setEditMajor(null);
        }}
      />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteMajor}
        onOpenChange={() => setDeleteMajor(null)}
        title="Hapus Jurusan"
        description={
          <>
            Yakin ingin menghapus jurusan <strong>&quot;{deleteMajor?.nama}&quot;</strong>? Jurusan
            tidak dapat dihapus jika masih digunakan oleh kelas.
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteMajor && deleteMutation.mutate(deleteMajor.id)}
      />
    </div>
  );
}

function MajorFormDialog({
  major,
  open,
  onClose,
}: {
  major: Major | null;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const isEdit = !!major;
  const [formData, setFormData] = useState({
    nama: '',
    kode: '',
  });

  useEffect(() => {
    if (open) {
      if (major) {
        setFormData({
          nama: major.nama,
          kode: major.kode,
        });
      } else {
        setFormData({ nama: '', kode: '' });
      }
    }
  }, [major, open]);

  const createMutation = useMutation({
    mutationFn: () => adminMajorsApi.createMajor(formData),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-majors'] });
      toast.success('Jurusan berhasil dibuat');
      onClose();
      setFormData({ nama: '', kode: '' });
    },
    onError: () => {
      toast.error('Gagal membuat jurusan');
    },
  });

  const updateMutation = useMutation({
    mutationFn: () => adminMajorsApi.updateMajor(major!.id, formData),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-majors'] });
      toast.success('Jurusan berhasil diperbarui');
      onClose();
    },
    onError: () => {
      toast.error('Gagal memperbarui jurusan');
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
              <GraduationCap className="h-5 w-5 text-primary" />
            </div>
            <div>
              <DialogTitle>{isEdit ? 'Edit Jurusan' : 'Tambah Jurusan Baru'}</DialogTitle>
              <DialogDescription>
                {isEdit
                  ? 'Ubah informasi jurusan yang sudah ada'
                  : 'Tambahkan jurusan/kompetensi keahlian baru'}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 pt-4">
          <div className="space-y-2">
            <Label htmlFor="kode">
              Kode Jurusan <span className="text-destructive">*</span>
            </Label>
            <Input
              id="kode"
              value={formData.kode}
              onChange={(e) => setFormData({ ...formData, kode: e.target.value.toLowerCase().replace(/[^a-z]/g, '') })}
              placeholder="Contoh: rpl"
              maxLength={10}
              required
            />
            <p className="text-xs text-muted-foreground">
              Singkatan jurusan dalam huruf kecil, hanya huruf (maks. 10 karakter)
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="nama">
              Nama Lengkap <span className="text-destructive">*</span>
            </Label>
            <Input
              id="nama"
              value={formData.nama}
              onChange={(e) => setFormData({ ...formData, nama: e.target.value })}
              placeholder="Contoh: Rekayasa Perangkat Lunak"
              required
            />
          </div>

          {/* Preview */}
          {formData.kode && (
            <div className="rounded-lg border bg-muted/50 p-3">
              <p className="text-xs font-medium text-muted-foreground">Preview nama kelas:</p>
              <p className="mt-1 font-semibold">
                XII-{formData.kode.toUpperCase()}-A
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                Kode disimpan: <code className="rounded bg-muted px-1">{formData.kode}</code>
              </p>
            </div>
          )}

          <DialogFooter className="pt-4">
            <Button type="button" variant="outline" onClick={onClose}>
              Batal
            </Button>
            <Button type="submit" disabled={isPending || !formData.nama.trim() || !formData.kode.trim()}>
              {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isEdit ? 'Simpan Perubahan' : 'Buat Jurusan'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
