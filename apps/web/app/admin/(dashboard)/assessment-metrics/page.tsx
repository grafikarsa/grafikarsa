'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Plus,
  Loader2,
  BarChart3,
  GripVertical,
  Check,
  X,
  AlertCircle,
  ListChecks,
  CheckCircle2,
  XCircle,
  Search,
} from 'lucide-react';
import { getDebugEmptyState } from '@/lib/utils/debug';
import { DebugBanner } from '@/components/admin/debug-banner';
import { useDebounce } from '@/lib/hooks/use-debounce';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
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
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { adminAssessmentMetricsApi } from '@/lib/api/admin';
import { AssessmentMetric } from '@/lib/types';
import { cn } from '@/lib/utils';

export default function AdminAssessmentMetricsPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [editMetric, setEditMetric] = useState<AssessmentMetric | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteMetric, setDeleteMetric] = useState<AssessmentMetric | null>(null);
  
  const debouncedSearch = useDebounce(search, 300);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin-assessment-metrics'],
    queryFn: () => adminAssessmentMetricsApi.getMetrics(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminAssessmentMetricsApi.deleteMetric(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-assessment-metrics'] });
      toast.success('Metrik berhasil dihapus');
      setDeleteMetric(null);
    },
    onError: (error: Error & { response?: { data?: { message?: string } } }) => {
      const message = error.response?.data?.message || 'Gagal menghapus metrik';
      toast.error(message);
    },
  });

  const toggleActiveMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      adminAssessmentMetricsApi.updateMetric(id, { is_active }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['admin-assessment-metrics'] });
      toast.success(variables.is_active ? 'Metrik diaktifkan' : 'Metrik dinonaktifkan');
    },
    onError: () => {
      toast.error('Gagal mengubah status metrik');
    },
  });

  const metrics = data?.data || [];
  const sortedMetrics = [...metrics].sort((a, b) => a.urutan - b.urutan);
  
  // Filter by search
  const filteredMetrics = debouncedSearch
    ? sortedMetrics.filter((m) =>
        m.nama.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
        m.deskripsi?.toLowerCase().includes(debouncedSearch.toLowerCase())
      )
    : sortedMetrics;
  
  const activeCount = metrics.filter((m) => m.is_active).length;
  const inactiveCount = metrics.filter((m) => !m.is_active).length;

  // Debug mode: Force empty state
  const debugMode = getDebugEmptyState();
  const displayMetrics = debugMode ? [] : filteredMetrics;

  // Loading state
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
              <Skeleton className="h-10 w-40" />
            </div>
          </div>
        </div>
        <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-[1600px]">
            <div className="space-y-3">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-16" />
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (isError) {
    return (
      <div className="flex flex-col -m-4 sm:-m-6 lg:-m-8">
        <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-[1600px]">
            <Card className="border-destructive/50 bg-destructive/5 p-8">
              <div className="flex flex-col items-center justify-center text-center">
                <AlertCircle className="h-12 w-12 text-destructive" />
                <h3 className="mt-4 text-lg font-semibold">Gagal Memuat Data</h3>
                <p className="mt-1 text-sm text-muted-foreground">
                  Terjadi kesalahan saat mengambil data metrik penilaian
                </p>
                <Button onClick={() => refetch()} className="mt-4">
                  Coba Lagi
                </Button>
              </div>
            </Card>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col -m-4 sm:-m-6 lg:-m-8">
      {/* Sticky Header - Stats, Search & Actions (Full Width) */}
      <div className="sticky top-0 z-10 bg-background/95 px-4 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6 lg:px-8 border-b">
        <div className="mx-auto w-full max-w-[1600px] space-y-4">
          {/* Stats Cards */}
          <div className="grid gap-4 sm:grid-cols-3">
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-muted p-2">
                  <ListChecks className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{metrics.length}</p>
                  <p className="text-sm text-muted-foreground">Total Metrik</p>
                </div>
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
                  <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{activeCount}</p>
                  <p className="text-sm text-muted-foreground">Aktif</p>
                </div>
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-gray-100 p-2 dark:bg-gray-800">
                  <XCircle className="h-5 w-5 text-gray-500" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{inactiveCount}</p>
                  <p className="text-sm text-muted-foreground">Nonaktif</p>
                </div>
              </div>
            </Card>
          </div>

          {/* Search & Action */}
          <div className="flex flex-col gap-4 sm:flex-row">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Cari metrik..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Button onClick={() => setIsCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Tambah Metrik
            </Button>
          </div>
        </div>
      </div>

      {/* Content Area with proper padding */}
      <div className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
        <div className="mx-auto w-full max-w-[1600px]">
          {debugMode && <DebugBanner pageName="Metrik Penilaian" />}

          {/* Metrics Table */}
          {displayMetrics.length === 0 ? (
            <div className="flex items-center justify-center min-h-[60vh]">
              <div className="flex flex-col items-center justify-center px-6">
                <div className="rounded-full bg-primary/10 p-4">
                  <BarChart3 className="h-10 w-10 text-primary" />
                </div>
                <h3 className="mt-6 text-xl font-semibold">
                  {search ? 'Tidak ada metrik yang sesuai' : 'Belum ada metrik'}
                </h3>
                <p className="mt-2 text-sm text-muted-foreground text-center max-w-md">
                  {search
                    ? 'Coba ubah kata kunci pencarian untuk menemukan metrik yang Anda cari.'
                    : 'Metrik penilaian digunakan untuk menilai portfolio siswa dengan skala 1-10. Buat metrik pertama untuk mulai menilai portfolio.'}
                </p>
                {!search && (
                  <Button onClick={() => setIsCreateOpen(true)} className="mt-6">
                    <Plus className="mr-2 h-4 w-4" />
                    Buat Metrik Pertama
                  </Button>
                )}
              </div>
            </div>
          ) : (
            <Card className="overflow-hidden p-0">
              <Table>
                <TableHeader>
                  <TableRow className="border-b bg-muted/30">
                    <TableHead className="w-16">#</TableHead>
                    <TableHead>Nama Metrik</TableHead>
                    <TableHead className="hidden md:table-cell">Deskripsi</TableHead>
                    <TableHead className="w-[120px]">Status</TableHead>
                    <TableHead className="w-[100px] text-right">Aksi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {displayMetrics.map((metric) => (
                    <TableRow key={metric.id} className={cn('group hover:bg-muted/50', !metric.is_active && 'opacity-60')}>
                      <TableCell>
                        <div className="flex items-center gap-2 text-muted-foreground">
                          <GripVertical className="h-4 w-4 cursor-grab" />
                          <span className="font-mono text-sm">{metric.urutan}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <BarChart3 className="h-4 w-4 text-primary" />
                          <span className="font-medium">{metric.nama}</span>
                        </div>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="text-muted-foreground text-sm line-clamp-1 cursor-help">
                                {metric.deskripsi || '-'}
                              </span>
                            </TooltipTrigger>
                            {metric.deskripsi && (
                              <TooltipContent side="bottom" className="max-w-xs">
                                <p>{metric.deskripsi}</p>
                              </TooltipContent>
                            )}
                          </Tooltip>
                        </TooltipProvider>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Switch
                            checked={metric.is_active}
                            onCheckedChange={(checked: boolean) =>
                              toggleActiveMutation.mutate({ id: metric.id, is_active: checked })
                            }
                            disabled={toggleActiveMutation.isPending}
                          />
                          <Badge
                            variant={metric.is_active ? 'default' : 'secondary'}
                            className={cn(
                              'text-xs',
                              metric.is_active
                                ? 'bg-green-100 text-green-700 hover:bg-green-100 dark:bg-green-900/30 dark:text-green-400'
                                : ''
                            )}
                          >
                            {metric.is_active ? (
                              <>
                                <Check className="h-3 w-3 mr-1" /> Aktif
                              </>
                            ) : (
                              <>
                                <X className="h-3 w-3 mr-1" /> Nonaktif
                              </>
                            )}
                          </Badge>
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setEditMetric(metric)}
                          >
                            Edit
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive hover:text-destructive"
                            onClick={() => setDeleteMetric(metric)}
                          >
                            Hapus
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Card>
          )}
        </div>
      </div>

      {/* Modals - Outside scrollable area */}
      <MetricFormDialog
        metric={editMetric}
        open={isCreateOpen || !!editMetric}
        onClose={() => {
          setIsCreateOpen(false);
          setEditMetric(null);
        }}
      />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteMetric}
        onOpenChange={() => setDeleteMetric(null)}
        title="Hapus Metrik"
        description={
          <>
            Yakin ingin menghapus metrik <strong>&quot;{deleteMetric?.nama}&quot;</strong>?
            <br />
            <span className="text-muted-foreground text-sm">
              Metrik yang sudah digunakan untuk penilaian tidak dapat dihapus.
            </span>
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteMetric && deleteMutation.mutate(deleteMetric.id)}
      />
    </div>
  );
}




function MetricFormDialog({
  metric,
  open,
  onClose,
}: {
  metric: AssessmentMetric | null;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const isEdit = !!metric;
  const [nama, setNama] = useState('');
  const [deskripsi, setDeskripsi] = useState('');
  const [isActive, setIsActive] = useState(true);
  const [errors, setErrors] = useState<{ nama?: string }>({});

  React.useEffect(() => {
    if (open) {
      if (metric) {
        setNama(metric.nama);
        setDeskripsi(metric.deskripsi || '');
        setIsActive(metric.is_active);
      } else {
        setNama('');
        setDeskripsi('');
        setIsActive(true);
      }
      setErrors({});
    }
  }, [metric, open]);

  const createMutation = useMutation({
    mutationFn: () =>
      adminAssessmentMetricsApi.createMetric({
        nama: nama.trim(),
        deskripsi: deskripsi.trim() || undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-assessment-metrics'] });
      toast.success('Metrik berhasil dibuat');
      onClose();
    },
    onError: (error: Error & { response?: { data?: { message?: string } } }) => {
      const message = error.response?.data?.message || 'Gagal membuat metrik';
      toast.error(message);
    },
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      adminAssessmentMetricsApi.updateMetric(metric!.id, {
        nama: nama.trim(),
        deskripsi: deskripsi.trim() || undefined,
        is_active: isActive,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-assessment-metrics'] });
      toast.success('Metrik berhasil diperbarui');
      onClose();
    },
    onError: (error: Error & { response?: { data?: { message?: string } } }) => {
      const message = error.response?.data?.message || 'Gagal memperbarui metrik';
      toast.error(message);
    },
  });

  const validate = (): boolean => {
    const newErrors: { nama?: string } = {};

    if (!nama.trim()) {
      newErrors.nama = 'Nama metrik wajib diisi';
    } else if (nama.trim().length < 2) {
      newErrors.nama = 'Nama metrik minimal 2 karakter';
    } else if (nama.trim().length > 100) {
      newErrors.nama = 'Nama metrik maksimal 100 karakter';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    if (isEdit) {
      updateMutation.mutate();
    } else {
      createMutation.mutate();
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-primary/10 p-2">
              <BarChart3 className="h-5 w-5 text-primary" />
            </div>
            <div>
              <DialogTitle>{isEdit ? 'Edit Metrik' : 'Tambah Metrik Baru'}</DialogTitle>
              <DialogDescription>
                {isEdit
                  ? 'Ubah detail metrik penilaian'
                  : 'Buat metrik baru untuk penilaian portfolio'}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 pt-4">
          <div className="space-y-2">
            <Label htmlFor="nama">
              Nama Metrik <span className="text-destructive">*</span>
            </Label>
            <Input
              id="nama"
              value={nama}
              onChange={(e) => {
                setNama(e.target.value);
                if (errors.nama) setErrors({});
              }}
              placeholder="Contoh: Kreativitas"
              className={cn(errors.nama && 'border-destructive')}
              autoFocus
            />
            {errors.nama && (
              <p className="text-xs text-destructive">{errors.nama}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="deskripsi">Deskripsi</Label>
            <Textarea
              id="deskripsi"
              value={deskripsi}
              onChange={(e) => setDeskripsi(e.target.value)}
              placeholder="Penjelasan tentang metrik ini untuk panduan penilaian..."
              rows={3}
            />
            <p className="text-xs text-muted-foreground">
              Deskripsi akan ditampilkan sebagai panduan saat admin melakukan penilaian
            </p>
          </div>

          {isEdit && (
            <div className="flex items-center justify-between rounded-lg border p-3 bg-muted/30">
              <div className="space-y-0.5">
                <Label htmlFor="is_active" className="cursor-pointer">
                  Status Aktif
                </Label>
                <p className="text-xs text-muted-foreground">
                  Metrik nonaktif tidak akan muncul saat penilaian baru
                </p>
              </div>
              <Switch
                id="is_active"
                checked={isActive}
                onCheckedChange={setIsActive}
              />
            </div>
          )}

          <DialogFooter className="pt-4 gap-2 sm:gap-0">
            <Button type="button" variant="outline" onClick={onClose} disabled={isPending}>
              Batal
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isEdit ? 'Simpan Perubahan' : 'Buat Metrik'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
