'use client';

import { useState, useEffect, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { pdf } from '@react-pdf/renderer';
import { PDFDocument, StandardFonts, rgb } from 'pdf-lib';
import QRCode from 'qrcode';
import { toast } from 'sonner';
import {
  FileDown,
  Loader2,
  FileText,
  Users,
  BookOpen,
  CheckCircle2,
  Circle,
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
import { adminSeriesApi, SeriesExportResponse } from '@/lib/api/admin';
import { publicApi } from '@/lib/api/public';
import { Series } from '@/lib/types';
import { PdfDocument } from './pdf-document';

// ── Export Step Types ──────────────────────────────────────────────────────
type StepStatus = 'pending' | 'active' | 'done';

interface ExportStep {
  id: string;
  label: string;
  status: StepStatus;
  detail?: string;
}

const STEP_DEFS = [
  { id: 'fetch', label: 'Mengambil data portofolio' },
  { id: 'process', label: 'Memproses portofolio' },
  { id: 'qr', label: 'Membuat QR codes' },
  { id: 'images', label: 'Mengambil gambar' },
  { id: 'bg', label: 'Menyiapkan background' },
  { id: 'render', label: 'Membuat dokumen PDF' },
  { id: 'number', label: 'Menambahkan nomor halaman' },
  { id: 'download', label: 'Menyiapkan download' },
] as const;

function formatElapsed(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return m > 0 ? `${m}m ${s}s` : `${s}s`;
}

function makeInitialSteps(): ExportStep[] {
  return STEP_DEFS.map((def) => ({ ...def, status: 'pending' as StepStatus }));
}

interface ExportPdfModalProps {
  series: Series | null;
  open: boolean;
  onClose: () => void;
}

export function ExportPdfModal({ series, open, onClose }: ExportPdfModalProps) {
  const [jurusanId, setJurusanId] = useState<string>('all');
  const [kelasId, setKelasId] = useState<string>('all');
  const [isGenerating, setIsGenerating] = useState(false);
  const [steps, setSteps] = useState<ExportStep[]>([]);
  const [elapsedTime, setElapsedTime] = useState(0);

  useEffect(() => {
    if (open) {
      setJurusanId('all');
      setKelasId('all');
      setSteps([]);
      setElapsedTime(0);
    }
  }, [open]);

  // Elapsed timer
  useEffect(() => {
    if (!isGenerating) return;
    setElapsedTime(0);
    const timer = setInterval(() => setElapsedTime((t) => t + 1), 1000);
    return () => clearInterval(timer);
  }, [isGenerating]);

  const { data: majorsData } = useQuery({
    queryKey: ['public-jurusan'],
    queryFn: () => publicApi.getJurusan(),
    enabled: open,
  });

  const { data: classesData } = useQuery({
    queryKey: ['public-kelas', jurusanId],
    queryFn: () =>
      publicApi.getKelas({
        jurusan_id: jurusanId !== 'all' ? jurusanId : undefined,
      }),
    enabled: open,
  });

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

  useEffect(() => {
    setKelasId('all');
  }, [jurusanId]);

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

  const fetchAllImages = useCallback(async (
    exportData: SeriesExportResponse,
    onProgress?: (fetched: number, total: number) => void,
  ): Promise<Map<string, string>> => {
    const imageCache = new Map<string, string>();
    const urls = new Set<string>();

    for (const portfolio of exportData.portfolios) {
      if (portfolio.user.avatar_url) urls.add(portfolio.user.avatar_url);
      if (portfolio.thumbnail_url) urls.add(portfolio.thumbnail_url);
      for (const block of portfolio.content_blocks) {
        if (block.block_type === 'image' && block.payload.url) {
          urls.add(String(block.payload.url));
        }
      }
    }

    const urlArray = Array.from(urls);
    let fetched = 0;
    for (let i = 0; i < urlArray.length; i += 5) {
      const batch = urlArray.slice(i, i + 5);
      const results = await Promise.all(batch.map(fetchImageAsBase64));
      batch.forEach((url, idx) => {
        if (results[idx]) imageCache.set(url, results[idx]!);
      });
      fetched += batch.length;
      onProgress?.(fetched, urlArray.length);
    }

    return imageCache;
  }, [fetchImageAsBase64]);

  const generateQrCodes = useCallback(async (
    usernames: string[],
    onProgress?: (done: number, total: number) => void,
  ): Promise<Map<string, string>> => {
    const qrCodes = new Map<string, string>();
    for (let i = 0; i < usernames.length; i++) {
      const username = usernames[i];
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
      onProgress?.(i + 1, usernames.length);
    }
    return qrCodes;
  }, []);

  const addPageNumbers = async (blob: Blob): Promise<Blob> => {
    const pdfBytes = await blob.arrayBuffer();
    const pdfDoc = await PDFDocument.load(pdfBytes);
    const helvetica = await pdfDoc.embedFont(StandardFonts.Helvetica);
    const pages = pdfDoc.getPages();
    const totalPages = pages.length;

    for (let i = 0; i < pages.length; i++) {
      const page = pages[i];
      const { width } = page.getSize();
      const text = `Halaman ${i + 1} dari ${totalPages}`;
      const textWidth = helvetica.widthOfTextAtSize(text, 7);

      page.drawText(text, {
        x: (width - textWidth) / 2,
        y: 20,
        size: 7,
        font: helvetica,
        color: rgb(1, 1, 1),
      });
    }

    const finalBytes = await pdfDoc.save();
    return new Blob([new Uint8Array(finalBytes)], { type: 'application/pdf' });
  };

  const handleExport = async () => {
    if (!series) return;

    setIsGenerating(true);
    const initSteps = makeInitialSteps();
    setSteps(initSteps);

    try {
      // Step 1: Fetch data
      initSteps[0].status = 'active';
      setSteps([...initSteps]);

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
      initSteps[0].status = 'done';

      // Step 2: Process
      initSteps[1].status = 'active';
      initSteps[1].detail = `${data.portfolios.length} portofolio`;
      setSteps([...initSteps]);

      // Build filter labels
      const jurusanLabel = jurusanId === 'all'
        ? 'Semua Jurusan'
        : (majors.find((m) => m.id === jurusanId)?.nama || jurusanId);
      const kelasLabel = kelasId === 'all'
        ? 'Semua Kelas'
        : (classes.find((c) => c.id === kelasId)?.nama || kelasId);

      initSteps[1].status = 'done';

      // Step 3: QR codes
      initSteps[2].status = 'active';
      setSteps([...initSteps]);

      const usernames = [...new Set(data.portfolios.map((p) => p.user.username))];
      const qrCodes = await generateQrCodes(usernames, (done, total) => {
        initSteps[2].detail = `${done}/${total}`;
        setSteps([...initSteps]);
      });
      initSteps[2].status = 'done';
      initSteps[2].detail = `${usernames.length} kode`;

      // Step 4: Fetch images
      initSteps[3].status = 'active';
      setSteps([...initSteps]);

      const imageCache = await fetchAllImages(data, (fetched, total) => {
        initSteps[3].detail = `${fetched}/${total}`;
        setSteps([...initSteps]);
      });
      initSteps[3].status = 'done';
      initSteps[3].detail = `${imageCache.size} gambar`;

      // Step 5: Background
      initSteps[4].status = 'active';
      setSteps([...initSteps]);

      const bgImageBase64 = await fetchImageAsBase64('/images/export/bg.png');
      initSteps[4].status = 'done';

      // Step 6: Render PDF
      initSteps[5].status = 'active';
      setSteps([...initSteps]);

      const doc = <PdfDocument data={data} qrCodes={qrCodes} imageCache={imageCache} jurusanLabel={jurusanLabel} kelasLabel={kelasLabel} bgImage={bgImageBase64 || undefined} />;
      const blob = await pdf(doc).toBlob();
      initSteps[5].status = 'done';

      // Step 7: Page numbers
      initSteps[6].status = 'active';
      setSteps([...initSteps]);

      const finalBlob = await addPageNumbers(blob);
      initSteps[6].status = 'done';

      // Step 8: Download
      initSteps[7].status = 'active';
      setSteps([...initSteps]);

      const url = URL.createObjectURL(finalBlob);
      const link = document.createElement('a');
      link.href = url;
      const timestamp = new Date().toISOString().slice(0, 10);
      const usernameList = usernames.slice(0, 3).join('_');
      const usernameSuffix = usernames.length > 3 ? `_dan_${usernames.length - 3}_lainnya` : '';
      const filename = `${series.nama.replace(/[^a-zA-Z0-9]/g, '_')}_${usernameList}${usernameSuffix}_${timestamp}.pdf`;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);

      initSteps[7].status = 'done';
      setSteps([...initSteps]);
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
      <DialogContent className="max-w-fit">
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
          <div className="py-4">
            <div className="space-y-2">
              {steps.map((step) => (
                <div key={step.id} className="flex items-center gap-3">
                  {step.status === 'done' ? (
                    <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
                  ) : step.status === 'active' ? (
                    <Loader2 className="h-4 w-4 animate-spin text-primary shrink-0" />
                  ) : (
                    <Circle className="h-4 w-4 text-muted-foreground/40 shrink-0" />
                  )}
                  <div className="flex-1 min-w-0">
                    <span className={`text-sm ${step.status === 'pending' ? 'text-muted-foreground/50' : ''}`}>
                      {step.label}
                    </span>
                    {step.detail && (
                      <span className="ml-2 text-xs text-muted-foreground">
                        {step.detail}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
            <div className="mt-4 pt-3 border-t text-center">
              <span className="text-xs text-muted-foreground">
                Waktu berlalu: {formatElapsed(elapsedTime)}
              </span>
            </div>
          </div>
        ) : (
          <>
            <div className="space-y-4 py-4">
              <div className="flex gap-3">
                <div className="flex-1 space-y-2">
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

                <div className="flex-1 space-y-2">
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
