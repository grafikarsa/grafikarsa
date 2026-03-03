'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';
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
  Eye,
  Pencil,
  Trash2,
  Search,
  Loader2,
  FileText,
  Upload,
  X,
  AlertCircle,
  GripVertical,
  Type,
  ImageIcon,
  Youtube,
  Link2,
  MoreVertical,
  CheckCircle,
  Clock,
  XCircle,
  Archive,
  FileEdit,
  Table as TableIcon,
  Code,
} from 'lucide-react';
import Image from 'next/image';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { adminPortfoliosApi, adminUsersApi, adminTagsApi, adminSeriesApi, uploadsApi } from '@/lib/api/admin';
import { portfoliosApi } from '@/lib/api';
import {
  PortfolioCard,
  Portfolio,
  Tag,
  Series,
  User as UserType,
  PortfolioStatus,
  ContentBlockType,
} from '@/lib/types';
import { BlockRenderer } from '@/components/portfolio/block-renderer';
import { useDebounce } from '@/lib/hooks/use-debounce';
import { formatDate } from '@/lib/utils/format';

// Types for local content blocks (before saving)
interface LocalContentBlock {
  id: string;
  block_type: ContentBlockType;
  block_order: number;
  payload: Record<string, unknown>;
  isNew?: boolean;
  // For pending image uploads
  pendingFile?: File;
  pendingPreview?: string;
}

const statusOptions = [
  { value: 'all', label: 'Semua Status' },
  { value: 'draft', label: 'Draft' },
  { value: 'pending_review', label: 'Pending Review' },
  { value: 'published', label: 'Published' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'archived', label: 'Archived' },
];

// Custom status styles with specific colors
const statusStyles: Record<PortfolioStatus, string> = {
  draft: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  pending_review: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  published: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  rejected: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  archived: 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400',
};

const blockTypeOptions = [
  { value: 'text', label: 'Teks', icon: Type },
  { value: 'image', label: 'Gambar', icon: ImageIcon },
  { value: 'youtube', label: 'YouTube', icon: Youtube },
  { value: 'button', label: 'Tombol/Link', icon: Link2 },
];

