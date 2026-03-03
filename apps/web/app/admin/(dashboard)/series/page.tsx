'use client';

import React, { useState } from 'react';
import dynamic from 'next/dynamic';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Plus,
  Loader2,
  Layers,
  Trash2,
  FileText,
  Image,
  Youtube,
  Table,
  Link,
  Eye,
  ChevronDown,
  ChevronUp,
  FileDown,
  Search,
} from 'lucide-react';

// Dynamic import for PDF export modal (client-side only)
const ExportPdfModal = dynamic(
  () => import('@/components/admin/pdf-export/export-pdf-modal').then((mod) => mod.ExportPdfModal),
  { ssr: false }
);
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
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Card, CardContent } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import { DataTable, Column } from '@/components/admin/data-table';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { adminSeriesApi } from '@/lib/api/admin';
import { Series, SeriesDetail, ContentBlockType } from '@/lib/types';

const BLOCK_TYPES: { value: ContentBlockType; label: string; icon: React.ReactNode; description: string }[] = [
  { value: 'text', label: 'Text', icon: <FileText className="h-4 w-4" />, description: 'Konten teks/paragraf' },
  { value: 'image', label: 'Image', icon: <Image className="h-4 w-4" />, description: 'Gambar/foto' },
  { value: 'youtube', label: 'YouTube', icon: <Youtube className="h-4 w-4" />, description: 'Video YouTube' },
  { value: 'table', label: 'Table', icon: <Table className="h-4 w-4" />, description: 'Tabel data' },
  { value: 'button', label: 'Button', icon: <Link className="h-4 w-4" />, description: 'Tombol link' },
];

function getBlockIcon(type: ContentBlockType) {
  const block = BLOCK_TYPES.find((b) => b.value === type);
  return block?.icon || <FileText className="h-4 w-4" />;
}

function getBlockLabel(type: ContentBlockType) {
  const block = BLOCK_TYPES.find((b) => b.value === type);
  return block?.label || type;
}

