'use client';

import { useState, useEffect, useRef, useMemo } from 'react';
import Image from 'next/image';
import { useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
  DragStartEvent,
  DragOverlay,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Plus,
  Loader2,
  FileText,
  Upload,
  X,
  GripVertical,
  Type,
  ImageIcon,
  Youtube,
  Link2,
  Trash2,
  Send,
  Save,
  Archive,
  ArchiveRestore,
  AlertCircle,
  Eye,
  ChevronDown,
  Check,
  Camera,
  Table as TableIcon,
  // New icons for embed blocks
  Figma,
  Globe,
  Presentation,
  FileUp,
  Download,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { Progress } from '@/components/ui/progress';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { portfoliosApi, tagsApi } from '@/lib/api';
import { seriesApi } from '@/lib/api/public';
import { uploadsApi } from '@/lib/api/admin';
import { useAuthStore } from '@/lib/stores/auth-store';
import { Portfolio, Tag, Series, ContentBlockType, ContentBlockPayload } from '@/lib/types';
import { cn } from '@/lib/utils';
import {
  getFigmaEmbedUrl,
  getCanvaEmbedUrl,
  getGoogleSlidesEmbedUrl,
  getGoogleDocsViewerUrl,
  isValidFigmaUrl,
  isValidCanvaUrl,
  isValidGoogleSlidesUrl,
  isAllowedDomain,
  isValidUrl,
  getAcceptedFileTypes,
  isValidFileType,
  formatFileSize,
  MAX_DOCUMENT_SIZE,
} from '@/lib/utils/embed-helpers';

// Types
interface LocalContentBlock {
  id: string;
  block_type: ContentBlockType;
  block_order: number;
  payload: Record<string, unknown>;
  isNew?: boolean;
  pendingFile?: File;
  pendingPreview?: string;
  seriesInstruksi?: string; // Instruction from series template
}

const blockTypeOptions = [
  { value: 'text', label: 'Teks', icon: Type, description: 'Paragraf atau deskripsi', iconColor: 'text-blue-500', bgColor: 'bg-blue-100 dark:bg-blue-500/20' },
  { value: 'image', label: 'Gambar', icon: ImageIcon, description: 'Foto atau ilustrasi', iconColor: 'text-green-500', bgColor: 'bg-green-100 dark:bg-green-500/20' },
  { value: 'youtube', label: 'Video', icon: Youtube, description: 'Embed YouTube', iconColor: 'text-red-500', bgColor: 'bg-red-100 dark:bg-red-500/20' },
  { value: 'table', label: 'Tabel', icon: TableIcon, description: 'Tabel data', iconColor: 'text-orange-500', bgColor: 'bg-orange-100 dark:bg-orange-500/20' },
  { value: 'button', label: 'Tombol', icon: Link2, description: 'Tombol dengan URL', iconColor: 'text-purple-500', bgColor: 'bg-purple-100 dark:bg-purple-500/20' },
  // New block types
  { value: 'figma', label: 'Figma', icon: Figma, description: 'Embed desain Figma', iconColor: 'text-purple-600', bgColor: 'bg-purple-100 dark:bg-purple-500/20' },
  { value: 'canva', label: 'Canva', icon: ImageIcon, description: 'Embed desain Canva', iconColor: 'text-cyan-500', bgColor: 'bg-cyan-100 dark:bg-cyan-500/20' },
  { value: 'ppt', label: 'Presentasi', icon: Presentation, description: 'Google Slides / PPT', iconColor: 'text-amber-500', bgColor: 'bg-amber-100 dark:bg-amber-500/20' },
  { value: 'pdf', label: 'PDF', icon: FileText, description: 'Dokumen PDF', iconColor: 'text-red-600', bgColor: 'bg-red-100 dark:bg-red-500/20' },
  { value: 'doc', label: 'Dokumen', icon: FileUp, description: 'Word / DOCX', iconColor: 'text-blue-600', bgColor: 'bg-blue-100 dark:bg-blue-500/20' },
];

const generateId = () => `local-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;

const statusConfig: Record<string, { label: string; color: string; bgColor: string }> = {
  draft: { label: 'Draft', color: 'text-gray-600', bgColor: 'bg-gray-100 dark:bg-gray-800' },
  pending_review: { label: 'Menunggu Review', color: 'text-amber-600', bgColor: 'bg-amber-100 dark:bg-amber-900/30' },
  published: { label: 'Published', color: 'text-green-600', bgColor: 'bg-green-100 dark:bg-green-900/30' },
  rejected: { label: 'Ditolak', color: 'text-red-600', bgColor: 'bg-red-100 dark:bg-red-900/30' },
  archived: { label: 'Diarsipkan', color: 'text-gray-500', bgColor: 'bg-gray-100 dark:bg-gray-800' },
};

interface PortfolioEditorProps {
  portfolio?: Portfolio;
  isEdit?: boolean;
}

export function PortfolioEditor({ portfolio, isEdit = false }: PortfolioEditorProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const titleInputRef = useRef<HTMLInputElement>(null);

  // Form state
  const [judul, setJudul] = useState(portfolio?.judul || '');
  const [selectedTags, setSelectedTags] = useState<Tag[]>(portfolio?.tags || []);
  const [selectedSeries, setSelectedSeries] = useState<Series | null>((portfolio?.series as Series | null) || null);
  const [pendingSeriesChange, setPendingSeriesChange] = useState<Series | null>(null);
  const [showSeriesConfirm, setShowSeriesConfirm] = useState(false);
  const [thumbnailFile, setThumbnailFile] = useState<File | null>(null);
  const [thumbnailPreview, setThumbnailPreview] = useState<string | null>(null);
  const [thumbnailUrl, setThumbnailUrl] = useState<string | null>(portfolio?.thumbnail_url || null);
  const [contentBlocks, setContentBlocks] = useState<LocalContentBlock[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitProgress, setSubmitProgress] = useState(0);
  const [submitMessage, setSubmitMessage] = useState('');
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [activeBlockId, setActiveBlockId] = useState<string | null>(null);
  const [showPreview, setShowPreview] = useState(false);

  // Initialize content blocks
  useEffect(() => {
    if (portfolio?.content_blocks) {
      setContentBlocks(
        portfolio.content_blocks.map((b) => ({
          ...b,
          payload: b.payload as unknown as Record<string, unknown>,
          isNew: false,
        }))
      );
    }
  }, [portfolio?.content_blocks]);

  // Auto-focus title on mount for new portfolios
  useEffect(() => {
    if (!isEdit && titleInputRef.current) {
      titleInputRef.current.focus();
    }
  }, [isEdit]);

  // DnD sensors
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  // Fetch tags
  const { data: tagsData } = useQuery({
    queryKey: ['tags'],
    queryFn: () => tagsApi.getTags(),
  });

  const tags = tagsData?.data || [];

  // Fetch series (active only)
  const { data: seriesData } = useQuery({
    queryKey: ['series'],
    queryFn: () => seriesApi.getSeries(),
  });

  // Merge active series with portfolio's existing series (including inactive ones)
  // This ensures inactive series that are already assigned to portfolio are still shown
  const seriesList = useMemo(() => {
    const activeSeries = seriesData?.data || [];
    const portfolioSeries = portfolio?.series;

    // Create a map of active series by ID - mark all as active since they come from active list
    const seriesMap = new Map(activeSeries.map((s) => [s.id, { ...s, is_active: true }]));

    // Add portfolio's series if not in active list (this means it's inactive)
    if (portfolioSeries && !seriesMap.has(portfolioSeries.id)) {
      seriesMap.set(portfolioSeries.id, { ...portfolioSeries, is_active: false } as Series);
    }

    return Array.from(seriesMap.values());
  }, [seriesData?.data, portfolio?.series]);

  const toggleTag = (tag: Tag) => {
    setSelectedTags((prev) =>
      prev.some((t) => t.id === tag.id) ? prev.filter((t) => t.id !== tag.id) : [...prev, tag]
    );
  };

  // Handle series selection - show confirmation if blocks exist
  const handleSeriesSelect = (s: Series | null) => {
    // If selecting the same series or deselecting, just update
    if (s?.id === selectedSeries?.id) {
      setSelectedSeries(null);
      return;
    }

    // If there are existing blocks and selecting a new series with blocks, show confirmation
    if (contentBlocks.length > 0 && s?.blocks && s.blocks.length > 0) {
      setPendingSeriesChange(s);
      setShowSeriesConfirm(true);
    } else if (s?.blocks && s.blocks.length > 0) {
      // No existing blocks, apply series template directly
      applySeriesTemplate(s);
    } else {
      // Series has no blocks, just select it
      setSelectedSeries(s);
    }
  };

  // Apply series template blocks
  const applySeriesTemplate = (s: Series) => {
    if (!s.blocks || s.blocks.length === 0) {
      setSelectedSeries(s);
      return;
    }

    // Replace content blocks with series template
    const templateBlocks: LocalContentBlock[] = s.blocks.map((block, index) => ({
      id: generateId(),
      block_type: block.block_type,
      block_order: index,
      payload: getDefaultPayload(block.block_type),
      isNew: true,
      seriesInstruksi: block.instruksi, // Store instruction for display
    }));

    setContentBlocks(templateBlocks);
    setSelectedSeries(s);
    toast.success(`Template "${s.nama}" diterapkan`);
  };

  // Confirm series change
  const confirmSeriesChange = () => {
    if (pendingSeriesChange) {
      applySeriesTemplate(pendingSeriesChange);
    }
    setPendingSeriesChange(null);
    setShowSeriesConfirm(false);
  };

  // Check if blocks are locked (series with template is selected)
  const isBlocksLocked = selectedSeries?.blocks && selectedSeries.blocks.length > 0;

  const handleThumbnailChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (!file.type.startsWith('image/')) {
        toast.error('File harus berupa gambar');
        return;
      }
      if (file.size > 5 * 1024 * 1024) {
        toast.error('Ukuran file maksimal 5MB');
        return;
      }
      setThumbnailFile(file);
      const reader = new FileReader();
      reader.onloadend = () => setThumbnailPreview(reader.result as string);
      reader.readAsDataURL(file);
    }
  };

  const removeThumbnail = () => {
    setThumbnailFile(null);
    setThumbnailPreview(null);
    setThumbnailUrl(null);
  };

  // Content Block handlers
  const addBlock = (type: ContentBlockType) => {
    const newBlock: LocalContentBlock = {
      id: generateId(),
      block_type: type,
      block_order: contentBlocks.length,
      payload: getDefaultPayload(type),
      isNew: true,
    };
    setContentBlocks([...contentBlocks, newBlock]);
    // Scroll to new block after a short delay
    setTimeout(() => {
      document.getElementById(`block-${newBlock.id}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }, 100);
  };

  const updateBlock = (id: string, payload: Record<string, unknown>) => {
    setContentBlocks((prev) => prev.map((b) => (b.id === id ? { ...b, payload } : b)));
  };

  const updateBlockFile = (id: string, file: File | null, preview: string) => {
    setContentBlocks((prev) =>
      prev.map((b) => (b.id === id ? { ...b, pendingFile: file || undefined, pendingPreview: preview || undefined } : b))
    );
  };

  const removeBlock = (id: string) => {
    setContentBlocks((prev) => prev.filter((b) => b.id !== id));
    toast.success('Block dihapus');
  };

  const duplicateBlock = (block: LocalContentBlock) => {
    const newBlock: LocalContentBlock = {
      ...block,
      id: generateId(),
      block_order: contentBlocks.length,
      isNew: true,
      pendingFile: undefined,
      pendingPreview: undefined,
    };
    setContentBlocks([...contentBlocks, newBlock]);
    toast.success('Block diduplikasi');
  };

  const handleDragStart = (event: DragStartEvent) => {
    setActiveBlockId(event.active.id as string);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveBlockId(null);
    const { active, over } = event;
    if (over && active.id !== over.id) {
      setContentBlocks((items) => {
        const oldIndex = items.findIndex((item) => item.id === active.id);
        const newIndex = items.findIndex((item) => item.id === over.id);
        const newItems = arrayMove(items, oldIndex, newIndex);
        return newItems.map((item, index) => ({ ...item, block_order: index }));
      });
    }
  };

  const getDefaultPayload = (type: ContentBlockType): Record<string, unknown> => {
    switch (type) {
      case 'text': return { content: '' };
      case 'image': return { url: '', caption: '' };
      case 'youtube': return { video_id: '', title: '' };
      case 'button': return { text: '', url: '' };
      // New block types
      case 'figma': return { url: '', title: '' };
      case 'canva': return { url: '', title: '' };
      case 'ppt': return { source: 'google_slides', url: '', title: '', file_name: '' };
      case 'pdf': return { url: '', title: '', file_name: '' };
      case 'doc': return { url: '', title: '', file_name: '' };
      default: return {};
    }
  };

  // Save handler
  const handleSave = async (submitForReview: boolean = false) => {
    if (!judul.trim()) {
      toast.error('Judul wajib diisi');
      titleInputRef.current?.focus();
      return;
    }

    setIsSubmitting(true);
    setSubmitProgress(0);

    try {
      let portfolioId = portfolio?.id;
      const totalSteps = 4 + contentBlocks.length;
      let currentStep = 0;

      const updateProgress = (message: string) => {
        currentStep++;
        setSubmitProgress((currentStep / totalSteps) * 100);
        setSubmitMessage(message);
      };

      if (isEdit && portfolioId) {
        // Update existing portfolio
        updateProgress('Menyimpan informasi...');
        await portfoliosApi.updatePortfolio(portfolioId, {
          judul: judul.trim(),
          tag_ids: selectedTags.map((t) => t.id),
          series_id: selectedSeries?.id,
        });
      } else {
        // Create new portfolio
        updateProgress('Membuat portfolio...');
        const createRes = await portfoliosApi.createPortfolio({
          judul: judul.trim(),
          tag_ids: selectedTags.map((t) => t.id),
          series_id: selectedSeries?.id,
        });
        if (!createRes.data) throw new Error('Gagal membuat portfolio');
        portfolioId = createRes.data.id;
      }

      // Upload thumbnail
      if (thumbnailFile) {
        updateProgress('Mengupload thumbnail...');
        const thumbUrl = await uploadsApi.uploadFile(thumbnailFile, 'thumbnail', portfolioId);
        await portfoliosApi.updatePortfolio(portfolioId, { thumbnail_url: thumbUrl });
      } else if (isEdit && thumbnailUrl !== portfolio?.thumbnail_url) {
        await portfoliosApi.updatePortfolio(portfolioId, { thumbnail_url: thumbnailUrl || undefined });
      }

      // Delete removed blocks (edit mode only)
      if (isEdit && portfolio?.content_blocks) {
        const existingIds = portfolio.content_blocks.map((b) => b.id);
        const currentIds = contentBlocks.filter((b) => !b.isNew).map((b) => b.id);
        const deletedIds = existingIds.filter((id) => !currentIds.includes(id));
        for (const blockId of deletedIds) {
          await portfoliosApi.deleteBlock(portfolioId, blockId);
        }
      }

      // Process content blocks
      for (let i = 0; i < contentBlocks.length; i++) {
        const block = contentBlocks[i];
        let blockPayload = { ...block.payload };

        // Handle file uploads for image, pdf, and doc blocks
        if (block.pendingFile) {
          if (block.block_type === 'image') {
            updateProgress(`Mengupload gambar ${i + 1}...`);
            const imageUrl = await uploadsApi.uploadFile(block.pendingFile, 'portfolio_image', portfolioId);
            blockPayload = { ...blockPayload, url: imageUrl };
          } else if (block.block_type === 'pdf' || block.block_type === 'doc') {
            updateProgress(`Mengupload dokumen ${i + 1}...`);
            const docUrl = await uploadsApi.uploadFile(block.pendingFile, 'document', portfolioId);
            blockPayload = { ...blockPayload, url: docUrl };
          }
        } else {
          updateProgress(`Menyimpan konten ${i + 1}...`);
        }

        if (block.isNew) {
          await portfoliosApi.addBlock(portfolioId, {
            block_type: block.block_type,
            block_order: i,
            payload: blockPayload as unknown as ContentBlockPayload,
          });
        } else {
          await portfoliosApi.updateBlock(portfolioId, block.id, { payload: blockPayload as unknown as ContentBlockPayload });
        }
      }

      // Reorder blocks
      const existingBlocks = contentBlocks.filter((b) => !b.isNew && !b.id.startsWith('local-'));
      if (existingBlocks.length > 0) {
        await portfoliosApi.reorderBlocks(
          portfolioId,
          existingBlocks.map((b, i) => ({ id: b.id, order: i }))
        );
      }

      // Submit for review
      if (submitForReview) {
        updateProgress('Mengirim untuk review...');
        await portfoliosApi.submitPortfolio(portfolioId);
        toast.success('Portfolio berhasil dikirim untuk review! 🎉');
      } else {
        toast.success(isEdit ? 'Perubahan tersimpan!' : 'Draft tersimpan!');
      }

      queryClient.invalidateQueries({ queryKey: ['portfolio'] });
      queryClient.invalidateQueries({ queryKey: ['my-portfolios'] });

      // Always redirect to profile after save/submit
      router.push(`/${user?.username}`);
    } catch (error) {
      console.error(error);
      toast.error('Gagal menyimpan portfolio');
    } finally {
      setIsSubmitting(false);
      setSubmitProgress(0);
      setSubmitMessage('');
    }
  };

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => portfoliosApi.deletePortfolio(portfolio!.id),
    onSuccess: () => {
      toast.success('Portfolio berhasil dihapus');
      router.push(`/${user?.username}`);
    },
    onError: () => toast.error('Gagal menghapus portfolio'),
  });

  // Archive mutation
  const archiveMutation = useMutation({
    mutationFn: () =>
      portfolio?.status === 'archived'
        ? portfoliosApi.unarchivePortfolio(portfolio.id)
        : portfoliosApi.archivePortfolio(portfolio!.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portfolio'] });
      queryClient.invalidateQueries({ queryKey: ['my-portfolios'] });
      toast.success(portfolio?.status === 'archived' ? 'Portfolio diaktifkan kembali' : 'Portfolio diarsipkan');
      // Redirect to profile after archive/unarchive
      router.push(`/${user?.username}`);
    },
    onError: () => toast.error('Gagal mengubah status'),
  });

  const displayThumbnail = thumbnailPreview || thumbnailUrl;
  // Show submit button: always for new portfolio, or for edit if status is draft/rejected/published
  // Published portfolios can be edited and re-submitted for moderation
  const canSubmitForReview =
    !isEdit ||
    portfolio?.status === 'draft' ||
    portfolio?.status === 'rejected' ||
    portfolio?.status === 'published';
  const canDelete = isEdit && portfolio?.status === 'draft';
  const canArchive = isEdit && (portfolio?.status === 'published' || portfolio?.status === 'archived');
  const status = portfolio?.status ? statusConfig[portfolio.status] : null;
  const activeBlock = contentBlocks.find((b) => b.id === activeBlockId);

  return (
    <TooltipProvider>
      <div className="relative min-h-screen pb-32">
        {/* Progress Overlay */}
        <AnimatePresence>
          {isSubmitting && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
            >
              <motion.div
                initial={{ scale: 0.9, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                className="w-full max-w-sm rounded-2xl border bg-card p-8 shadow-2xl"
              >
                <div className="flex flex-col items-center text-center">
                  <div className="relative mb-4">
                    <div className="h-16 w-16 rounded-full bg-primary/10" />
                    <Loader2 className="absolute inset-0 m-auto h-8 w-8 animate-spin text-primary" />
                  </div>
                  <h3 className="text-lg font-semibold">Menyimpan Portfolio</h3>
                  <p className="mt-1 text-sm text-muted-foreground">{submitMessage}</p>
                  <Progress value={submitProgress} className="mt-4 h-2 w-full" />
                  <p className="mt-2 text-xs text-muted-foreground">{Math.round(submitProgress)}%</p>
                </div>
              </motion.div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Admin Review Note */}
        {portfolio?.admin_review_note && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className="mb-6 flex gap-3 rounded-xl border border-amber-500/30 bg-gradient-to-r from-amber-500/10 to-orange-500/10 p-4"
          >
            <AlertCircle className="h-5 w-5 shrink-0 text-amber-500" />
            <div>
              <p className="text-sm font-medium text-amber-700 dark:text-amber-400">Catatan dari Admin</p>
              <p className="mt-1 text-sm text-amber-600 dark:text-amber-300">{portfolio.admin_review_note}</p>
            </div>
          </motion.div>
        )}

        {/* Hero Section - Thumbnail */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="relative mb-8 overflow-hidden rounded-2xl border bg-card shadow-sm"
        >
          {displayThumbnail ? (
            <div className="group relative aspect-[21/9] w-full overflow-hidden">
              <Image src={displayThumbnail} alt="Thumbnail" fill className="object-cover transition-transform duration-500 group-hover:scale-105" />
              <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent" />
              <div className="absolute inset-0 flex items-center justify-center gap-3 bg-black/40 opacity-0 transition-all duration-300 group-hover:opacity-100">
                <label className="cursor-pointer">
                  <Button variant="secondary" size="sm" className="shadow-lg" asChild>
                    <span><Camera className="mr-2 h-4 w-4" />Ganti Thumbnail</span>
                  </Button>
                  <input type="file" accept="image/*" className="hidden" onChange={handleThumbnailChange} />
                </label>
                <Button variant="destructive" size="sm" className="shadow-lg" onClick={removeThumbnail}>
                  <Trash2 className="mr-2 h-4 w-4" />Hapus
                </Button>
              </div>
              {/* Status Badge */}
              {status && (
                <div className={cn('absolute right-4 top-4 rounded-full px-3 py-1 text-xs font-medium shadow-lg', status.bgColor, status.color)}>
                  {status.label}
                </div>
              )}
            </div>
          ) : (
            <label className="group flex aspect-[21/9] w-full cursor-pointer flex-col items-center justify-center bg-gradient-to-br from-muted/50 to-muted transition-colors hover:from-muted hover:to-muted/80">
              <div className="flex flex-col items-center p-4 text-center">
                <div className="rounded-full bg-primary/10 p-3 transition-transform group-hover:scale-110 md:p-4">
                  <ImageIcon className="h-6 w-6 text-primary md:h-8 md:w-8" />
                </div>
                <span className="mt-3 text-sm font-medium md:mt-4">Tambahkan Thumbnail</span>
                <span className="mt-1 text-xs text-muted-foreground">Gambar utama menarik</span>
                <span className="mt-1 text-[10px] text-muted-foreground md:mt-2 md:text-xs">PNG, JPG, WebP • Max 5MB</span>
              </div>
              <input type="file" accept="image/*" className="hidden" onChange={handleThumbnailChange} />
            </label>
          )}
        </motion.div>

        {/* Title Input */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="mb-6"
        >
          <Label htmlFor="judul" className="mb-2 block text-sm font-medium text-muted-foreground">
            Judul Portfolio
          </Label>
          <Input
            id="judul"
            ref={titleInputRef}
            value={judul}
            onChange={(e) => setJudul(e.target.value)}
            placeholder="Masukkan judul portfolio..."
            className="h-12 rounded-xl border-2 bg-muted/30 px-4 text-lg font-semibold transition-colors focus:border-primary focus:bg-background sm:h-14 sm:text-2xl"
          />
        </motion.div>

        {/* Tags */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.15 }}
          className="mb-8"
        >
          <Label className="mb-2 block text-sm font-medium text-muted-foreground">
            Tags <span className="font-normal">(pilih kategori yang sesuai)</span>
          </Label>
          <div className="flex flex-wrap items-center gap-2">
            {tags.map((tag) => {
              const isSelected = selectedTags.some((t) => t.id === tag.id);
              return (
                <motion.button
                  key={tag.id}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  onClick={() => toggleTag(tag)}
                  className={cn(
                    'rounded-full px-3 py-1.5 text-sm font-medium transition-all',
                    isSelected
                      ? 'bg-primary text-primary-foreground shadow-md'
                      : 'bg-muted text-muted-foreground hover:bg-muted/80'
                  )}
                >
                  {tag.nama}
                  {isSelected && <Check className="ml-1.5 inline h-3 w-3" />}
                </motion.button>
              );
            })}
            {tags.length === 0 && (
              <span className="text-sm text-muted-foreground">Tidak ada tags tersedia</span>
            )}
          </div>
        </motion.div>

        {/* Series */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.17 }}
          className="mb-8"
        >
          <Label className="mb-2 block text-sm font-medium text-muted-foreground">
            Series Template <span className="font-normal">(opsional - pilih untuk menggunakan template)</span>
          </Label>
          <div className="flex flex-wrap items-center gap-2">
            {seriesList.map((s) => {
              const isSelected = selectedSeries?.id === s.id;
              const isInactive = s.is_active === false; // Explicitly check for false
              const hasBlocks = s.blocks && s.blocks.length > 0;
              // Inactive series can only be deselected, not selected
              const canSelect = !isInactive || isSelected;
              return (
                <motion.button
                  key={s.id}
                  whileHover={canSelect ? { scale: 1.05 } : undefined}
                  whileTap={canSelect ? { scale: 0.95 } : undefined}
                  onClick={() => canSelect && handleSeriesSelect(isSelected ? null : s)}
                  disabled={!canSelect}
                  className={cn(
                    'rounded-full px-3 py-1.5 text-sm font-medium transition-all',
                    isSelected
                      ? isInactive
                        ? 'bg-gray-400 text-white shadow-md'
                        : 'bg-blue-500 text-white shadow-md'
                      : isInactive
                        ? 'bg-muted/30 text-muted-foreground/50 cursor-not-allowed'
                        : 'bg-muted text-muted-foreground hover:bg-muted/80'
                  )}
                  title={isInactive ? 'Series ini sudah tidak aktif - tidak dapat dipilih' : hasBlocks ? `${s.blocks?.length} blocks template` : undefined}
                >
                  {s.nama}
                  {hasBlocks && <span className="ml-1 text-xs opacity-70">({s.blocks?.length})</span>}
                  {isInactive && <span className="ml-1 text-xs opacity-70">(nonaktif)</span>}
                  {isSelected && <Check className="ml-1.5 inline h-3 w-3" />}
                </motion.button>
              );
            })}
            {seriesList.length === 0 && (
              <span className="text-sm text-muted-foreground">Tidak ada series tersedia</span>
            )}
          </div>
          {selectedSeries && isBlocksLocked && (
            <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">
              ⚠️ Menggunakan template series - urutan dan tipe block tidak dapat diubah
            </p>
          )}
        </motion.div>

        {/* Series Change Confirmation Dialog */}
        <ConfirmDialog
          open={showSeriesConfirm}
          onOpenChange={setShowSeriesConfirm}
          title="Ganti Series Template?"
          description={
            <>
              Anda akan mengganti ke series <strong>&quot;{pendingSeriesChange?.nama}&quot;</strong>.
              <span className="mt-2 block text-amber-600">
                ⚠️ Semua block konten yang sudah ada ({contentBlocks.length} blocks) akan DIHAPUS dan diganti dengan template baru ({pendingSeriesChange?.blocks?.length || 0} blocks).
              </span>
            </>
          }
          confirmText="Ya, Ganti Series"
          variant="destructive"
          onConfirm={confirmSeriesChange}
        />

        {/* Content Blocks */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="space-y-4"
        >
          {/* Section Header */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold">Konten</h2>
              <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                {contentBlocks.length} block
              </span>
              {isBlocksLocked && (
                <Badge variant="outline" className="text-amber-600 border-amber-300">
                  Template Terkunci
                </Badge>
              )}
            </div>
          </div>

          {/* Empty State */}
          {contentBlocks.length === 0 && !isBlocksLocked && (
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="rounded-2xl border-2 border-dashed bg-muted/30 p-12 text-center"
            >
              <div className="mx-auto w-fit rounded-full bg-primary/10 p-4">
                <FileText className="h-8 w-8 text-primary" />
              </div>
              <h3 className="mt-4 font-semibold">Mulai Berkreasi</h3>
              <p className="mt-1 text-sm text-muted-foreground">
                Tambahkan teks, gambar, video, atau link untuk menceritakan karyamu
              </p>
              <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
                {blockTypeOptions.map((opt) => (
                  <Button
                    key={opt.value}
                    variant="outline"
                    className="flex h-auto flex-col items-center justify-center gap-2 rounded-xl border-dashed p-4 hover:border-solid hover:bg-muted/50"
                    onClick={() => addBlock(opt.value as ContentBlockType)}
                  >
                    <div className={cn('rounded-lg p-2.5', opt.bgColor)}>
                      <opt.icon className={cn('h-6 w-6', opt.iconColor)} />
                    </div>
                    <span className="text-xs font-medium">{opt.label}</span>
                  </Button>
                ))}
              </div>
            </motion.div>
          )}

          {/* Content Blocks with DnD */}
          {contentBlocks.length > 0 && (
            <DndContext
              sensors={sensors}
              collisionDetection={closestCenter}
              onDragStart={handleDragStart}
              onDragEnd={handleDragEnd}
            >
              <SortableContext items={contentBlocks.map((b) => b.id)} strategy={verticalListSortingStrategy}>
                <div className="space-y-3">
                  <AnimatePresence mode="popLayout">
                    {contentBlocks.map((block, index) => (
                      <ContentBlockEditor
                        key={block.id}
                        block={block}
                        index={index}
                        isLocked={!!isBlocksLocked}
                        onUpdate={(payload) => updateBlock(block.id, payload)}
                        onUpdateFile={(file, preview) => updateBlockFile(block.id, file, preview)}
                        onRemove={() => removeBlock(block.id)}
                        onDuplicate={() => duplicateBlock(block)}
                      />
                    ))}
                  </AnimatePresence>
                </div>
              </SortableContext>

              {/* Drag Overlay */}
              <DragOverlay>
                {activeBlock && (
                  <div className="rounded-xl border bg-card p-4 opacity-90 shadow-2xl">
                    <div className="flex items-center gap-2">
                      <GripVertical className="h-4 w-4 text-muted-foreground" />
                      <span className="text-sm font-medium capitalize">{activeBlock.block_type}</span>
                    </div>
                  </div>
                )}
              </DragOverlay>
            </DndContext>
          )}

          {/* Add Block Button - hidden when using series template */}
          {contentBlocks.length > 0 && !isBlocksLocked && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="flex justify-center pt-4"
            >
              <Popover>
                <PopoverTrigger asChild>
                  <Button variant="outline" className="gap-2 rounded-full">
                    <Plus className="h-4 w-4" />
                    Tambah Konten
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-[90vw] max-w-md p-2 sm:w-[500px]" align="center">
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                    {blockTypeOptions.map((opt) => (
                      <Button
                        key={opt.value}
                        variant="ghost"
                        className="flex h-auto flex-col items-center justify-center gap-2 rounded-xl p-3 hover:bg-muted/50"
                        onClick={() => addBlock(opt.value as ContentBlockType)}
                      >
                        <div className={cn('rounded-lg p-2', opt.bgColor)}>
                          <opt.icon className={cn('h-5 w-5', opt.iconColor)} />
                        </div>
                        <div className="flex flex-col items-center gap-0.5 text-center">
                          <span className="text-xs font-medium">{opt.label}</span>
                          <span className="text-[10px] text-muted-foreground line-clamp-1">
                            {opt.description}
                          </span>
                        </div>
                      </Button>
                    ))}
                  </div>
                </PopoverContent>
              </Popover>
            </motion.div>
          )}
        </motion.div>

        {/* Floating Action Bar - centered relative to content area (accounting for 64px sidebar) */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="fixed bottom-20 left-1/2 z-[90] -translate-x-1/2 md:bottom-6 md:left-[calc(50%+32px)]"
        >
          <div className="flex items-center gap-1.5 rounded-full border bg-card/95 p-1.5 shadow-2xl backdrop-blur-lg sm:gap-2 sm:p-2">
            {/* Delete/Archive Actions */}
            {(canDelete || canArchive) && (
              <>
                {canDelete && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-10 w-10 rounded-full text-destructive hover:bg-destructive/10 hover:text-destructive"
                        onClick={() => setDeleteDialogOpen(true)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Hapus Portfolio</TooltipContent>
                  </Tooltip>
                )}
                {canArchive && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-10 w-10 rounded-full"
                        onClick={() => archiveMutation.mutate()}
                        disabled={archiveMutation.isPending}
                      >
                        {archiveMutation.isPending ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : portfolio?.status === 'archived' ? (
                          <ArchiveRestore className="h-4 w-4" />
                        ) : (
                          <Archive className="h-4 w-4" />
                        )}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      {portfolio?.status === 'archived' ? 'Aktifkan Kembali' : 'Arsipkan'}
                    </TooltipContent>
                  </Tooltip>
                )}
                <div className="mx-1 h-6 w-px bg-border" />
              </>
            )}

            {/* Preview Button */}
            {isEdit && portfolio && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-10 w-10 rounded-full"
                    onClick={() => window.open(`/${user?.username}/${portfolio.slug}`, '_blank')}
                  >
                    <Eye className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Lihat Portfolio</TooltipContent>
              </Tooltip>
            )}

            {/* Save Button */}
            <Button
              variant="outline"
              className="h-10 gap-2 rounded-full px-4"
              onClick={() => handleSave(false)}
              disabled={isSubmitting}
            >
              <Save className="h-4 w-4" />
              <span className="hidden sm:inline">Simpan Sebagai Draft</span>
            </Button>

            {/* Submit Button */}
            {canSubmitForReview && (
              <Button
                className="h-10 gap-2 rounded-full px-4 shadow-lg"
                onClick={() => handleSave(true)}
                disabled={isSubmitting}
              >
                <Send className="h-4 w-4" />
                <span className="hidden sm:inline">Kirim Review</span>
              </Button>
            )}
          </div>
        </motion.div>

        {/* Delete Confirmation */}
        <ConfirmDialog
          open={deleteDialogOpen}
          onOpenChange={setDeleteDialogOpen}
          title="Hapus Portfolio"
          description={<>Yakin ingin menghapus <strong>{portfolio?.judul}</strong>? Tindakan ini tidak dapat dibatalkan.</>}
          confirmText="Hapus"
          variant="destructive"
          isLoading={deleteMutation.isPending}
          onConfirm={() => deleteMutation.mutate()}
        />
      </div>
    </TooltipProvider>
  );
}