// Generate unique ID for local blocks
const generateId = () => `local-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;


export default function AdminPortfoliosPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('all');
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebounce(search, 300);

  const [selectedPortfolio, setSelectedPortfolio] = useState<PortfolioCard | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingPortfolio, setEditingPortfolio] = useState<PortfolioCard | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<PortfolioCard | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-portfolios', debouncedSearch, status, page],
    queryFn: () =>
      adminPortfoliosApi.getPortfolios({
        search: debouncedSearch || undefined,
        status: status === 'all' ? undefined : status,
        page,
        limit: 21,
      }),
  });

  const { data: detailData, isLoading: detailLoading } = useQuery({
    queryKey: ['portfolio-detail', selectedPortfolio?.id],
    queryFn: () => portfoliosApi.getPortfolioById(selectedPortfolio!.id),
    enabled: !!selectedPortfolio,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminPortfoliosApi.deletePortfolio(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-portfolios'] });
      toast.success('Portfolio berhasil dihapus');
      setDeleteTarget(null);
    },
    onError: () => toast.error('Gagal menghapus portfolio'),
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: PortfolioStatus }) =>
      adminPortfoliosApi.updatePortfolio(id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-portfolios'] });
      queryClient.invalidateQueries({ queryKey: ['admin-dashboard-stats'] });
      toast.success('Status portfolio berhasil diubah');
    },
    onError: () => toast.error('Gagal mengubah status portfolio'),
  });

  const portfolios = data?.data || [];
  const pagination = (data as { pagination?: { total_pages: number } })?.pagination;

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-9 w-48" />
          <Skeleton className="h-10 w-32" />
        </div>
        <div className="grid gap-6 sm:grid-cols-2 xl:grid-cols-3">
          {[...Array(6)].map((_, i) => (
            <Skeleton key={i} className="h-80" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Filters & Actions */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari judul portfolio..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-full sm:w-48">
            <SelectValue placeholder="Filter status" />
          </SelectTrigger>
          <SelectContent>
            {statusOptions.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Buat Portfolio
        </Button>
      </div>

      {portfolios.length === 0 ? (
        <Card className="border-dashed py-16">
          <div className="flex flex-col items-center justify-center">
            <div className="rounded-full bg-muted p-4">
              <FileText className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-lg font-semibold">Tidak ada portfolio</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {search || status !== 'all' ? 'Tidak ada portfolio yang sesuai' : 'Belum ada portfolio'}
            </p>
          </div>
        </Card>
      ) : (
        <>
          <div className="grid gap-6 sm:grid-cols-2 xl:grid-cols-3">
            {portfolios.map((portfolio) => (
              <PortfolioCardItem
                key={portfolio.id}
                portfolio={portfolio}
                onPreview={() => setSelectedPortfolio(portfolio)}
                onEdit={() => {
                  setEditingPortfolio(portfolio);
                  setShowEditModal(true);
                }}
                onDelete={() => setDeleteTarget(portfolio)}
                onStatusChange={(newStatus) => updateStatusMutation.mutate({ id: portfolio.id, status: newStatus as PortfolioStatus })}
                isUpdatingStatus={updateStatusMutation.isPending}
              />
            ))}
          </div>

          {pagination && pagination.total_pages > 1 && (
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                Previous
              </Button>
              <span className="text-sm text-muted-foreground">
                Halaman {page} dari {pagination.total_pages}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
                disabled={page === pagination.total_pages}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}

      <PreviewDialog
        portfolio={selectedPortfolio}
        detailData={detailData?.data}
        detailLoading={detailLoading}
        onClose={() => setSelectedPortfolio(null)}
      />

      <PortfolioFormModal
        open={showCreateModal}
        onOpenChange={setShowCreateModal}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['admin-portfolios'] })}
      />

      <PortfolioFormModal
        open={showEditModal}
        onOpenChange={setShowEditModal}
        portfolio={editingPortfolio}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['admin-portfolios'] })}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Hapus Portfolio"
        description={<>Yakin ingin menghapus portfolio <strong>{deleteTarget?.judul}</strong>?</>}
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  );
}


// Status change options with icons
const statusChangeOptions = [
  { value: 'draft', label: 'Draft', icon: FileEdit, color: 'text-gray-500' },
  { value: 'pending_review', label: 'Pending Review', icon: Clock, color: 'text-yellow-500' },
  { value: 'published', label: 'Published', icon: CheckCircle, color: 'text-green-500' },
  { value: 'rejected', label: 'Rejected', icon: XCircle, color: 'text-red-500' },
  { value: 'archived', label: 'Archived', icon: Archive, color: 'text-gray-400' },
];

// Portfolio Card Item Component
function PortfolioCardItem({
  portfolio,
  onPreview,
  onEdit,
  onDelete,
  onStatusChange,
  isUpdatingStatus,
}: {
  portfolio: PortfolioCard;
  onPreview: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onStatusChange: (status: string) => void;
  isUpdatingStatus: boolean;
}) {
  const firstTag = portfolio.tags?.[0];
  const displayDate = portfolio.published_at || portfolio.created_at;

  return (
    <Card className="group gap-0 overflow-hidden border py-0 transition-shadow hover:shadow-lg">
      <div className="p-3 pb-4">
        <div className="relative aspect-[4/3] w-full overflow-hidden rounded-xl bg-muted">
          {portfolio.thumbnail_url ? (
            <Image src={portfolio.thumbnail_url} alt={portfolio.judul} fill className="object-cover transition-transform group-hover:scale-105" />
          ) : (
            <div className="flex h-full items-center justify-center text-muted-foreground">No Image</div>
          )}
          <div className="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
            <Button variant="secondary" size="sm" onClick={onPreview}>
              <Eye className="mr-2 h-4 w-4" />
              Preview
            </Button>
          </div>
        </div>

        {/* Status badge with dropdown - moved outside image area */}
        <div className="mt-3 flex items-center justify-between gap-2">
          {firstTag ? (
            <Badge variant="secondary" className="rounded-full px-2.5 py-0.5 text-xs font-normal">
              {firstTag.nama}
            </Badge>
          ) : (
            <div />
          )}
          {portfolio.status && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button className={`flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium capitalize transition-opacity ${statusStyles[portfolio.status]} hover:opacity-80`}>
                  {isUpdatingStatus ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : null}
                  {portfolio.status.replace('_', ' ')}
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-44">
                <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
                  Ubah Status
                </div>
                <DropdownMenuSeparator />
                {statusChangeOptions.map((opt) => {
                  const Icon = opt.icon;
                  const isActive = portfolio.status === opt.value;
                  return (
                    <DropdownMenuItem
                      key={opt.value}
                      onClick={() => !isActive && onStatusChange(opt.value)}
                      disabled={isActive || isUpdatingStatus}
                      className={isActive ? 'bg-muted' : ''}
                    >
                      <Icon className={`mr-2 h-4 w-4 ${opt.color}`} />
                      {opt.label}
                      {isActive && <CheckCircle className="ml-auto h-3 w-3 text-green-500" />}
                    </DropdownMenuItem>
                  );
                })}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>

        <h3 className="mt-2 line-clamp-2 font-semibold leading-tight">{portfolio.judul}</h3>

        {portfolio.user && (
          <div className="mt-3 flex items-center gap-2">
            <Avatar className="h-8 w-8">
              <AvatarImage src={portfolio.user.avatar_url} alt={portfolio.user.nama} />
              <AvatarFallback className="text-xs">{portfolio.user.nama?.charAt(0)}</AvatarFallback>
            </Avatar>
            <div className="flex flex-col">
              <span className="text-sm font-medium leading-tight">{portfolio.user.nama}</span>
              <span className="text-xs text-muted-foreground">Posted on {formatDate(displayDate)}</span>
            </div>
          </div>
        )}

        <div className="mt-4 flex gap-2">
          <Button size="sm" variant="outline" className="flex-1" onClick={onEdit}>
            <Pencil className="mr-1.5 h-4 w-4" />
            Edit
          </Button>
          <Button size="sm" variant="destructive" className="flex-1" onClick={onDelete}>
            <Trash2 className="mr-1.5 h-4 w-4" />
            Hapus
          </Button>
        </div>
      </div>
    </Card>
  );
}

// Preview Dialog Component
function PreviewDialog({
  portfolio,
  detailData,
  detailLoading,
  onClose,
}: {
  portfolio: PortfolioCard | null;
  detailData?: Portfolio;
  detailLoading: boolean;
  onClose: () => void;
}) {
  if (!portfolio) return null;

  return (
    <Dialog open={!!portfolio} onOpenChange={() => onClose()}>
      <DialogContent className="flex max-h-[95vh] w-full max-w-4xl flex-col gap-0 overflow-hidden p-0">
        <div className="border-b bg-background px-4 py-4 sm:px-6">
          <DialogTitle className="line-clamp-2 text-lg font-semibold">{portfolio.judul}</DialogTitle>
          <DialogDescription asChild>
            <div className="mt-1 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <span className="flex items-center gap-1.5">
                <Avatar className="h-5 w-5">
                  <AvatarImage src={portfolio.user?.avatar_url} />
                  <AvatarFallback className="text-[10px]">{portfolio.user?.nama?.charAt(0)}</AvatarFallback>
                </Avatar>
                <span className="font-medium text-foreground">{portfolio.user?.nama}</span>
              </span>
              {portfolio.status && (
                <Badge className={`text-xs capitalize ${statusStyles[portfolio.status]}`}>
                  {portfolio.status.replace('_', ' ')}
                </Badge>
              )}
            </div>
          </DialogDescription>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-5 sm:px-6">
          {portfolio.thumbnail_url && (
            <div className="relative mb-6 aspect-video w-full overflow-hidden rounded-lg bg-muted">
              <Image src={portfolio.thumbnail_url} alt={portfolio.judul} fill className="object-cover" />
            </div>
          )}

          {detailLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-32 w-full rounded-lg" />
              <Skeleton className="h-6 w-3/4" />
            </div>
          ) : detailData?.content_blocks?.length ? (
            <div className="prose prose-sm max-w-none dark:prose-invert">
              <BlockRenderer blocks={detailData.content_blocks} />
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <AlertCircle className="h-8 w-8 text-muted-foreground" />
              <p className="mt-3 font-medium">Tidak ada konten</p>
            </div>
          )}
        </div>

        <div className="flex justify-between border-t bg-background px-4 py-3 sm:px-6">
          <Button variant="ghost" onClick={onClose}>Tutup</Button>
          <Link href={`/${portfolio.user?.username}/${portfolio.slug}`}>
            <Button variant="outline">
              <Eye className="mr-2 h-4 w-4" />
              Lihat Portfolio
            </Button>
          </Link>
        </div>
      </DialogContent>
    </Dialog>
  );
}


// Portfolio Form Modal - Create & Edit with Content Blocks and Drag-Drop
function PortfolioFormModal({
  open,
  onOpenChange,
  portfolio,
  onSuccess,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  portfolio?: PortfolioCard | null;
  onSuccess: () => void;
}) {
  const isEdit = !!portfolio;

  // Form state
  const [judul, setJudul] = useState('');
  const [selectedStatus, setSelectedStatus] = useState<PortfolioStatus>('draft');
  const [selectedUser, setSelectedUser] = useState<UserType | null>(null);
  const [selectedTags, setSelectedTags] = useState<Tag[]>([]);
  const [selectedSeries, setSelectedSeries] = useState<Series | null>(null);
  const [thumbnailFile, setThumbnailFile] = useState<File | null>(null);
  const [thumbnailPreview, setThumbnailPreview] = useState<string | null>(null);
  const [thumbnailUrl, setThumbnailUrl] = useState<string | null>(null);
  const [contentBlocks, setContentBlocks] = useState<LocalContentBlock[]>([]);
  const [userSearch, setUserSearch] = useState('');
  const [userPopoverOpen, setUserPopoverOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitProgress, setSubmitProgress] = useState('');

  const debouncedUserSearch = useDebounce(userSearch, 300);

  // DnD sensors
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // Fetch users
  const { data: usersData, isLoading: usersLoading } = useQuery({
    queryKey: ['admin-users-search', debouncedUserSearch],
    queryFn: () => adminUsersApi.getUsers({ search: debouncedUserSearch || undefined, limit: 20 }),
    enabled: open && !isEdit,
  });

  // Fetch tags
  const { data: tagsData } = useQuery({
    queryKey: ['admin-tags'],
    queryFn: () => adminTagsApi.getTags({ limit: 100 }),
    enabled: open,
  });

  // Fetch series
  const { data: seriesData } = useQuery({
    queryKey: ['admin-series'],
    queryFn: () => adminSeriesApi.getSeries({ limit: 100 }),
    enabled: open,
  });

  // Fetch portfolio detail for edit
  const { data: detailData } = useQuery({
    queryKey: ['portfolio-edit-detail', portfolio?.id],
    queryFn: () => adminPortfoliosApi.getPortfolio(portfolio!.id),
    enabled: open && isEdit && !!portfolio,
  });

  // Reset form
  const resetForm = useCallback(() => {
    setJudul('');
    setSelectedStatus('draft');
    setSelectedUser(null);
    setSelectedTags([]);
    setSelectedSeries(null);
    setThumbnailFile(null);
    setThumbnailPreview(null);
    setThumbnailUrl(null);
    setContentBlocks([]);
    setUserSearch('');
    setIsSubmitting(false);
    setSubmitProgress('');
  }, []);

  // Sync form state when editing
  useEffect(() => {
    if (open && isEdit && portfolio) {
      setJudul(portfolio.judul);
      setSelectedStatus(portfolio.status || 'draft');
      setSelectedTags(portfolio.tags || []);
      // Convert PortfolioSeries to Series format for the form
      setSelectedSeries(portfolio.series ? { ...portfolio.series, is_active: true, created_at: '' } as Series : null);
      setThumbnailUrl(portfolio.thumbnail_url || null);
      if (detailData?.data?.content_blocks) {
        setContentBlocks(
          detailData.data.content_blocks.map((b) => ({
            ...b,
            payload: b.payload as unknown as Record<string, unknown>,
            isNew: false,
          }))
        );
      }
    }
  }, [open, isEdit, portfolio, detailData]);

  useEffect(() => {
    if (!open) resetForm();
  }, [open, resetForm]);

  const users = usersData?.data || [];
  const tags = tagsData?.data || [];

  // Merge active series with portfolio's existing series (including inactive ones)
  const seriesList = useMemo(() => {
    const activeSeries = seriesData?.data || [];
    const portfolioSeries = portfolio?.series;

    // Create a map of series by ID
    const seriesMap = new Map(activeSeries.map((s) => [s.id, s]));

    // Add portfolio's series if not in active list (inactive series)
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

  const handleSeriesSelect = (s: Series | null) => {
    if (s?.id === selectedSeries?.id) {
      setSelectedSeries(null);
    } else {
      setSelectedSeries(s);
    }
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
    setContentBlocks((prev) =>
      prev.map((b) => (b.id === id ? { ...b, payload } : b))
    );
  };

  const updateBlockFile = (id: string, file: File | null, preview: string) => {
    setContentBlocks((prev) =>
      prev.map((b) => (b.id === id ? { ...b, pendingFile: file || undefined, pendingPreview: preview || undefined } : b))
    );
  };

  const removeBlock = (id: string) => {
    setContentBlocks((prev) => prev.filter((b) => b.id !== id));
  };

  // Drag and drop handler
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

  // Submit handler
  const handleSubmit = async () => {
    if (!judul.trim()) {
      toast.error('Judul wajib diisi');
      return;
    }
    if (!isEdit && !selectedUser) {
      toast.error('User wajib dipilih');
      return;
    }

    setIsSubmitting(true);

    try {
      let portfolioId = portfolio?.id;

      // Step 1: Create or Update Portfolio
      if (isEdit && portfolioId) {
        setSubmitProgress('Memperbarui portfolio...');
        await adminPortfoliosApi.updatePortfolio(portfolioId, {
          judul: judul.trim(),
          status: selectedStatus,
          tag_ids: selectedTags.map((t) => t.id),
          series_id: selectedSeries?.id,
        });
      } else {
        setSubmitProgress('Membuat portfolio...');
        const createRes = await adminPortfoliosApi.createPortfolio({
          judul: judul.trim(),
          user_id: selectedUser!.id,
          tag_ids: selectedTags.map((t) => t.id),
        });
        if (!createRes.data) throw new Error('Gagal membuat portfolio');
        portfolioId = createRes.data.id;
        // Set status after creation if not draft
        if (selectedStatus !== 'draft') {
          await adminPortfoliosApi.updatePortfolio(portfolioId, { status: selectedStatus, series_id: selectedSeries?.id });
        } else if (selectedSeries) {
          await adminPortfoliosApi.updatePortfolio(portfolioId, { series_id: selectedSeries.id });
        }
      }

      // Step 2: Upload thumbnail if new file
      if (thumbnailFile && portfolioId) {
        setSubmitProgress('Mengupload thumbnail...');
        const thumbUrl = await uploadsApi.uploadFile(thumbnailFile, 'thumbnail', portfolioId);
        await adminPortfoliosApi.updatePortfolio(portfolioId, { thumbnail_url: thumbUrl });
      }

      // Step 3: Handle content blocks
      if (portfolioId) {
        // For edit mode, delete removed blocks first
        if (isEdit && detailData?.data?.content_blocks) {
          const existingIds = detailData.data.content_blocks.map((b) => b.id);
          const currentIds = contentBlocks.filter((b) => !b.isNew).map((b) => b.id);
          const deletedIds = existingIds.filter((id) => !currentIds.includes(id));

          for (const blockId of deletedIds) {
            setSubmitProgress('Menghapus content block...');
            await adminPortfoliosApi.deleteBlock(portfolioId, blockId);
          }
        }

        // Process blocks - upload images first if needed
        for (let i = 0; i < contentBlocks.length; i++) {
          const block = contentBlocks[i];
          let blockPayload = { ...block.payload };

          // Upload pending image file
          if (block.block_type === 'image' && block.pendingFile) {
            setSubmitProgress(`Mengupload gambar block ${i + 1}/${contentBlocks.length}...`);
            const imageUrl = await uploadsApi.uploadFile(block.pendingFile, 'portfolio_image', portfolioId);
            blockPayload = { ...blockPayload, url: imageUrl };
          }

          if (block.isNew) {
            setSubmitProgress(`Menambah content block ${i + 1}/${contentBlocks.length}...`);
            await adminPortfoliosApi.addBlock(portfolioId, {
              block_type: block.block_type,
              block_order: i,
              payload: blockPayload,
            });
          } else {
            // Update existing block
            setSubmitProgress(`Memperbarui content block ${i + 1}/${contentBlocks.length}...`);
            await adminPortfoliosApi.updateBlock(portfolioId, block.id, {
              payload: blockPayload,
            });
          }
        }

        // Reorder blocks
        if (contentBlocks.length > 0) {
          setSubmitProgress('Menyusun ulang content blocks...');
          const blockOrders = contentBlocks.map((b, i) => ({ id: b.id, order: i }));
          // Only reorder if we have non-new blocks
          const existingBlocks = blockOrders.filter((b) => !b.id.startsWith('local-'));
          if (existingBlocks.length > 0) {
            await adminPortfoliosApi.reorderBlocks(portfolioId, existingBlocks);
          }
        }
      }

      toast.success(isEdit ? 'Portfolio berhasil diperbarui' : 'Portfolio berhasil dibuat');
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      console.error(error);
      toast.error(isEdit ? 'Gagal memperbarui portfolio' : 'Gagal membuat portfolio');
    } finally {
      setIsSubmitting(false);
      setSubmitProgress('');
    }
  };

  const displayThumbnail = thumbnailPreview || thumbnailUrl;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[95vh] w-full max-w-2xl flex-col gap-0 overflow-hidden p-0">
        {/* Header */}
        <div className="border-b bg-background px-4 py-4 sm:px-6">
          <div className="flex items-center gap-3">
            <div className={`rounded-full p-2.5 ${isEdit ? 'bg-blue-100 dark:bg-blue-900/30' : 'bg-primary/10'}`}>
              {isEdit ? (
                <Pencil className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              ) : (
                <Plus className="h-5 w-5 text-primary" />
              )}
            </div>
            <div>
              <DialogTitle className="text-lg font-semibold">
                {isEdit ? 'Edit Portfolio' : 'Buat Portfolio Baru'}
              </DialogTitle>
              <DialogDescription className="text-sm">
                {isEdit ? 'Perbarui informasi dan konten portfolio' : 'Buat portfolio baru dengan konten'}
              </DialogDescription>
            </div>
          </div>
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto px-4 py-5 sm:px-6">
          <div className="space-y-6">
            {/* Thumbnail */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">Thumbnail</Label>
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
                <label className="flex aspect-video w-full cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed bg-muted/30 hover:bg-muted/50">
                  <Upload className="h-8 w-8 text-muted-foreground" />
                  <span className="mt-2 text-sm">Klik untuk upload thumbnail</span>
                  <span className="text-xs text-muted-foreground">PNG, JPG, max 5MB</span>
                  <input type="file" accept="image/*" className="hidden" onChange={handleThumbnailChange} />
                </label>
              )}
            </div>

            {/* Judul & Status */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="judul" className="text-sm font-medium">
                  Judul Portfolio <span className="text-destructive">*</span>
                </Label>
                <Input
                  id="judul"
                  value={judul}
                  onChange={(e) => setJudul(e.target.value)}
                  placeholder="Masukkan judul portfolio"
                />
              </div>
              <div className="space-y-2">
                <Label className="text-sm font-medium">Status</Label>
                <Select
                  value={selectedStatus}
                  onValueChange={(v) => setSelectedStatus(v as PortfolioStatus)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {statusChangeOptions.map((opt) => {
                      const Icon = opt.icon;
                      return (
                        <SelectItem key={opt.value} value={opt.value}>
                          <div className="flex items-center gap-2">
                            <Icon className={`h-4 w-4 ${opt.color}`} />
                            {opt.label}
                          </div>
                        </SelectItem>
                      );
                    })}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* User Selection (only for create) */}
            {!isEdit && (
              <div className="space-y-2">
                <Label className="text-sm font-medium">
                  Pemilik Portfolio <span className="text-destructive">*</span>
                </Label>
                <Popover open={userPopoverOpen} onOpenChange={setUserPopoverOpen}>
                  <PopoverTrigger asChild>
                    <Button variant="outline" className="h-auto w-full justify-start px-3 py-2.5 font-normal">
                      {selectedUser ? (
                        <div className="flex items-center gap-2.5">
                          <Avatar className="h-8 w-8">
                            <AvatarImage src={selectedUser.avatar_url} />
                            <AvatarFallback className="text-xs">{selectedUser.nama?.charAt(0)}</AvatarFallback>
                          </Avatar>
                          <div className="flex flex-col items-start text-left">
                            <span className="font-medium">{selectedUser.nama}</span>
                            <span className="text-xs text-muted-foreground">@{selectedUser.username}</span>
                          </div>
                        </div>
                      ) : (
                        <span className="text-muted-foreground">Pilih user...</span>
                      )}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-[350px] p-0" align="start">
                    <Command>
                      <CommandInput placeholder="Cari user..." value={userSearch} onValueChange={setUserSearch} />
                      <CommandList className="max-h-[200px]">
                        {usersLoading ? (
                          <div className="flex items-center justify-center p-4">
                            <Loader2 className="h-4 w-4 animate-spin" />
                          </div>
                        ) : users.length === 0 ? (
                          <CommandEmpty>Tidak ada user</CommandEmpty>
                        ) : (
                          <CommandGroup>
                            {users.map((user) => (
                              <CommandItem
                                key={user.id}
                                value={user.username}
                                onSelect={() => {
                                  setSelectedUser(user);
                                  setUserPopoverOpen(false);
                                }}
                              >
                                <Avatar className="mr-2 h-8 w-8">
                                  <AvatarImage src={user.avatar_url} />
                                  <AvatarFallback className="text-xs">{user.nama?.charAt(0)}</AvatarFallback>
                                </Avatar>
                                <div className="flex flex-col">
                                  <span className="font-medium">{user.nama}</span>
                                  <span className="text-xs text-muted-foreground">@{user.username}</span>
                                </div>
                              </CommandItem>
                            ))}
                          </CommandGroup>
                        )}
                      </CommandList>
                    </Command>
                  </PopoverContent>
                </Popover>
              </div>
            )}

            {/* Owner info for edit */}
            {isEdit && portfolio?.user && (
              <div className="space-y-2">
                <Label className="text-sm font-medium">Pemilik Portfolio</Label>
                <div className="flex items-center gap-3 rounded-lg border bg-muted/30 p-3">
                  <Avatar className="h-10 w-10">
                    <AvatarImage src={portfolio.user.avatar_url} />
                    <AvatarFallback>{portfolio.user.nama?.charAt(0)}</AvatarFallback>
                  </Avatar>
                  <div>
                    <p className="font-medium">{portfolio.user.nama}</p>
                    <p className="text-sm text-muted-foreground">@{portfolio.user.username}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Tags */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">Tags</Label>
              <div className="flex flex-wrap gap-2 rounded-lg border bg-muted/30 p-3">
                {tags.length === 0 ? (
                  <span className="text-sm text-muted-foreground">Tidak ada tags</span>
                ) : (
                  tags.map((tag) => {
                    const isSelected = selectedTags.some((t) => t.id === tag.id);
                    return (
                      <Badge
                        key={tag.id}
                        variant={isSelected ? 'default' : 'outline'}
                        className="cursor-pointer"
                        onClick={() => toggleTag(tag)}
                      >
                        {tag.nama}
                        {isSelected && <X className="ml-1 h-3 w-3" />}
                      </Badge>
                    );
                  })
                )}
              </div>
            </div>

            {/* Series */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">Series Template</Label>
              <div className="flex flex-wrap gap-2 rounded-lg border bg-muted/30 p-3">
                {seriesList.length === 0 ? (
                  <span className="text-sm text-muted-foreground">Tidak ada series</span>
                ) : (
                  seriesList.map((s) => {
                    const isSelected = selectedSeries?.id === s.id;
                    const isInactive = !s.is_active;
                    return (
                      <Badge
                        key={s.id}
                        variant={isSelected ? 'default' : 'outline'}
                        className={`cursor-pointer ${
                          isSelected
                            ? isInactive
                              ? 'bg-blue-400/70 hover:bg-blue-500/70'
                              : 'bg-blue-500 hover:bg-blue-600'
                            : isInactive
                              ? 'opacity-60'
                              : ''
                        }`}
                        onClick={() => handleSeriesSelect(isSelected ? null : s)}
                        title={isInactive ? 'Series ini sudah tidak aktif' : undefined}
                      >
                        {s.nama}
                        {isInactive && <span className="ml-1 text-xs opacity-70">(nonaktif)</span>}
                        {isSelected && <X className="ml-1 h-3 w-3" />}
                      </Badge>
                    );
                  })
                )}
              </div>
            </div>

            {/* Content Blocks with Drag and Drop */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label className="text-sm font-medium">Konten Portfolio</Label>
                <Popover>
                  <PopoverTrigger asChild>
                    <Button type="button" variant="outline" size="sm">
                      <Plus className="mr-1.5 h-4 w-4" />
                      Tambah Block
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-48 p-2" align="end">
                    {blockTypeOptions.map((opt) => (
                      <Button
                        key={opt.value}
                        variant="ghost"
                        className="w-full justify-start"
                        onClick={() => addBlock(opt.value as ContentBlockType)}
                      >
                        <opt.icon className="mr-2 h-4 w-4" />
                        {opt.label}
                      </Button>
                    ))}
                  </PopoverContent>
                </Popover>
              </div>

              {contentBlocks.length === 0 ? (
                <div className="rounded-lg border-2 border-dashed bg-muted/30 p-8 text-center">
                  <FileText className="mx-auto h-8 w-8 text-muted-foreground" />
                  <p className="mt-2 text-sm text-muted-foreground">
                    Belum ada konten. Klik &quot;Tambah Block&quot; untuk menambahkan.
                  </p>
                </div>
              ) : (
                <DndContext
                  sensors={sensors}
                  collisionDetection={closestCenter}
                  onDragEnd={handleDragEnd}
                >
                  <SortableContext
                    items={contentBlocks.map((b) => b.id)}
                    strategy={verticalListSortingStrategy}
                  >
                    <div className="space-y-3">
                      {contentBlocks.map((block) => (
                        <SortableContentBlockEditor
                          key={block.id}
                          block={block}
                          onUpdate={(payload) => updateBlock(block.id, payload)}
                          onUpdateFile={(file: File | null, preview: string) => updateBlockFile(block.id, file, preview)}
                          onRemove={() => removeBlock(block.id)}
                        />
                      ))}
                    </div>
                  </SortableContext>
                </DndContext>
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex flex-col-reverse gap-2 border-t bg-background px-4 py-4 sm:flex-row sm:justify-between sm:px-6">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            Batal
          </Button>
          <div className="flex items-center gap-2">
            {isSubmitting && submitProgress && (
              <span className="text-sm text-muted-foreground">{submitProgress}</span>
            )}
            <Button onClick={handleSubmit} disabled={isSubmitting}>
              {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isEdit ? 'Simpan Perubahan' : 'Buat Portfolio'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}


// Sortable Content Block Editor Component with Drag Handle
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
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: block.id });

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
      reader.onloadend = () => {
        onUpdateFile(file, reader.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  const displayImageUrl = block.pendingPreview || (block.payload.url as string) || '';

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`rounded-lg border bg-background ${isDragging ? 'shadow-lg ring-2 ring-primary' : ''}`}
    >
      {/* Block Header with Drag Handle */}
      <div className="flex items-center gap-2 border-b bg-muted/30 px-3 py-2">
        <button
          type="button"
          className="cursor-grab touch-none rounded p-1 hover:bg-muted active:cursor-grabbing"
          {...attributes}
          {...listeners}
        >
          <GripVertical className="h-4 w-4 text-muted-foreground" />
        </button>
        <Icon className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm font-medium capitalize">{block.block_type}</span>
        <div className="ml-auto flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-destructive hover:text-destructive"
            onClick={onRemove}
          >
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

        {block.block_type === 'image' && (() => {
          const urlValue = ((block.payload.url as string) || '').trim();
          const hasUrlInput = !!urlValue;
          const hasPendingFile = !!block.pendingFile || !!block.pendingPreview;
          const isUploadDisabled = hasUrlInput && !hasPendingFile;
          const isUrlDisabled = hasPendingFile;

          return (
            <div className="space-y-3">
              {/* Image Preview - shows for both uploaded and URL images */}
              {displayImageUrl && (
                <div className="space-y-2">
                  <Label className="text-xs font-medium">Preview Gambar</Label>
                  <div className="group relative aspect-video w-full overflow-hidden rounded-lg border bg-muted">
                    <Image
                      src={displayImageUrl}
                      alt="Preview"
                      fill
                      className="object-contain"
                      onError={(e) => {
                        const target = e.target as HTMLImageElement;
                        target.style.display = 'none';
                        target.parentElement?.classList.add('preview-error');
                      }}
                    />
                    <div className="absolute inset-0 flex items-center justify-center gap-2 bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
                      {!isUploadDisabled && (
                        <label className="cursor-pointer">
                          <Button type="button" variant="secondary" size="sm" asChild>
                            <span><Upload className="mr-1.5 h-4 w-4" />Ganti</span>
                          </Button>
                          <input type="file" accept="image/*" className="hidden" onChange={handleImageFileChange} />
                        </label>
                      )}
                      <Button
                        type="button"
                        variant="destructive"
                        size="sm"
                        onClick={() => {
                          onUpdate({ ...block.payload, url: '' });
                          onUpdateFile(null, '');
                        }}
                      >
                        <Trash2 className="mr-1.5 h-4 w-4" />Hapus
                      </Button>
                    </div>
                  </div>
                  {hasPendingFile && (
                    <p className="text-xs text-amber-600">Gambar akan diupload saat menyimpan</p>
                  )}
                </div>
              )}

              {/* Image Upload - only show when no image */}
              {!displayImageUrl && (
                <div className="space-y-2">
                  <Label className="text-xs font-medium">Upload Gambar</Label>
                  <label className={`flex aspect-video w-full flex-col items-center justify-center rounded-lg border-2 border-dashed ${isUploadDisabled ? 'cursor-not-allowed bg-muted/20 opacity-50' : 'cursor-pointer bg-muted/30 hover:bg-muted/50'}`}>
                    <Upload className="h-8 w-8 text-muted-foreground" />
                    <span className="mt-2 text-sm">{isUploadDisabled ? 'Hapus URL untuk upload' : 'Klik untuk upload gambar'}</span>
                    <span className="text-xs text-muted-foreground">PNG, JPG, GIF, max 10MB</span>
                    <input type="file" accept="image/*" className="hidden" onChange={handleImageFileChange} disabled={isUploadDisabled} />
                  </label>
                </div>
              )}

              {/* URL Input (alternative) */}
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">Atau masukkan URL gambar</Label>
                <Input
                  value={(block.payload.url as string) || ''}
                  onChange={(e) => onUpdate({ ...block.payload, url: e.target.value })}
                  placeholder="https://example.com/image.jpg"
                  className="text-sm"
                  disabled={isUrlDisabled}
                />
                {isUrlDisabled && (
                  <p className="text-xs text-muted-foreground">Hapus gambar yang diupload untuk menggunakan URL</p>
                )}
              </div>

              {/* Caption */}
              <div>
                <Label className="text-xs">Caption (opsional)</Label>
                <Input
                  value={(block.payload.caption as string) || ''}
                  onChange={(e) => onUpdate({ ...block.payload, caption: e.target.value })}
                  placeholder="Deskripsi gambar"
                />
              </div>
            </div>
          );
        })()}

        {block.block_type === 'youtube' && (() => {
          // Extract video ID from URL or use as-is if already an ID
          const extractVideoId = (input: string): string => {
            if (!input) return '';
            // Already a video ID (11 chars, alphanumeric with - and _)
            if (/^[a-zA-Z0-9_-]{11}$/.test(input)) return input;
            // YouTube URL patterns
            const patterns = [
              /(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([a-zA-Z0-9_-]{11})/,
              /youtube\.com\/shorts\/([a-zA-Z0-9_-]{11})/,
            ];
            for (const pattern of patterns) {
              const match = input.match(pattern);
              if (match) return match[1];
            }
            return input; // Return as-is if no match
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
                <p className="mt-1 text-xs text-muted-foreground">
                  Bisa paste link YouTube langsung atau video ID saja
                </p>
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
                  <iframe
                    src={`https://www.youtube.com/embed/${videoId}`}
                    className="h-full w-full"
                    allowFullScreen
                  />
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
            {typeof block.payload.text === 'string' && typeof block.payload.url === 'string' && block.payload.text && block.payload.url && (
              <div className="pt-2">
                <Button variant="outline" size="sm" asChild>
                  <a href={block.payload.url} target="_blank" rel="noopener noreferrer">
                    {block.payload.text}
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
