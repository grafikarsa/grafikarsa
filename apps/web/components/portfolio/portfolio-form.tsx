'use client';

import { useState, useEffect, useCallback } from 'react';
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
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
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
  Table as TableIcon,
  Trash2,
  Send,
  Save,
  Archive,
  ArchiveRestore,
  AlertCircle,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Separator } from '@/components/ui/separator';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { portfoliosApi, tagsApi } from '@/lib/api';
import { uploadsApi } from '@/lib/api/admin';
import { useAuthStore } from '@/lib/stores/auth-store';
import { Portfolio, Tag, ContentBlockType } from '@/lib/types';

// Types for local content blocks
interface LocalContentBlock {
  id: string;
  block_type: ContentBlockType;
  block_order: number;
  payload: Record<string, unknown>;
  isNew?: boolean;
  pendingFile?: File;
  pendingPreview?: string;
}

const blockTypeOptions = [
  { value: 'text', label: 'Teks', icon: Type, description: 'Paragraf teks atau HTML' },
  { value: 'image', label: 'Gambar', icon: ImageIcon, description: 'Upload atau URL gambar' },
  { value: 'youtube', label: 'YouTube', icon: Youtube, description: 'Embed video YouTube' },
  { value: 'table', label: 'Tabel', icon: TableIcon, description: 'Tabel data' },
  { value: 'button', label: 'Tombol/Link', icon: Link2, description: 'Tombol dengan link' },
];

