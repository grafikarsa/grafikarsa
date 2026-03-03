'use client';

import { useState, useEffect, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { pdf } from '@react-pdf/renderer';
import QRCode from 'qrcode';
import { toast } from 'sonner';
import {
  FileDown,
  Loader2,
  FileText,
  Users,
  BookOpen,
} from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Progress } from '@/components/ui/progress';
import { adminSeriesApi, adminMajorsApi, adminClassesApi, Major, Class, SeriesExportResponse } from '@/lib/api/admin';
import { Series } from '@/lib/types';
import { PdfDocument } from './pdf-document';

interface ExportPdfModalProps {
  series: Series | null;
  open: boolean;
  onClose: () => void;
}

export function ExportPdfModal({ series, open, onClose }: ExportPdfModalProps) {
  const [jurusanId, setJurusanId] = useState<string>('all');
  const [kelasId, setKelasId] = useState<string>('all');
  const [isGenerating, setIsGenerating] = useState(false);
  const [progress, setProgress] = useState(0);
  const [progressMessage, setProgressMessage] = useState('');

  // Reset filters when modal opens
  useEffect(() => {
    if (open) {
      setJurusanId('all');
      setKelasId('all');
      setProgress(0);
      setProgressMessage('');
    }
  }, [open]);

  // Fetch majors
  const { data: majorsData } = useQuery({
    queryKey: ['admin-majors'],
    queryFn: () => adminMajorsApi.getMajors(),
    enabled: open,
  });

  // Fetch classes (filtered by jurusan if selected)
  const { data: classesData } = useQuery({
    queryKey: ['admin-classes', jurusanId],
    queryFn: () =>
      adminClassesApi.getClasses({
        jurusan_id: jurusanId !== 'all' ? jurusanId : undefined,
        limit: 100,
      }),
    enabled: open,
  });

  // Fetch export preview
  const { data: previewData, isLoading: isLoadingPreview } = useQuery({
    queryKey: ['export-preview', series?.id, jurusanId, kelasId],
    queryFn: () =>
      adminSeriesApi.getExportPreview(series!.id, {
        jurusan_id: jurusanId !== 'all' ? jurusanId : undefined,
        kelas_id: kelasId !== 'all' ? kelasId : undefined,
      }),
    enabled: open && !!series?.id,
  });

  const majors = majorsData?.data || [];
  const classes = classesData?.data || [];
  const preview = previewData?.data;

  // Reset kelas when jurusan changes
  useEffect(() => {
    setKelasId('all');
  }, [jurusanId]);

  const generateQrCodes = useCallback(async (usernames: string[]): Promise<Map<string, string>> => {
    const qrCodes = new Map<string, string>();
    for (const username of usernames) {
      try {
        const url = `https://grafikarsa.com/${username}`;
        const dataUrl = await QRCode.toDataURL(url, {
          width: 120,
          margin: 1,
          color: { dark: '#000000', light: '#ffffff' },
        });
        qrCodes.set(username, dataUrl);
      } catch (err) {
        console.error(`Failed to generate QR for ${username}:`, err);
      }
    }
    return qrCodes;
  }, []);

  // Fetch image and convert to base64
  const fetchImageAsBase64 = useCallback(async (url: string): Promise<string | null> => {
    try {
      const response = await fetch(url);
      if (!response.ok) return null;
      const blob = await response.blob();
      return new Promise((resolve) => {
        const reader = new FileReader();
        reader.onloadend = () => resolve(reader.result as string);
        reader.onerror = () => resolve(null);
        reader.readAsDataURL(blob);
      });
    } catch {
      return null;
    }
  }, []);

  // Fetch all images and cache them
  const fetchAllImages = useCallback(async (exportData: SeriesExportResponse): Promise<Map<string, string>> => {
    const imageCache = new Map<string, string>();
    const urls = new Set<string>();

    // Collect all image URLs
    for (const portfolio of exportData.portfolios) {
      if (portfolio.user.avatar_url) urls.add(portfolio.user.avatar_url);
      if (portfolio.thumbnail_url) urls.add(portfolio.thumbnail_url);
      for (const block of portfolio.content_blocks) {
        if (block.block_type === 'image' && block.payload.url) {
          urls.add(String(block.payload.url));
        }
      }
    }

    // Fetch all images in parallel (max 5 concurrent)
    const urlArray = Array.from(urls);
    for (let i = 0; i < urlArray.length; i += 5) {
      const batch = urlArray.slice(i, i + 5);
      const results = await Promise.all(batch.map(fetchImageAsBase64));
      batch.forEach((url, idx) => {
        if (results[idx]) imageCache.set(url, results[idx]!);
      });
    }

    return imageCache;
  }, [fetchImageAsBase64]);

  const handleExport = async () => {
    if (!series) return;

    setIsGenerating(true);
    setProgress(10);
    setProgressMessage('Mengambil data portofolio...');

    try {
      // Fetch export data
      const response = await adminSeriesApi.getExportData(series.id, {
        jurusan_id: jurusanId !== 'all' ? jurusanId : undefined,
        kelas_id: kelasId !== 'all' ? kelasId : undefined,
      });

      if (!response.data || response.data.portfolios.length === 0) {
        toast.error('Tidak ada portofolio untuk di-export');
        setIsGenerating(false);
        return;
      }

      const data = response.data;
      setProgress(20);
      setProgressMessage(`Memproses ${data.portfolios.length} portofolio...`);

      // Generate QR codes
      setProgress(30);
      setProgressMessage('Membuat QR codes...');
      const usernames = [...new Set(data.portfolios.map((p) => p.user.username))];
      const qrCodes = await generateQrCodes(usernames);

      // Fetch logo
      setProgress(40);
      setProgressMessage('Mengambil logo...');
      const logoBase64 = await fetchImageAsBase64('/logo.png');

      // Fetch all images
      setProgress(50);
      setProgressMessage('Mengambil gambar...');
      const imageCache = await fetchAllImages(data);

      setProgress(70);
      setProgressMessage('Membuat dokumen PDF...');

      // Generate PDF
      const doc = <PdfDocument data={data} qrCodes={qrCodes} imageCache={imageCache} logoBase64={logoBase64 || undefined} />;
      const blob = await pdf(doc).toBlob();

      setProgress(90);
      setProgressMessage('Menyiapkan download...');

      // Download
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      const timestamp = new Date().toISOString().slice(0, 10);
      // Include usernames in filename (max 3)
      const usernameList = usernames.slice(0, 3).join('_');
      const usernameSuffix = usernames.length > 3 ? `_dan_${usernames.length - 3}_lainnya` : '';
      const filename = `${series.nama.replace(/[^a-zA-Z0-9]/g, '_')}_${usernameList}${usernameSuffix}_${timestamp}.pdf`;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);

      setProgress(100);
      setProgressMessage('Selesai!');
      toast.success(`PDF berhasil di-download: ${filename}`);

      setTimeout(() => {
        onClose();
      }, 1000);
    } catch (error) {
      console.error('Export error:', error);
      toast.error('Gagal membuat PDF');
    } finally {
      setIsGenerating(false);
    }
  };

  if (!series) return null;

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="rounded-full bg-primary/10 p-2">
              <FileDown className="h-5 w-5 text-primary" />
            </div>
            <div>
              <DialogTitle>Export PDF</DialogTitle>
              <DialogDescription>{series.nama}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {isGenerating ? (
          <div className="py-8">
            <div className="flex flex-col items-center text-center">
              <Loader2 className="h-10 w-10 animate-spin text-primary mb-4" />
              <p className="text-sm font-medium">{progressMessage}</p>
              <Progress value={progress} className="mt-4 h-2 w-full" />
              <p className="mt-2 text-xs text-muted-foreground">{Math.round(progress)}%</p>
            </div>
          </div>
        ) : (
          <>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label>Jurusan</Label>
                <Select value={jurusanId} onValueChange={setJurusanId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Pilih jurusan" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Semua Jurusan</SelectItem>
                    {majors.map((m) => (
                      <SelectItem key={m.id} value={m.id}>
                        {m.nama}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label>Kelas</Label>
                <Select value={kelasId} onValueChange={setKelasId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Pilih kelas" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Semua Kelas</SelectItem>
                    {classes.map((c) => (
                      <SelectItem key={c.id} value={c.id}>
                        {c.nama}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Preview */}
              <div className="rounded-lg border bg-muted/30 p-4 space-y-2">
                <p className="text-sm font-medium text-muted-foreground">Preview</p>
                {isLoadingPreview ? (
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Menghitung...
                  </div>
                ) : preview ? (
                  <div className="space-y-1">
                    <div className="flex items-center gap-2 text-sm">
                      <BookOpen className="h-4 w-4 text-muted-foreground" />
                      <span>{preview.portfolio_count} portofolio</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <Users className="h-4 w-4 text-muted-foreground" />
                      <span>Dari {preview.user_count} siswa</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <FileText className="h-4 w-4 text-muted-foreground" />
                      <span>Estimasi ~{preview.estimated_pages} halaman</span>
                    </div>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">Tidak ada data</p>
                )}
              </div>

              {preview && preview.portfolio_count === 0 && (
                <p className="text-sm text-amber-600">
                  ⚠️ Tidak ada portofolio published yang sesuai filter
                </p>
              )}
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={onClose}>
                Batal
              </Button>
              <Button
                onClick={handleExport}
                disabled={!preview || preview.portfolio_count === 0}
              >
                <FileDown className="mr-2 h-4 w-4" />
                Download PDF
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