// Content Block Editor Component
function ContentBlockEditor({
  block,
  index,
  isLocked,
  onUpdate,
  onUpdateFile,
  onRemove,
  onDuplicate,
}: {
  block: LocalContentBlock;
  index: number;
  isLocked: boolean;
  onUpdate: (payload: Record<string, unknown>) => void;
  onUpdateFile: (file: File | null, preview: string) => void;
  onRemove: () => void;
  onDuplicate: () => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: block.id,
    disabled: isLocked, // Disable drag when locked
  });
  const [isExpanded, setIsExpanded] = useState(true);

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const blockTypeInfo = blockTypeOptions.find((o) => o.value === block.block_type);
  const Icon = blockTypeInfo?.icon || FileText;

  const handleImageFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (!file.type.startsWith('image/')) {
        toast.error('File harus berupa gambar');
        return;
      }
      if (file.size > 10 * 1024 * 1024) {
        toast.error('Ukuran file maksimal 10MB');
        return;
      }
      const reader = new FileReader();
      reader.onloadend = () => onUpdateFile(file, reader.result as string);
      reader.readAsDataURL(file);
    }
  };

  const displayImageUrl = block.pendingPreview || (block.payload.url as string) || '';

  return (
    <motion.div
      id={`block-${block.id}`}
      ref={setNodeRef}
      style={style}
      layout
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: isDragging ? 0.5 : 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.95 }}
      drag={!isLocked ? "x" : false}
      dragConstraints={{ left: 0, right: 0 }}
      dragElastic={{ left: 0.5, right: 0 }}
      onDragEnd={(e, info) => {
        if (!isLocked && info.offset.x < -100) {
          onRemove();
        }
      }}
      className={cn(
        'group rounded-xl border bg-card transition-shadow touch-pan-y',
        isDragging && 'shadow-2xl ring-2 ring-primary'
      )}
    >
      {/* Block Header */}
      <div className="flex items-center gap-2 border-b px-3 py-2">
        {!isLocked && (
          <button
            type="button"
            className="cursor-grab touch-none rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground active:cursor-grabbing"
            {...attributes}
            {...listeners}
          >
            <GripVertical className="h-4 w-4" />
          </button>
        )}

        <div className={cn('rounded-md p-1.5', blockTypeInfo?.bgColor)}>
          <Icon className={cn('h-3.5 w-3.5', blockTypeInfo?.iconColor)} />
        </div>

        <span className="text-sm font-medium">{blockTypeInfo?.label}</span>
        <span className="text-xs text-muted-foreground">#{index + 1}</span>

        <div className="ml-auto flex items-center gap-1">
          {!isLocked && (
            <>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onDuplicate}>
                    <Plus className="h-3.5 w-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Duplikasi</TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-destructive hover:bg-destructive/10 hover:text-destructive"
                    onClick={onRemove}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Hapus</TooltipContent>
              </Tooltip>
            </>
          )}

          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setIsExpanded(!isExpanded)}>
            <ChevronDown className={cn('h-3.5 w-3.5 transition-transform', !isExpanded && '-rotate-90')} />
          </Button>
        </div>
      </div>

      {/* Block Content */}
      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            className="overflow-hidden"
          >
            <div className="p-4">
              {/* Series Instruction */}
              {block.seriesInstruksi && (
                <div className="mb-4 flex items-start gap-2 rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30">
                  <AlertCircle className="h-4 w-4 shrink-0 text-blue-500 mt-0.5" />
                  <p className="text-sm text-blue-700 dark:text-blue-300">{block.seriesInstruksi}</p>
                </div>
              )}

              {block.block_type === 'text' && (
                <Textarea
                  value={(block.payload.content as string) || ''}
                  onChange={(e) => onUpdate({ content: e.target.value })}
                  placeholder="Tulis sesuatu yang menarik..."
                  rows={4}
                  className="resize-none border-0 bg-transparent p-0 text-base focus-visible:ring-0"
                />
              )}

              {block.block_type === 'image' && (
                <div className="space-y-4">
                  {displayImageUrl ? (
                    <div className="group/img relative aspect-video w-full overflow-hidden rounded-lg bg-muted">
                      <Image src={displayImageUrl} alt="Preview" fill className="object-contain" />
                      <div className="absolute inset-0 flex items-center justify-center gap-2 bg-black/50 opacity-0 transition-opacity group-hover/img:opacity-100">
                        <label className="cursor-pointer">
                          <Button variant="secondary" size="sm" asChild>
                            <span><Camera className="mr-1.5 h-4 w-4" />Ganti</span>
                          </Button>
                          <input type="file" accept="image/*" className="hidden" onChange={handleImageFileChange} />
                        </label>
                        <Button variant="destructive" size="sm" onClick={() => { onUpdate({ ...block.payload, url: '' }); onUpdateFile(null, ''); }}>
                          <Trash2 className="mr-1.5 h-4 w-4" />Hapus
                        </Button>
                      </div>
                      {block.pendingFile && (
                        <div className="absolute bottom-2 left-2 rounded-full bg-amber-500 px-2 py-1 text-xs text-white">
                          Akan diupload
                        </div>
                      )}
                    </div>
                  ) : (
                    <label className="flex aspect-video w-full cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed bg-muted/30 transition-colors hover:bg-muted/50">
                      <Upload className="h-8 w-8 text-muted-foreground" />
                      <span className="mt-2 text-sm font-medium">Upload Gambar</span>
                      <span className="text-xs text-muted-foreground">PNG, JPG, GIF • Max 10MB</span>
                      <input type="file" accept="image/*" className="hidden" onChange={handleImageFileChange} />
                    </label>
                  )}
                  <div className="space-y-2">
                    <Input
                      value={(block.payload.url as string) || ''}
                      onChange={(e) => onUpdate({ ...block.payload, url: e.target.value })}
                      placeholder="Atau masukkan URL gambar..."
                      className="text-sm"
                      disabled={!!block.pendingFile}
                    />
                    <Input
                      value={(block.payload.caption as string) || ''}
                      onChange={(e) => onUpdate({ ...block.payload, caption: e.target.value })}
                      placeholder="Caption (opsional)"
                      className="text-sm"
                    />
                  </div>
                </div>
              )}

              {block.block_type === 'youtube' && (() => {
                const extractVideoId = (input: string): string => {
                  if (!input) return '';
                  if (/^[a-zA-Z0-9_-]{11}$/.test(input)) return input;
                  const patterns = [
                    /(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([a-zA-Z0-9_-]{11})/,
                    /youtube\.com\/shorts\/([a-zA-Z0-9_-]{11})/,
                  ];
                  for (const pattern of patterns) {
                    const match = input.match(pattern);
                    if (match) return match[1];
                  }
                  return input;
                };

                const videoId = (block.payload.video_id as string) || '';

                return (
                  <div className="space-y-4">
                    <Input
                      value={videoId}
                      onChange={(e) => onUpdate({ ...block.payload, video_id: extractVideoId(e.target.value.trim()) })}
                      placeholder="Paste link YouTube atau video ID..."
                      className="text-sm"
                    />
                    {videoId && (
                      <div className="aspect-video w-full overflow-hidden rounded-lg">
                        <iframe
                          src={`https://www.youtube.com/embed/${videoId}`}
                          className="h-full w-full"
                          allowFullScreen
                        />
                      </div>
                    )}
                    <Input
                      value={(block.payload.title as string) || ''}
                      onChange={(e) => onUpdate({ ...block.payload, title: e.target.value })}
                      placeholder="Judul video (opsional)"
                      className="text-sm"
                    />
                  </div>
                );
              })()}

              {block.block_type === 'button' && (
                <div className="space-y-4">
                  <div className="grid gap-4 sm:grid-cols-2">
                    <Input
                      value={(block.payload.text as string) || ''}
                      onChange={(e) => onUpdate({ ...block.payload, text: e.target.value })}
                      placeholder="Teks tombol"
                    />
                    <Input
                      value={(block.payload.url as string) || ''}
                      onChange={(e) => onUpdate({ ...block.payload, url: e.target.value })}
                      placeholder="https://..."
                    />
                  </div>
                  {(block.payload.text as string) && (block.payload.url as string) && (
                    <div className="flex justify-center rounded-lg bg-muted/50 p-4">
                      <Button variant="default" asChild>
                        <a href={block.payload.url as string} target="_blank" rel="noopener noreferrer">
                          {String(block.payload.text)}
                        </a>
                      </Button>
                    </div>
                  )}
                </div>
              )}

              {block.block_type === 'table' && (() => {
                const headers = (block.payload.headers as string[]) || ['Kolom 1', 'Kolom 2'];
                const rows = (block.payload.rows as string[][]) || [['', '']];

                const updateHeaders = (newHeaders: string[]) => {
                  onUpdate({ ...block.payload, headers: newHeaders });
                };

                const updateRows = (newRows: string[][]) => {
                  onUpdate({ ...block.payload, rows: newRows });
                };

                const addColumn = () => {
                  const newHeaders = [...headers, `Kolom ${headers.length + 1}`];
                  const newRows = rows.map(row => [...row, '']);
                  onUpdate({ headers: newHeaders, rows: newRows });
                };

                const removeColumn = (colIndex: number) => {
                  if (headers.length <= 1) return;
                  const newHeaders = headers.filter((_, i) => i !== colIndex);
                  const newRows = rows.map(row => row.filter((_, i) => i !== colIndex));
                  onUpdate({ headers: newHeaders, rows: newRows });
                };

                const addRow = () => {
                  const newRow = headers.map(() => '');
                  updateRows([...rows, newRow]);
                };

                const removeRow = (rowIndex: number) => {
                  if (rows.length <= 1) return;
                  updateRows(rows.filter((_, i) => i !== rowIndex));
                };

                const updateCell = (rowIndex: number, colIndex: number, value: string) => {
                  const newRows = rows.map((row, ri) =>
                    ri === rowIndex ? row.map((cell, ci) => (ci === colIndex ? value : cell)) : row
                  );
                  updateRows(newRows);
                };

                const updateHeader = (colIndex: number, value: string) => {
                  const newHeaders = headers.map((h, i) => (i === colIndex ? value : h));
                  updateHeaders(newHeaders);
                };

                return (
                  <div className="space-y-4">
                    <div className="flex gap-2">
                      <Button type="button" variant="outline" size="sm" onClick={addColumn}>
                        <Plus className="h-4 w-4 mr-1" />
                        Kolom
                      </Button>
                      <Button type="button" variant="outline" size="sm" onClick={addRow}>
                        <Plus className="h-4 w-4 mr-1" />
                        Baris
                      </Button>
                    </div>
                    <div className="overflow-x-auto rounded-lg border">
                      <table className="w-full text-sm">
                        <thead className="bg-muted/50">
                          <tr>
                            {headers.map((header, colIndex) => (
                              <th key={colIndex} className="border-b p-2">
                                <div className="flex items-center gap-1">
                                  <Input
                                    value={header}
                                    onChange={(e) => updateHeader(colIndex, e.target.value)}
                                    className="h-8 text-sm font-medium"
                                    placeholder={`Kolom ${colIndex + 1}`}
                                  />
                                  {headers.length > 1 && (
                                    <Button
                                      type="button"
                                      variant="ghost"
                                      size="icon"
                                      className="h-6 w-6 shrink-0"
                                      onClick={() => removeColumn(colIndex)}
                                    >
                                      <X className="h-3 w-3" />
                                    </Button>
                                  )}
                                </div>
                              </th>
                            ))}
                            <th className="w-10 border-b p-2"></th>
                          </tr>
                        </thead>
                        <tbody>
                          {rows.map((row, rowIndex) => (
                            <tr key={rowIndex}>
                              {row.map((cell, colIndex) => (
                                <td key={colIndex} className="border-b p-2">
                                  <Input
                                    value={cell}
                                    onChange={(e) => updateCell(rowIndex, colIndex, e.target.value)}
                                    className="h-8 text-sm"
                                    placeholder="..."
                                  />
                                </td>
                              ))}
                              <td className="border-b p-2">
                                {rows.length > 1 && (
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    className="h-6 w-6"
                                    onClick={() => removeRow(rowIndex)}
                                  >
                                    <Trash2 className="h-3 w-3" />
                                  </Button>
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                );
              })()}

              {block.block_type === 'embed' && (
                <div className="space-y-4">
                  <Textarea
                    value={(block.payload.html as string) || ''}
                    onChange={(e) => onUpdate({ ...block.payload, html: e.target.value })}
                    placeholder="Paste kode embed HTML/iframe di sini..."
                    rows={4}
                    className="font-mono text-sm"
                  />
                  <Input
                    value={(block.payload.title as string) || ''}
                    onChange={(e) => onUpdate({ ...block.payload, title: e.target.value })}
                    placeholder="Judul embed (opsional)"
                    className="text-sm"
                  />
                  {(block.payload.html as string) && (
                    <div className="rounded-lg border bg-muted/30 p-4">
                      <p className="text-xs text-muted-foreground mb-2">Preview:</p>
                      <div
                        className="overflow-hidden rounded"
                        dangerouslySetInnerHTML={{ __html: String(block.payload.html) }}
                      />
                    </div>
                  )}
                </div>
              )}

              {/* ===== FIGMA BLOCK ===== */}
              {block.block_type === 'figma' && (
                <div className="space-y-4">
                  <Input
                    value={(block.payload.url as string) || ''}
                    onChange={(e) => onUpdate({ url: e.target.value.trim() })}
                    placeholder="https://www.figma.com/file/..."
                    className="text-sm"
                  />
                  {Boolean(block.payload.url) && isValidFigmaUrl(block.payload.url as string) && (
                    <div className="aspect-video w-full overflow-hidden rounded-lg border shadow-sm">
                      <iframe
                        src={getFigmaEmbedUrl(block.payload.url as string) || ''}
                        className="h-full w-full"
                        allowFullScreen
                        loading="lazy"
                      />
                    </div>
                  )}
                  {Boolean(block.payload.url) && !isValidFigmaUrl(block.payload.url as string) && (
                    <p className="text-sm text-amber-600">⚠️ URL Figma tidak valid</p>
                  )}
                  <Input
                    value={(block.payload.title as string) || ''}
                    onChange={(e) => onUpdate({ ...block.payload, title: e.target.value })}
                    placeholder="Judul (opsional)"
                    className="text-sm"
                  />
                </div>
              )}

              {/* ===== CANVA BLOCK ===== */}
              {block.block_type === 'canva' && (
                <div className="space-y-4">
                  <Input
                    value={(block.payload.url as string) || ''}
                    onChange={(e) => onUpdate({ url: e.target.value.trim() })}
                    placeholder="https://www.canva.com/design/..."
                    className="text-sm"
                  />
                  {Boolean(block.payload.url) && isValidCanvaUrl(block.payload.url as string) && (
                    <div className="aspect-video w-full overflow-hidden rounded-lg border shadow-sm">
                      <iframe
                        src={getCanvaEmbedUrl(block.payload.url as string) || ''}
                        className="h-full w-full"
                        allowFullScreen
                        loading="lazy"
                      />
                    </div>
                  )}
                  {Boolean(block.payload.url) && !isValidCanvaUrl(block.payload.url as string) && (
                    <p className="text-sm text-amber-600">⚠️ URL Canva tidak valid</p>
                  )}
                  <Input
                    value={(block.payload.title as string) || ''}
                    onChange={(e) => onUpdate({ ...block.payload, title: e.target.value })}
                    placeholder="Judul (opsional)"
                    className="text-sm"
                  />
                </div>
              )}

              {/* ===== PPT BLOCK (Google Slides) ===== */}
              {block.block_type === 'ppt' && (
                <div className="space-y-4">
                  <Input
                    value={(block.payload.url as string) || ''}
                    onChange={(e) => onUpdate({ ...block.payload, source: 'google_slides', url: e.target.value.trim() })}
                    placeholder="https://docs.google.com/presentation/d/..."
                    className="text-sm"
                  />
                  {Boolean(block.payload.url) && isValidGoogleSlidesUrl(block.payload.url as string) && (
                    <div className="aspect-video w-full overflow-hidden rounded-lg border shadow-sm">
                      <iframe
                        src={getGoogleSlidesEmbedUrl(block.payload.url as string) || ''}
                        className="h-full w-full"
                        allowFullScreen
                        loading="lazy"
                      />
                    </div>
                  )}
                  {Boolean(block.payload.url) && !isValidGoogleSlidesUrl(block.payload.url as string) && (
                    <p className="text-sm text-amber-600">⚠️ URL Google Slides tidak valid</p>
                  )}
                  <Input
                    value={(block.payload.title as string) || ''}
                    onChange={(e) => onUpdate({ ...block.payload, title: e.target.value })}
                    placeholder="Judul presentasi (opsional)"
                    className="text-sm"
                  />
                </div>
              )}

              {/* ===== PDF BLOCK ===== */}
              {block.block_type === 'pdf' && (() => {
                const pdfUrl = (block.payload.url as string) || '';
                const fileName = (block.payload.file_name as string) || '';

                const handlePdfUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
                  const file = e.target.files?.[0];
                  if (!file) return;
                  if (!isValidFileType(file, 'pdf')) {
                    toast.error('File harus berformat PDF');
                    return;
                  }
                  if (file.size > MAX_DOCUMENT_SIZE) {
                    toast.error(`Ukuran file maksimal ${formatFileSize(MAX_DOCUMENT_SIZE)}`);
                    return;
                  }
                  const reader = new FileReader();
                  reader.onloadend = () => {
                    onUpdateFile(file, reader.result as string);
                    onUpdate({ file_name: file.name });
                  };
                  reader.readAsDataURL(file);
                };

                return (
                  <div className="space-y-4">
                    {pdfUrl || block.pendingPreview ? (
                      <div className="space-y-3">
                        <div className="flex items-center justify-between rounded-lg border bg-muted/30 p-3">
                          <div className="flex items-center gap-2">
                            <FileText className="h-5 w-5 text-red-500" />
                            <span className="text-sm font-medium truncate max-w-[200px]">
                              {fileName || 'document.pdf'}
                            </span>
                            {block.pendingFile && (
                              <Badge variant="outline" className="text-xs">Akan diupload</Badge>
                            )}
                          </div>
                          <label className="cursor-pointer">
                            <Button variant="outline" size="sm" asChild>
                              <span>Ganti File</span>
                            </Button>
                            <input
                              type="file"
                              accept={getAcceptedFileTypes('pdf')}
                              className="hidden"
                              onChange={handlePdfUpload}
                            />
                          </label>
                        </div>
                        {!block.pendingFile && pdfUrl && (
                          <div className="aspect-[3/4] max-h-[500px] w-full overflow-hidden rounded-lg border">
                            <iframe
                              src={`${pdfUrl}#toolbar=1&navpanes=0`}
                              className="h-full w-full"
                              title={fileName}
                            />
                          </div>
                        )}
                      </div>
                    ) : (
                      <label className="flex aspect-video cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed bg-muted/30 transition-colors hover:bg-muted/50">
                        <FileUp className="h-8 w-8 text-muted-foreground" />
                        <span className="mt-2 text-sm font-medium">Upload PDF</span>
                        <span className="text-xs text-muted-foreground">Max {formatFileSize(MAX_DOCUMENT_SIZE)}</span>
                        <input
                          type="file"
                          accept={getAcceptedFileTypes('pdf')}
                          className="hidden"
                          onChange={handlePdfUpload}
                        />
                      </label>
                    )}
                    <Input
                      value={(block.payload.title as string) || ''}
                      onChange={(e) => onUpdate({ ...block.payload, title: e.target.value })}
                      placeholder="Judul dokumen (opsional)"
                      className="text-sm"
                    />
                  </div>
                );
              })()}

              {/* ===== DOC BLOCK ===== */}
              {block.block_type === 'doc' && (() => {
                const docUrl = (block.payload.url as string) || '';
                const fileName = (block.payload.file_name as string) || '';

                const handleDocUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
                  const file = e.target.files?.[0];
                  if (!file) return;
                  if (!isValidFileType(file, 'doc')) {
                    toast.error('File harus berformat DOC atau DOCX');
                    return;
                  }
                  if (file.size > MAX_DOCUMENT_SIZE) {
                    toast.error(`Ukuran file maksimal ${formatFileSize(MAX_DOCUMENT_SIZE)}`);
                    return;
                  }
                  const reader = new FileReader();
                  reader.onloadend = () => {
                    onUpdateFile(file, reader.result as string);
                    onUpdate({ file_name: file.name });
                  };
                  reader.readAsDataURL(file);
                };

                return (
                  <div className="space-y-4">
                    {docUrl || block.pendingPreview ? (
                      <div className="space-y-3">
                        <div className="flex items-center justify-between rounded-lg border bg-muted/30 p-3">
                          <div className="flex items-center gap-2">
                            <FileUp className="h-5 w-5 text-blue-500" />
                            <span className="text-sm font-medium truncate max-w-[200px]">
                              {fileName || 'document.docx'}
                            </span>
                            {block.pendingFile && (
                              <Badge variant="outline" className="text-xs">Akan diupload</Badge>
                            )}
                          </div>
                          <label className="cursor-pointer">
                            <Button variant="outline" size="sm" asChild>
                              <span>Ganti File</span>
                            </Button>
                            <input
                              type="file"
                              accept={getAcceptedFileTypes('doc')}
                              className="hidden"
                              onChange={handleDocUpload}
                            />
                          </label>
                        </div>
                        {!block.pendingFile && docUrl && (
                          <div className="aspect-[3/4] max-h-[500px] w-full overflow-hidden rounded-lg border">
                            <iframe
                              src={getGoogleDocsViewerUrl(docUrl)}
                              className="h-full w-full"
                              title={fileName}
                            />
                          </div>
                        )}
                      </div>
                    ) : (
                      <label className="flex aspect-video cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed bg-muted/30 transition-colors hover:bg-muted/50">
                        <FileUp className="h-8 w-8 text-muted-foreground" />
                        <span className="mt-2 text-sm font-medium">Upload Word Document</span>
                        <span className="text-xs text-muted-foreground">.doc, .docx • Max {formatFileSize(MAX_DOCUMENT_SIZE)}</span>
                        <input
                          type="file"
                          accept={getAcceptedFileTypes('doc')}
                          className="hidden"
                          onChange={handleDocUpload}
                        />
                      </label>
                    )}
                    <Input
                      value={(block.payload.title as string) || ''}
                      onChange={(e) => onUpdate({ ...block.payload, title: e.target.value })}
                      placeholder="Judul dokumen (opsional)"
                      className="text-sm"
                    />
                  </div>
                );
              })()}


            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