const generateId = () => `local-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;

interface PortfolioFormProps {
  portfolio: Portfolio;
  isEdit?: boolean;
}

export function PortfolioForm({ portfolio }: PortfolioFormProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user } = useAuthStore();

  // Form state
  const [judul, setJudul] = useState(portfolio.judul);
  const [selectedTags, setSelectedTags] = useState<Tag[]>(portfolio.tags || []);
  const [thumbnailFile, setThumbnailFile] = useState<File | null>(null);
  const [thumbnailPreview, setThumbnailPreview] = useState<string | null>(null);
  const [thumbnailUrl, setThumbnailUrl] = useState<string | null>(portfolio.thumbnail_url || null);
  const [contentBlocks, setContentBlocks] = useState<LocalContentBlock[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitProgress, setSubmitProgress] = useState('');
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  // Initialize content blocks from portfolio
  useEffect(() => {
    if (portfolio.content_blocks) {
      setContentBlocks(
        portfolio.content_blocks.map((b) => ({
          ...b,
          payload: b.payload as unknown as Record<string, unknown>,
          isNew: false,
        }))
      );
    }
  }, [portfolio.content_blocks]);

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

  const toggleTag = (tag: Tag) => {
    setSelectedTags((prev) =>
      prev.some((t) => t.id === tag.id) ? prev.filter((t) => t.id !== tag.id) : [...prev, tag]
    );
  };

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
  };

  const handleDragEnd = (event: DragEndEvent) => {
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
      default: return {};
    }
  };

  // Save handler
  const handleSave = async (submitForReview: boolean = false) => {
    if (!judul.trim()) {
      toast.error('Judul wajib diisi');
      return;
    }

    setIsSubmitting(true);

    try {
      // Step 1: Update portfolio basic info
      setSubmitProgress('Menyimpan portfolio...');
      await portfoliosApi.updatePortfolio(portfolio.id, {
        judul: judul.trim(),
        tag_ids: selectedTags.map((t) => t.id),
      });

      // Step 2: Upload thumbnail if new file
      if (thumbnailFile) {
        setSubmitProgress('Mengupload thumbnail...');
        const thumbUrl = await uploadsApi.uploadFile(thumbnailFile, 'thumbnail', portfolio.id);
        await portfoliosApi.updatePortfolio(portfolio.id, { thumbnail_url: thumbUrl });
      } else if (thumbnailUrl !== portfolio.thumbnail_url) {
        // Thumbnail was removed or URL changed
        await portfoliosApi.updatePortfolio(portfolio.id, { thumbnail_url: thumbnailUrl || undefined });
      }

      // Step 3: Handle content blocks - delete removed blocks first
      if (portfolio.content_blocks) {
        const existingIds = portfolio.content_blocks.map((b) => b.id);
        const currentIds = contentBlocks.filter((b) => !b.isNew).map((b) => b.id);
        const deletedIds = existingIds.filter((id) => !currentIds.includes(id));

        for (const blockId of deletedIds) {
          setSubmitProgress('Menghapus content block...');
          await portfoliosApi.deleteBlock(portfolio.id, blockId);
        }
      }

      // Step 4: Process blocks - upload images and add/update blocks
      for (let i = 0; i < contentBlocks.length; i++) {
        const block = contentBlocks[i];
        let blockPayload = { ...block.payload };

        // Upload pending image file
        if (block.block_type === 'image' && block.pendingFile) {
          setSubmitProgress(`Mengupload gambar ${i + 1}/${contentBlocks.length}...`);
          const imageUrl = await uploadsApi.uploadFile(block.pendingFile, 'portfolio_image', portfolio.id);
          blockPayload = { ...blockPayload, url: imageUrl };
        }

        if (block.isNew) {
          setSubmitProgress(`Menambah konten ${i + 1}/${contentBlocks.length}...`);
          await portfoliosApi.addBlock(portfolio.id, {
            block_type: block.block_type,
            block_order: i,
            payload: blockPayload,
          });
        } else {
          setSubmitProgress(`Memperbarui konten ${i + 1}/${contentBlocks.length}...`);
          await portfoliosApi.updateBlock(portfolio.id, block.id, { payload: blockPayload });
        }
      }

      // Step 5: Reorder blocks
      if (contentBlocks.length > 0) {
        const existingBlocks = contentBlocks.filter((b) => !b.isNew && !b.id.startsWith('local-'));
        if (existingBlocks.length > 0) {
          setSubmitProgress('Menyusun ulang konten...');
          await portfoliosApi.reorderBlocks(
            portfolio.id,
            existingBlocks.map((b, i) => ({ id: b.id, order: i }))
          );
        }
      }

      // Step 6: Submit for review if requested
      if (submitForReview) {
        setSubmitProgress('Mengirim untuk review...');
        await portfoliosApi.submitPortfolio(portfolio.id);
        toast.success('Portfolio berhasil disimpan dan dikirim untuk review');
      } else {
        toast.success('Portfolio berhasil disimpan');
      }

      queryClient.invalidateQueries({ queryKey: ['portfolio', portfolio.slug] });
      queryClient.invalidateQueries({ queryKey: ['my-portfolios'] });
      
      if (submitForReview) {
        router.push(`/${user?.username}/${portfolio.slug}`);
      }
    } catch (error) {
      console.error(error);
      toast.error('Gagal menyimpan portfolio');
    } finally {
      setIsSubmitting(false);
      setSubmitProgress('');
    }
  };

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => portfoliosApi.deletePortfolio(portfolio.id),
    onSuccess: () => {
      toast.success('Portfolio berhasil dihapus');
      router.push(`/${user?.username}`);
    },
    onError: () => toast.error('Gagal menghapus portfolio'),
  });

  // Archive mutation
  const archiveMutation = useMutation({
    mutationFn: () => 
      portfolio.status === 'archived' 
        ? portfoliosApi.unarchivePortfolio(portfolio.id)
        : portfoliosApi.archivePortfolio(portfolio.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portfolio', portfolio.slug] });
      toast.success(portfolio.status === 'archived' ? 'Portfolio berhasil diaktifkan kembali' : 'Portfolio berhasil diarsipkan');
    },
    onError: () => toast.error('Gagal mengubah status portfolio'),
  });

  const displayThumbnail = thumbnailPreview || thumbnailUrl;
  const canSubmitForReview = portfolio.status === 'draft' || portfolio.status === 'rejected';
  const canDelete = portfolio.status === 'draft';
  const canArchive = portfolio.status === 'published' || portfolio.status === 'archived';

  return (
    <div className="space-y-6">
      {/* Admin Review Note */}
      {portfolio.admin_review_note && (
        <div className="flex gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4">
          <AlertCircle className="h-5 w-5 shrink-0 text-amber-500" />
          <div>
            <p className="text-sm font-medium text-amber-700 dark:text-amber-400">Catatan dari Admin</p>
            <p className="mt-1 text-sm text-amber-600 dark:text-amber-300">{portfolio.admin_review_note}</p>
          </div>
        </div>
      )}

      {/* Thumbnail */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Thumbnail</CardTitle>
          <CardDescription>Gambar utama yang akan ditampilkan di katalog</CardDescription>
        </CardHeader>
        <CardContent>
          {displayThumbnail ? (
            <div className="group relative aspect-video w-full overflow-hidden rounded-lg border bg-muted">
              <Image src={displayThumbnail} alt="Thumbnail" fill className="object-cover" />
              <div className="absolute inset-0 flex items-center justify-center gap-2 bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
                <label className="cursor-pointer">
                  <Button type="button" variant="secondary" size="sm" asChild>
                    <span><Upload className="mr-1.5 h-4 w-4" />Ganti</span>
                  </Button>
                  <input type="file" accept="image/*" className="hidden" onChange={handleThumbnailChange} />
                </label>
                <Button type="button" variant="destructive" size="sm" onClick={removeThumbnail}>
                  <Trash2 className="mr-1.5 h-4 w-4" />Hapus
                </Button>
              </div>
            </div>
          ) : (
            <label className="flex aspect-video w-full cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed bg-muted/30 transition-colors hover:bg-muted/50">
              <Upload className="h-8 w-8 text-muted-foreground" />
              <span className="mt-2 text-sm font-medium">Klik untuk upload thumbnail</span>
              <span className="text-xs text-muted-foreground">PNG, JPG, WebP, max 5MB</span>
              <input type="file" accept="image/*" className="hidden" onChange={handleThumbnailChange} />
            </label>
          )}
        </CardContent>
      </Card>

      {/* Title */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Judul Portfolio <span className="text-destructive">*</span></CardTitle>
        </CardHeader>
        <CardContent>
          <Input
            value={judul}
            onChange={(e) => setJudul(e.target.value)}
            placeholder="Masukkan judul portfolio"
            className="text-lg"
          />
        </CardContent>
      </Card>

      {/* Tags */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Tags</CardTitle>
          <CardDescription>Pilih kategori yang sesuai dengan portfoliomu</CardDescription>
        </CardHeader>
        <CardContent>
          {tags.length === 0 ? (
            <p className="text-sm text-muted-foreground">Tidak ada tags tersedia</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {tags.map((tag) => {
                const isSelected = selectedTags.some((t) => t.id === tag.id);
                return (
                  <Badge
                    key={tag.id}
                    variant={isSelected ? 'default' : 'outline'}
                    className="cursor-pointer transition-colors"
                    onClick={() => toggleTag(tag)}
                  >
                    {tag.nama}
                    {isSelected && <X className="ml-1 h-3 w-3" />}
                  </Badge>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Content Blocks */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base">Konten Portfolio</CardTitle>
              <CardDescription>Tambahkan teks, gambar, video, atau link</CardDescription>
            </div>
            <Popover>
              <PopoverTrigger asChild>
                <Button type="button" variant="outline" size="sm">
                  <Plus className="mr-1.5 h-4 w-4" />
                  Tambah
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-56 p-2" align="end">
                {blockTypeOptions.map((opt) => (
                  <Button
                    key={opt.value}
                    variant="ghost"
                    className="w-full justify-start"
                    onClick={() => addBlock(opt.value as ContentBlockType)}
                  >
                    <opt.icon className="mr-2 h-4 w-4" />
                    <div className="flex flex-col items-start">
                      <span>{opt.label}</span>
                      <span className="text-xs text-muted-foreground">{opt.description}</span>
                    </div>
                  </Button>
                ))}
              </PopoverContent>
            </Popover>
          </div>
        </CardHeader>
        <CardContent>
          {contentBlocks.length === 0 ? (
            <div className="rounded-lg border-2 border-dashed bg-muted/30 p-8 text-center">
              <FileText className="mx-auto h-8 w-8 text-muted-foreground" />
              <p className="mt-2 text-sm text-muted-foreground">
                Belum ada konten. Klik &quot;Tambah&quot; untuk menambahkan.
              </p>
            </div>
          ) : (
            <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
              <SortableContext items={contentBlocks.map((b) => b.id)} strategy={verticalListSortingStrategy}>
                <div className="space-y-3">
                  {contentBlocks.map((block) => (
                    <SortableContentBlockEditor
                      key={block.id}
                      block={block}
                      onUpdate={(payload) => updateBlock(block.id, payload)}
                      onUpdateFile={(file, preview) => updateBlockFile(block.id, file, preview)}
                      onRemove={() => removeBlock(block.id)}
                    />
                  ))}
                </div>
              </SortableContext>
            </DndContext>
          )}
        </CardContent>
      </Card>

      <Separator />

      {/* Actions */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex gap-2">
          {canDelete && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteDialogOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              Hapus
            </Button>
          )}
          {canArchive && (
            <Button variant="outline" size="sm" onClick={() => archiveMutation.mutate()} disabled={archiveMutation.isPending}>
              {archiveMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : portfolio.status === 'archived' ? (
                <ArchiveRestore className="mr-2 h-4 w-4" />
              ) : (
                <Archive className="mr-2 h-4 w-4" />
              )}
              {portfolio.status === 'archived' ? 'Aktifkan Kembali' : 'Arsipkan'}
            </Button>
          )}
        </div>
        <div className="flex items-center gap-2">
          {isSubmitting && submitProgress && (
            <span className="text-sm text-muted-foreground">{submitProgress}</span>
          )}
          <Button variant="outline" onClick={() => handleSave(false)} disabled={isSubmitting}>
            {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Save className="mr-2 h-4 w-4" />}
            Simpan
          </Button>
          {canSubmitForReview && (
            <Button onClick={() => handleSave(true)} disabled={isSubmitting}>
              {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Send className="mr-2 h-4 w-4" />}
              Kirim untuk Review
            </Button>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Hapus Portfolio"
        description={<>Yakin ingin menghapus portfolio <strong>{portfolio.judul}</strong>? Tindakan ini tidak dapat dibatalkan.</>}
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}


// Sortable Content Block Editor Component
function SortableContentBlockEditor({
  block,
  onUpdate,
  onUpdateFile,
  onRemove,
}: {
  block: LocalContentBlock;
  onUpdate: (payload: Record<string, unknown>) => void;
  onUpdateFile: (file: File | null, preview: string) => void;
  onRemove: () => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: block.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
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
    <div ref={setNodeRef} style={style} className={`rounded-lg border bg-background ${isDragging ? 'shadow-lg ring-2 ring-primary' : ''}`}>
      {/* Block Header */}
      <div className="flex items-center gap-2 border-b bg-muted/30 px-3 py-2">
        <button type="button" className="cursor-grab touch-none rounded p-1 hover:bg-muted active:cursor-grabbing" {...attributes} {...listeners}>
          <GripVertical className="h-4 w-4 text-muted-foreground" />
        </button>
        <Icon className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm font-medium capitalize">{block.block_type}</span>
        <div className="ml-auto">
          <Button type="button" variant="ghost" size="icon" className="h-7 w-7 text-destructive hover:text-destructive" onClick={onRemove}>
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Block Content */}
      <div className="p-3">
        {block.block_type === 'text' && (
          <Textarea
            value={(block.payload.content as string) || ''}
            onChange={(e) => onUpdate({ content: e.target.value })}
            placeholder="Masukkan teks konten... (mendukung HTML)"
            rows={4}
            className="resize-none"
          />
        )}

        {block.block_type === 'image' && (
          <div className="space-y-3">
            {displayImageUrl ? (
              <div className="space-y-2">
                <Label className="text-xs font-medium">Preview Gambar</Label>
                <div className="group relative aspect-video w-full overflow-hidden rounded-lg border bg-muted">
                  <Image src={displayImageUrl} alt="Preview" fill className="object-contain" />
                  <div className="absolute inset-0 flex items-center justify-center gap-2 bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
                    <label className="cursor-pointer">
                      <Button type="button" variant="secondary" size="sm" asChild>
                        <span><Upload className="mr-1.5 h-4 w-4" />Ganti</span>
                      </Button>
                      <input type="file" accept="image/*" className="hidden" onChange={handleImageFileChange} />
                    </label>
                    <Button type="button" variant="destructive" size="sm" onClick={() => { onUpdate({ ...block.payload, url: '' }); onUpdateFile(null, ''); }}>
                      <Trash2 className="mr-1.5 h-4 w-4" />Hapus
                    </Button>
                  </div>
                </div>
                {block.pendingFile && <p className="text-xs text-amber-600">Gambar akan diupload saat menyimpan</p>}
              </div>
            ) : (
              <div className="space-y-2">
                <Label className="text-xs font-medium">Upload Gambar</Label>
                <label className="flex aspect-video w-full cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed bg-muted/30 hover:bg-muted/50">
                  <Upload className="h-8 w-8 text-muted-foreground" />
                  <span className="mt-2 text-sm">Klik untuk upload gambar</span>
                  <span className="text-xs text-muted-foreground">PNG, JPG, GIF, max 10MB</span>
                  <input type="file" accept="image/*" className="hidden" onChange={handleImageFileChange} />
                </label>
              </div>
            )}
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">Atau masukkan URL gambar</Label>
              <Input
                value={(block.payload.url as string) || ''}
                onChange={(e) => onUpdate({ ...block.payload, url: e.target.value })}
                placeholder="https://example.com/image.jpg"
                className="text-sm"
                disabled={!!block.pendingFile}
              />
            </div>
            <div>
              <Label className="text-xs">Caption (opsional)</Label>
              <Input
                value={(block.payload.caption as string) || ''}
                onChange={(e) => onUpdate({ ...block.payload, caption: e.target.value })}
                placeholder="Deskripsi gambar"
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

          const handleYoutubeInput = (value: string) => {
            const videoId = extractVideoId(value.trim());
            onUpdate({ ...block.payload, video_id: videoId });
          };

          const videoId = (block.payload.video_id as string) || '';

          return (
            <div className="space-y-3">
              <div>
                <Label className="text-xs">Link atau Video ID YouTube</Label>
                <Input
                  value={videoId}
                  onChange={(e) => handleYoutubeInput(e.target.value)}
                  placeholder="https://youtube.com/watch?v=xxx atau video ID"
                />
                <p className="mt-1 text-xs text-muted-foreground">Bisa paste link YouTube langsung atau video ID saja</p>
              </div>
              <div>
                <Label className="text-xs">Judul (opsional)</Label>
                <Input
                  value={(block.payload.title as string) || ''}
                  onChange={(e) => onUpdate({ ...block.payload, title: e.target.value })}
                  placeholder="Judul video"
                />
              </div>
              {videoId && (
                <div className="aspect-video w-full overflow-hidden rounded-lg border">
                  <iframe src={`https://www.youtube.com/embed/${videoId}`} className="h-full w-full" allowFullScreen />
                </div>
              )}
            </div>
          );
        })()}

        {block.block_type === 'button' && (
          <div className="space-y-3">
            <div>
              <Label className="text-xs">Teks Tombol</Label>
              <Input
                value={(block.payload.text as string) || ''}
                onChange={(e) => onUpdate({ ...block.payload, text: e.target.value })}
                placeholder="Lihat Demo"
              />
            </div>
            <div>
              <Label className="text-xs">URL Link</Label>
              <Input
                value={(block.payload.url as string) || ''}
                onChange={(e) => onUpdate({ ...block.payload, url: e.target.value })}
                placeholder="https://example.com"
              />
            </div>
            {block.payload.text && block.payload.url && (
              <div className="pt-2">
                <Button variant="outline" size="sm" asChild>
                  <a href={block.payload.url as string} target="_blank" rel="noopener noreferrer">
                    {block.payload.text as string}
                  </a>
                </Button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