export default function AdminSeriesPage() {
  const queryClient = useQueryClient();
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [editSeriesId, setEditSeriesId] = useState<string | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteSeries, setDeleteSeries] = useState<Series | null>(null);
  const [previewSeriesId, setPreviewSeriesId] = useState<string | null>(null);
  const [exportSeries, setExportSeries] = useState<Series | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-series', searchQuery],
    queryFn: () => adminSeriesApi.getSeries({ search: searchQuery || undefined }),
  });

  const handleSearch = () => {
    setSearchQuery(searchInput);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminSeriesApi.deleteSeries(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-series'] });
      toast.success('Series berhasil dihapus');
      setDeleteSeries(null);
    },
    onError: () => {
      toast.error('Gagal menghapus series');
    },
  });

  const toggleActiveMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      adminSeriesApi.updateSeries(id, { is_active }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-series'] });
      toast.success('Status series diperbarui');
    },
    onError: () => {
      toast.error('Gagal memperbarui status');
    },
  });

  const series = data?.data || [];

  const columns: Column<Series>[] = [
    {
      key: 'nama',
      header: 'Nama Series',
      render: (s) => (
        <div className="flex items-center gap-3">
          <div className="rounded-lg bg-primary/10 p-2">
            <Layers className="h-4 w-4 text-primary" />
          </div>
          <div>
            <p className="font-medium">{s.nama}</p>
            {s.deskripsi && (
              <p className="text-xs text-muted-foreground line-clamp-1 max-w-[300px]">
                {s.deskripsi}
              </p>
            )}
          </div>
        </div>
      ),
    },
    {
      key: 'block_count',
      header: 'Blocks',
      render: (s) => (
        <Badge variant="outline" className="gap-1">
          <Layers className="h-3 w-3" />
          {s.block_count || 0}
        </Badge>
      ),
    },
    {
      key: 'portfolio_count',
      header: 'Portfolios',
      render: (s) => <span className="text-muted-foreground">{s.portfolio_count || 0}</span>,
    },
    {
      key: 'is_active',
      header: 'Status',
      render: (s) => (
        <div className="flex items-center gap-2">
          <Switch
            checked={s.is_active}
            onCheckedChange={(checked) =>
              toggleActiveMutation.mutate({ id: s.id, is_active: checked })
            }
          />
          <Badge
            variant={s.is_active ? 'default' : 'secondary'}
            className={
              s.is_active
                ? 'bg-green-100 text-green-700 hover:bg-green-100 dark:bg-green-900/30 dark:text-green-400'
                : ''
            }
          >
            {s.is_active ? 'Aktif' : 'Nonaktif'}
          </Badge>
        </div>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (s) => (
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setPreviewSeriesId(s.id);
            }}
          >
            <Eye className="h-4 w-4 mr-1" />
            Preview
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setExportSeries(s);
            }}
          >
            <FileDown className="h-4 w-4 mr-1" />
            Export
          </Button>
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
            placeholder="Cari series..."
            className="pl-9"
          />
        </div>
        <Button variant="secondary" onClick={handleSearch}>
          <Search className="mr-2 h-4 w-4" />
          Cari
        </Button>
        <Button onClick={() => setIsCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Tambah Series
        </Button>
      </div>

      <DataTable
        data={series}
        columns={columns}
        isLoading={isLoading}
        onEdit={(s) => setEditSeriesId(s.id)}
        onDelete={setDeleteSeries}
      />

      <SeriesFormDialog
        seriesId={editSeriesId}
        open={isCreateOpen || !!editSeriesId}
        onClose={() => {
          setIsCreateOpen(false);
          setEditSeriesId(null);
        }}
      />

      <SeriesPreviewDialog
        seriesId={previewSeriesId}
        open={!!previewSeriesId}
        onClose={() => setPreviewSeriesId(null)}
      />

      <ConfirmDialog
        open={!!deleteSeries}
        onOpenChange={() => setDeleteSeries(null)}
        title="Hapus Series"
        description={
          <>
            Yakin ingin menghapus series <strong>&quot;{deleteSeries?.nama}&quot;</strong>?
            {(deleteSeries?.portfolio_count || 0) > 0 && (
              <span className="block mt-2 text-amber-600">
                ⚠️ {deleteSeries?.portfolio_count} portfolio menggunakan series ini.
                Portfolio tidak akan terhapus, hanya relasi series-nya yang dihapus.
              </span>
            )}
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteSeries && deleteMutation.mutate(deleteSeries.id)}
      />

      <ExportPdfModal
        series={exportSeries}
        open={!!exportSeries}
        onClose={() => setExportSeries(null)}
      />
    </div>
  );
}

interface BlockItem {
  id: string;
  block_type: ContentBlockType;
  instruksi: string;
}

