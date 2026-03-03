'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, Loader2, CheckCircle, Calendar } from 'lucide-react';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { DataTable, Column } from '@/components/admin/data-table';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { adminAcademicYearsApi, AcademicYear } from '@/lib/api/admin';

const MONTHS = [
  { value: 1, label: 'Januari' },
  { value: 2, label: 'Februari' },
  { value: 3, label: 'Maret' },
  { value: 4, label: 'April' },
  { value: 5, label: 'Mei' },
  { value: 6, label: 'Juni' },
  { value: 7, label: 'Juli' },
  { value: 8, label: 'Agustus' },
  { value: 9, label: 'September' },
  { value: 10, label: 'Oktober' },
  { value: 11, label: 'November' },
  { value: 12, label: 'Desember' },
];

export default function AdminAcademicYearsPage() {
  const queryClient = useQueryClient();
  const [editYear, setEditYear] = useState<AcademicYear | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteYear, setDeleteYear] = useState<AcademicYear | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-academic-years'],
    queryFn: () => adminAcademicYearsApi.getAcademicYears(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminAcademicYearsApi.deleteAcademicYear(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-academic-years'] });
      toast.success('Tahun ajaran berhasil dihapus');
      setDeleteYear(null);
    },
    onError: () => {
      toast.error('Gagal menghapus tahun ajaran. Pastikan tidak ada kelas yang menggunakan tahun ajaran ini.');
    },
  });

  const setActiveMutation = useMutation({
    mutationFn: (id: string) => adminAcademicYearsApi.setActive(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-academic-years'] });
      toast.success('Tahun ajaran aktif berhasil diubah');
    },
    onError: () => {
      toast.error('Gagal mengubah tahun ajaran aktif');
    },
  });

  const years = data?.data || [];

  const columns: Column<AcademicYear>[] = [
    {
      key: 'tahun_mulai',
      header: 'Tahun Ajaran',
      render: (y) => (
        <span className="font-semibold">
          {y.tahun_mulai}/{y.tahun_mulai + 1}
        </span>
      ),
    },
    {
      key: 'promotion',
      header: 'Tanggal Kenaikan',
      render: (y) => {
        const month = MONTHS.find((m) => m.value === y.promotion_month);
        return (
          <span className="text-muted-foreground">
            {y.promotion_day} {month?.label || y.promotion_month}
          </span>
        );
      },
    },
    {
      key: 'is_active',
      header: 'Status',
      render: (y) =>
        y.is_active ? (
          <Badge variant="default" className="gap-1 bg-green-100 text-green-700 hover:bg-green-100 dark:bg-green-900/30 dark:text-green-400">
            <CheckCircle className="h-3 w-3" />
            Aktif
          </Badge>
        ) : (
          <Badge variant="secondary">Tidak Aktif</Badge>
        ),
    },
  ];

  const handleDelete = (year: AcademicYear) => {
    if (year.is_active) {
      toast.error('Tidak dapat menghapus tahun ajaran yang aktif');
      return;
    }
    setDeleteYear(year);
  };

  return (
    <div className="space-y-6">
      {/* Action Button */}
      <div className="flex justify-end">
        <Button onClick={() => setIsCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Tambah Tahun Ajaran
        </Button>
      </div>

      <DataTable
        data={years}
        columns={columns}
        isLoading={isLoading}
        onEdit={setEditYear}
        onDelete={handleDelete}
        actions={(year) =>
          !year.is_active ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setActiveMutation.mutate(year.id)}
              disabled={setActiveMutation.isPending}
            >
              Set Aktif
            </Button>
          ) : null
        }
      />

      {/* Form Dialog */}
      <AcademicYearFormDialog
        year={editYear}
        open={isCreateOpen || !!editYear}
        onClose={() => {
          setIsCreateOpen(false);
          setEditYear(null);
        }}
      />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteYear}
        onOpenChange={() => setDeleteYear(null)}
        title="Hapus Tahun Ajaran"
        description={
          <>
            Yakin ingin menghapus tahun ajaran{' '}
            <strong>
              {deleteYear?.tahun_mulai}/{deleteYear && deleteYear.tahun_mulai + 1}
            </strong>
            ? Tahun ajaran tidak dapat dihapus jika masih digunakan oleh kelas.
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteYear && deleteMutation.mutate(deleteYear.id)}
      />
    </div>
  );
}

