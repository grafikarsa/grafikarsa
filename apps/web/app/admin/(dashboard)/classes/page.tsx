'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, Loader2, Users, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { DataTable, Column } from '@/components/admin/data-table';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import {
  adminClassesApi,
  adminMajorsApi,
  adminAcademicYearsApi,
  Class,
} from '@/lib/api/admin';

export default function AdminClassesPage() {
  const queryClient = useQueryClient();
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [editClass, setEditClass] = useState<Class | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteClass, setDeleteClass] = useState<Class | null>(null);

  const handleSearch = () => {
    setSearchQuery(searchInput);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  const { data, isLoading } = useQuery({
    queryKey: ['admin-classes'],
    queryFn: () => adminClassesApi.getClasses(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminClassesApi.deleteClass(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-classes'] });
      toast.success('Kelas berhasil dihapus');
      setDeleteClass(null);
    },
    onError: () => {
      toast.error('Gagal menghapus kelas. Pastikan tidak ada siswa di kelas ini.');
    },
  });

  const allClasses = data?.data || [];
  
  // Filter classes by search query (client-side)
  const classes = searchQuery
    ? allClasses.filter((c) =>
        c.nama.toLowerCase().includes(searchQuery.toLowerCase()) ||
        c.jurusan?.nama?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        c.rombel?.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : allClasses;

  const columns: Column<Class>[] = [
    {
      key: 'nama',
      header: 'Nama Kelas',
      render: (c) => <span className="font-semibold">{c.nama}</span>,
    },
    {
      key: 'tingkat',
      header: 'Tingkat',
      render: (c) => (
        <Badge variant="outline">
          {c.tingkat === 10 ? 'X' : c.tingkat === 11 ? 'XI' : 'XII'}
        </Badge>
      ),
    },
    {
      key: 'jurusan',
      header: 'Jurusan',
      render: (c) => (
        <span className="text-muted-foreground">{c.jurusan?.nama || '-'}</span>
      ),
    },
    {
      key: 'tahun_ajaran',
      header: 'Tahun Ajaran',
      render: (c) =>
        c.tahun_ajaran ? (
          <Badge variant="secondary">
            {c.tahun_ajaran.tahun_mulai}/{c.tahun_ajaran.tahun_mulai + 1}
          </Badge>
        ) : (
          '-'
        ),
    },
    {
      key: 'student_count',
      header: 'Siswa',
      render: (c) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Users className="h-4 w-4" />
          <span>{c.student_count || 0}</span>
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
            placeholder="Cari kelas..."
            className="pl-9"
          />
        </div>
        <Button variant="secondary" onClick={handleSearch}>
          <Search className="mr-2 h-4 w-4" />
          Cari
        </Button>
        <Button onClick={() => setIsCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Tambah Kelas
        </Button>
      </div>

      <DataTable
        data={classes}
        columns={columns}
        isLoading={isLoading}
        onEdit={setEditClass}
        onDelete={setDeleteClass}
      />

      {/* Form Dialog */}
      <ClassFormDialog
        kelas={editClass}
        open={isCreateOpen || !!editClass}
        onClose={() => {
          setIsCreateOpen(false);
          setEditClass(null);
        }}
      />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteClass}
        onOpenChange={() => setDeleteClass(null)}
        title="Hapus Kelas"
        description={
          <>
            Yakin ingin menghapus kelas <strong>&quot;{deleteClass?.nama}&quot;</strong>? Kelas tidak
            dapat dihapus jika masih memiliki siswa.
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteClass && deleteMutation.mutate(deleteClass.id)}
      />
    </div>
  );
}

function ClassFormDialog({
  kelas,
  open,
  onClose,
}: {
  kelas: Class | null;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const isEdit = !!kelas;
  const [formData, setFormData] = useState({
    tingkat: '10',
    rombel: 'A',
    jurusan_id: '',
    tahun_ajaran_id: '',
  });

  useEffect(() => {
    if (open) {
      if (kelas) {
        setFormData({
          tingkat: kelas.tingkat.toString(),
          rombel: kelas.rombel || 'A',
          jurusan_id: kelas.jurusan_id || '',
          tahun_ajaran_id: kelas.tahun_ajaran_id || '',
        });
      } else {
        setFormData({
          tingkat: '10',
          rombel: 'A',
          jurusan_id: '',
          tahun_ajaran_id: '',
        });
      }
    }
  }, [kelas, open]);

  const { data: majorsData } = useQuery({
    queryKey: ['admin-majors'],
    queryFn: () => adminMajorsApi.getMajors(),
  });

  const { data: yearsData } = useQuery({
    queryKey: ['admin-academic-years'],
    queryFn: () => adminAcademicYearsApi.getAcademicYears(),
  });

  const majors = majorsData?.data || [];
  const years = yearsData?.data || [];

  const createMutation = useMutation({
    mutationFn: () =>
      adminClassesApi.createClass({
        tingkat: parseInt(formData.tingkat),
        rombel: formData.rombel,
        jurusan_id: formData.jurusan_id,
        tahun_ajaran_id: formData.tahun_ajaran_id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-classes'] });
      toast.success('Kelas berhasil dibuat');
      onClose();
    },
    onError: () => {
      toast.error('Gagal membuat kelas');
    },
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      adminClassesApi.updateClass(kelas!.id, {
        tingkat: parseInt(formData.tingkat),
        rombel: formData.rombel,
        jurusan_id: formData.jurusan_id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-classes'] });
      toast.success('Kelas berhasil diperbarui');
      onClose();
    },
    onError: () => {
      toast.error('Gagal memperbarui kelas');
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

  // Preview nama kelas
  const selectedJurusan = majors.find((m) => m.id === formData.jurusan_id);
  const tingkatRomawi =
    formData.tingkat === '10' ? 'X' : formData.tingkat === '11' ? 'XI' : 'XII';
  const previewNama = selectedJurusan
    ? `${tingkatRomawi}-${selectedJurusan.kode.toUpperCase()}-${formData.rombel}`
    : '';

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-primary/10 p-2">
              <Users className="h-5 w-5 text-primary" />
            </div>
            <div>
              <DialogTitle>{isEdit ? 'Edit Kelas' : 'Tambah Kelas Baru'}</DialogTitle>
              <DialogDescription>
                {isEdit
                  ? 'Ubah informasi kelas yang sudah ada'
                  : 'Tambahkan kelas/rombongan belajar baru'}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 pt-4">
          {/* Tingkat */}
          <div className="space-y-2">
            <Label htmlFor="tingkat">
              Tingkat <span className="text-destructive">*</span>
            </Label>
            <Select
              value={formData.tingkat}
              onValueChange={(v) => setFormData({ ...formData, tingkat: v })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="10">Kelas 10 (X)</SelectItem>
                <SelectItem value="11">Kelas 11 (XI)</SelectItem>
                <SelectItem value="12">Kelas 12 (XII)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Jurusan */}
          <div className="space-y-2">
            <Label htmlFor="jurusan">
              Jurusan <span className="text-destructive">*</span>
            </Label>
            <Select
              value={formData.jurusan_id}
              onValueChange={(v) => setFormData({ ...formData, jurusan_id: v })}
            >
              <SelectTrigger>
                <SelectValue placeholder="Pilih jurusan" />
              </SelectTrigger>
              <SelectContent>
                {majors.map((m) => (
                  <SelectItem key={m.id} value={m.id}>
                    {m.nama} ({m.kode.toUpperCase()})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {majors.length === 0 && (
              <p className="text-xs text-amber-600">
                Belum ada jurusan. Tambahkan jurusan terlebih dahulu.
              </p>
            )}
          </div>

          {/* Rombel */}
          <div className="space-y-2">
            <Label htmlFor="rombel">
              Rombel <span className="text-destructive">*</span>
            </Label>
            <Input
              id="rombel"
              value={formData.rombel}
              onChange={(e) =>
                setFormData({ ...formData, rombel: e.target.value.toUpperCase() })
              }
              placeholder="A, B, C, ..."
              maxLength={1}
              required
            />
            <p className="text-xs text-muted-foreground">
              Satu huruf untuk identifikasi rombongan belajar (A-Z)
            </p>
          </div>

          {/* Tahun Ajaran (hanya untuk create) */}
          {!isEdit && (
            <div className="space-y-2">
              <Label htmlFor="tahun_ajaran">
                Tahun Ajaran <span className="text-destructive">*</span>
              </Label>
              <Select
                value={formData.tahun_ajaran_id}
                onValueChange={(v) => setFormData({ ...formData, tahun_ajaran_id: v })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Pilih tahun ajaran" />
                </SelectTrigger>
                <SelectContent>
                  {years.map((y) => (
                    <SelectItem key={y.id} value={y.id}>
                      {y.tahun_mulai}/{y.tahun_mulai + 1}
                      {y.is_active && ' (Aktif)'}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {years.length === 0 && (
                <p className="text-xs text-amber-600">
                  Belum ada tahun ajaran. Tambahkan tahun ajaran terlebih dahulu.
                </p>
              )}
            </div>
          )}

          {/* Preview */}
          {previewNama && (
            <div className="rounded-lg border bg-muted/50 p-3">
              <p className="text-xs font-medium text-muted-foreground">Preview nama kelas:</p>
              <p className="mt-1 text-lg font-semibold">{previewNama}</p>
            </div>
          )}

          <DialogFooter className="pt-4">
            <Button type="button" variant="outline" onClick={onClose}>
              Batal
            </Button>
            <Button
              type="submit"
              disabled={
                isPending ||
                !formData.jurusan_id ||
                !formData.rombel ||
                (!isEdit && !formData.tahun_ajaran_id)
              }
            >
              {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isEdit ? 'Simpan Perubahan' : 'Buat Kelas'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