function SeriesFormDialog({
  seriesId,
  open,
  onClose,
}: {
  seriesId: string | null;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const isEdit = !!seriesId;
  const [nama, setNama] = useState('');
  const [deskripsi, setDeskripsi] = useState('');
  const [isActive, setIsActive] = useState(true);
  const [blocks, setBlocks] = useState<BlockItem[]>([]);
  const [isInitialized, setIsInitialized] = useState(false);

  const { data: seriesData, isLoading: isLoadingDetail } = useQuery({
    queryKey: ['admin-series-detail', seriesId],
    queryFn: () => adminSeriesApi.getSeriesById(seriesId!),
    enabled: !!seriesId && open,
  });

  // Reset form when dialog opens/closes or when switching between create/edit
  React.useEffect(() => {
    if (open) {
      if (seriesId && seriesData?.data) {
        const s = seriesData.data;
        setNama(s.nama);
        setDeskripsi(s.deskripsi || '');
        setIsActive(s.is_active);
        setBlocks(
          (s.blocks || []).map((b, idx) => ({
            id: `block-${idx}-${Date.now()}`,
            block_type: b.block_type,
            instruksi: b.instruksi,
          }))
        );
        setIsInitialized(true);
      } else if (!seriesId) {
        // Reset for create mode
        setNama('');
        setDeskripsi('');
        setIsActive(true);
        setBlocks([]);
        setIsInitialized(true);
      }
    } else {
      setIsInitialized(false);
    }
  }, [seriesData, seriesId, open]);

  const createMutation = useMutation({
    mutationFn: () =>
      adminSeriesApi.createSeries({
        nama,
        deskripsi: deskripsi || undefined,
        is_active: isActive,
        blocks: blocks.map((b) => ({
          block_type: b.block_type,
          instruksi: b.instruksi,
        })),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-series'] });
      toast.success('Series berhasil dibuat');
      onClose();
    },
    onError: () => {
      toast.error('Gagal membuat series');
    },
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      adminSeriesApi.updateSeries(seriesId!, {
        nama,
        deskripsi: deskripsi || undefined,
        is_active: isActive,
        blocks: blocks.map((b) => ({
          block_type: b.block_type,
          instruksi: b.instruksi,
        })),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-series'] });
      queryClient.invalidateQueries({ queryKey: ['admin-series-detail', seriesId] });
      toast.success('Series berhasil diperbarui');
      onClose();
    },
    onError: () => {
      toast.error('Gagal memperbarui series');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (blocks.length === 0) {
      toast.error('Series harus memiliki minimal 1 block');
      return;
    }
    if (blocks.some((b) => !b.instruksi.trim())) {
      toast.error('Semua block harus memiliki instruksi');
      return;
    }
    if (isEdit) {
      updateMutation.mutate();
    } else {
      createMutation.mutate();
    }
  };

  const addBlock = () => {
    setBlocks([
      ...blocks,
      {
        id: `block-${blocks.length}-${Date.now()}`,
        block_type: 'text',
        instruksi: '',
      },
    ]);
  };

  const removeBlock = (id: string) => {
    setBlocks(blocks.filter((b) => b.id !== id));
  };

  const updateBlock = (id: string, updates: Partial<BlockItem>) => {
    setBlocks(blocks.map((b) => (b.id === id ? { ...b, ...updates } : b)));
  };

  const moveBlock = (index: number, direction: 'up' | 'down') => {
    const newBlocks = [...blocks];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= blocks.length) return;
    [newBlocks[index], newBlocks[targetIndex]] = [newBlocks[targetIndex], newBlocks[index]];
    setBlocks(newBlocks);
  };

  const isPending = createMutation.isPending || updateMutation.isPending;
  const showLoading = isEdit && (isLoadingDetail || !isInitialized);

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-primary/10 p-2">
              <Layers className="h-5 w-5 text-primary" />
            </div>
            <div>
              <DialogTitle>
                {isEdit ? 'Edit Series Template' : 'Buat Series Template Baru'}
              </DialogTitle>
              <DialogDescription>
                {isEdit
                  ? 'Ubah informasi dan template blocks series'
                  : 'Buat series baru dengan template blocks untuk portofolio'}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {showLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-6">
            <div className="max-h-[60vh] overflow-y-auto pr-2 space-y-6">
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="nama">
                    Nama Series <span className="text-destructive">*</span>
                  </Label>
                  <Input
                    id="nama"
                    value={nama}
                    onChange={(e) => setNama(e.target.value)}
                    placeholder="Contoh: PJBL Semester 1 2024"
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="deskripsi">Deskripsi</Label>
                  <Textarea
                    id="deskripsi"
                    value={deskripsi}
                    onChange={(e) => setDeskripsi(e.target.value)}
                    placeholder="Deskripsi singkat tentang series ini..."
                    rows={3}
                  />
                </div>

                <div className="flex items-center justify-between rounded-lg border p-4">
                  <div>
                    <Label>Status Aktif</Label>
                    <p className="text-xs text-muted-foreground">
                      Series aktif akan ditampilkan ke siswa
                    </p>
                  </div>
                  <Switch checked={isActive} onCheckedChange={setIsActive} />
                </div>
              </div>

              <Separator />

              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <Label>
                      Template Blocks <span className="text-destructive">*</span>
                    </Label>
                    <p className="text-xs text-muted-foreground">
                      Tentukan struktur block konten untuk portofolio
                    </p>
                  </div>
                  <Button type="button" variant="outline" size="sm" onClick={addBlock}>
                    <Plus className="mr-2 h-4 w-4" />
                    Tambah Block
                  </Button>
                </div>

                {blocks.length === 0 ? (
                  <Card className="border-dashed">
                    <CardContent className="flex flex-col items-center justify-center py-8">
                      <Layers className="h-8 w-8 text-muted-foreground mb-2" />
                      <p className="text-sm text-muted-foreground">
                        Belum ada block. Klik &quot;Tambah Block&quot; untuk memulai.
                      </p>
                    </CardContent>
                  </Card>
                ) : (
                  <div className="space-y-3">
                    {blocks.map((block, index) => (
                      <Card key={block.id} className="relative">
                        <CardContent className="pt-4 pb-4">
                          <div className="flex gap-3">
                            <div className="flex flex-col items-center gap-1">
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6"
                                disabled={index === 0}
                                onClick={() => moveBlock(index, 'up')}
                              >
                                <ChevronUp className="h-4 w-4" />
                              </Button>
                              <div className="flex items-center justify-center h-6 w-6 rounded bg-muted text-xs font-medium">
                                {index + 1}
                              </div>
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6"
                                disabled={index === blocks.length - 1}
                                onClick={() => moveBlock(index, 'down')}
                              >
                                <ChevronDown className="h-4 w-4" />
                              </Button>
                            </div>

                            <div className="flex-1 space-y-3">
                              <div className="flex items-center gap-2">
                                <Select
                                  value={block.block_type}
                                  onValueChange={(v) =>
                                    updateBlock(block.id, { block_type: v as ContentBlockType })
                                  }
                                >
                                  <SelectTrigger className="w-[160px]">
                                    <SelectValue />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {BLOCK_TYPES.map((type) => (
                                      <SelectItem key={type.value} value={type.value}>
                                        <div className="flex items-center gap-2">
                                          {type.icon}
                                          <span>{type.label}</span>
                                        </div>
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>

                                <Button
                                  type="button"
                                  variant="ghost"
                                  size="icon"
                                  className="h-8 w-8 text-destructive hover:text-destructive ml-auto"
                                  onClick={() => removeBlock(block.id)}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </div>

                              <div className="space-y-1">
                                <Label className="text-xs">Instruksi untuk siswa</Label>
                                <Textarea
                                  value={block.instruksi}
                                  onChange={(e) =>
                                    updateBlock(block.id, { instruksi: e.target.value })
                                  }
                                  placeholder="Contoh: Masukkan judul dan deskripsi singkat proyek..."
                                  rows={2}
                                  className="text-sm"
                                />
                              </div>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                )}
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={onClose}>
                Batal
              </Button>
              <Button type="submit" disabled={isPending || !nama.trim() || blocks.length === 0}>
                {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {isEdit ? 'Simpan Perubahan' : 'Buat Series'}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}

function SeriesPreviewDialog({
  seriesId,
  open,
  onClose,
}: {
  seriesId: string | null;
  open: boolean;
  onClose: () => void;
}) {
  const { data, isLoading } = useQuery({
    queryKey: ['admin-series-detail', seriesId],
    queryFn: () => adminSeriesApi.getSeriesById(seriesId!),
    enabled: !!seriesId && open,
  });

  const series = data?.data;

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Eye className="h-5 w-5" />
            Preview Template
          </DialogTitle>
          <DialogDescription>
            Tampilan template blocks yang akan dilihat siswa
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : series ? (
          <ScrollArea className="max-h-[60vh] pr-2">
            <div className="space-y-4">
              <div>
                <h3 className="font-semibold text-lg">{series.nama}</h3>
                {series.deskripsi && (
                  <p className="text-sm text-muted-foreground mt-1">{series.deskripsi}</p>
                )}
              </div>

              <Separator />

              {series.blocks && series.blocks.length > 0 ? (
                <div className="space-y-3">
                  <p className="text-sm font-medium">Template Blocks ({series.blocks.length})</p>
                  {series.blocks.map((block, index) => (
                    <Card key={block.id || index}>
                      <CardContent className="pt-4 pb-4">
                        <div className="flex items-start gap-3">
                          <div className="flex items-center justify-center h-8 w-8 rounded-full bg-primary/10 text-primary shrink-0">
                            {getBlockIcon(block.block_type)}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 mb-1">
                              <span className="text-xs font-medium text-muted-foreground">
                                Block {index + 1}
                              </span>
                              <Badge variant="outline" className="text-xs">
                                {getBlockLabel(block.block_type)}
                              </Badge>
                            </div>
                            <p className="text-sm">{block.instruksi}</p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground text-center py-8">
                  Series ini belum memiliki blocks
                </p>
              )}
            </div>
          </ScrollArea>
        ) : (
          <p className="text-sm text-muted-foreground text-center py-8">
            Series tidak ditemukan
          </p>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Tutup
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