function AcademicYearFormDialog({
  year,
  open,
  onClose,
}: {
  year: AcademicYear | null;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const isEdit = !!year;
  const currentYear = new Date().getFullYear();
  const [tahun, setTahun] = useState(currentYear.toString());
  const [promotionMonth, setPromotionMonth] = useState('7');
  const [promotionDay, setPromotionDay] = useState('1');

  useEffect(() => {
    if (open) {
      if (year) {
        setTahun(year.tahun_mulai.toString());
        setPromotionMonth(year.promotion_month.toString());
        setPromotionDay(year.promotion_day.toString());
      } else {
        setTahun(currentYear.toString());
        setPromotionMonth('7');
        setPromotionDay('1');
      }
    }
  }, [year, currentYear, open]);

  const createMutation = useMutation({
    mutationFn: () =>
      adminAcademicYearsApi.createAcademicYear({
        tahun_mulai: parseInt(tahun),
        promotion_month: parseInt(promotionMonth),
        promotion_day: parseInt(promotionDay),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-academic-years'] });
      toast.success('Tahun ajaran berhasil dibuat');
      onClose();
    },
    onError: () => {
      toast.error('Gagal membuat tahun ajaran');
    },
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      adminAcademicYearsApi.updateAcademicYear(year!.id, {
        tahun_mulai: parseInt(tahun),
        promotion_month: parseInt(promotionMonth),
        promotion_day: parseInt(promotionDay),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-academic-years'] });
      toast.success('Tahun ajaran berhasil diperbarui');
      onClose();
    },
    onError: () => {
      toast.error('Gagal memperbarui tahun ajaran');
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
  const tahunNum = parseInt(tahun);
  const maxDays = new Date(2024, parseInt(promotionMonth), 0).getDate();
  const selectedMonth = MONTHS.find((m) => m.value === parseInt(promotionMonth));

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-primary/10 p-2">
              <Calendar className="h-5 w-5 text-primary" />
            </div>
            <div>
              <DialogTitle>{isEdit ? 'Edit Tahun Ajaran' : 'Tambah Tahun Ajaran Baru'}</DialogTitle>
              <DialogDescription>
                {isEdit
                  ? 'Ubah informasi tahun ajaran yang sudah ada'
                  : 'Tambahkan tahun ajaran baru untuk sistem'}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 pt-4">
          {/* Tahun Mulai */}
          <div className="space-y-2">
            <Label htmlFor="tahun">
              Tahun Mulai <span className="text-destructive">*</span>
            </Label>
            <Input
              id="tahun"
              type="number"
              min="2000"
              max="2100"
              value={tahun}
              onChange={(e) => setTahun(e.target.value)}
              required
            />
            {tahunNum >= 2000 && (
              <p className="text-sm text-muted-foreground">
                Tahun Ajaran:{' '}
                <span className="font-medium text-foreground">
                  {tahunNum}/{tahunNum + 1}
                </span>
              </p>
            )}
          </div>

          {/* Tanggal Kenaikan */}
          <div className="space-y-3">
            <div>
              <Label>
                Tanggal Kenaikan Kelas <span className="text-destructive">*</span>
              </Label>
              <p className="text-xs text-muted-foreground">
                Siswa akan otomatis naik kelas pada tanggal ini setiap tahun
              </p>
            </div>
            <div className="flex gap-2">
              <div className="flex-1">
                <Select value={promotionMonth} onValueChange={setPromotionMonth}>
                  <SelectTrigger>
                    <SelectValue placeholder="Bulan" />
                  </SelectTrigger>
                  <SelectContent>
                    {MONTHS.map((month) => (
                      <SelectItem key={month.value} value={month.value.toString()}>
                        {month.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="w-24">
                <Input
                  type="number"
                  min="1"
                  max={maxDays}
                  value={promotionDay}
                  onChange={(e) => setPromotionDay(e.target.value)}
                  placeholder="Tgl"
                />
              </div>
            </div>
          </div>

          {/* Preview */}
          <div className="rounded-lg border bg-muted/50 p-3">
            <p className="text-xs font-medium text-muted-foreground">Kenaikan kelas otomatis:</p>
            <p className="mt-1 font-semibold">
              {promotionDay} {selectedMonth?.label} setiap tahun
            </p>
          </div>

          <DialogFooter className="pt-4">
            <Button type="button" variant="outline" onClick={onClose}>
              Batal
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isEdit ? 'Simpan Perubahan' : 'Buat Tahun Ajaran'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
