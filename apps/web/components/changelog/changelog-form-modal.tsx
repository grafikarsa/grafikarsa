'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, Minus, Bug, RefreshCw, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import {
  getAdminChangelogById,
  createChangelog,
  updateChangelog,
} from '@/lib/api/changelog';
import type {
  ChangelogCategory,
  ChangelogSectionRequest,
  ChangelogContributorRequest,
  CreateChangelogRequest,
  ChangelogBlockRequest,
} from '@/lib/types/changelog';
import { cn } from '@/lib/utils';
import { ContributorPicker } from './contributor-picker';
import {
  ContentBlockEditor,
  LocalContentBlock,
} from '@/components/admin/content-block-editor';

const categories: {
  key: ChangelogCategory;
  label: string;
  icon: React.ElementType;
  color: string;
  bgColor: string;
}[] = [
  { key: 'added', label: 'Added', icon: Plus, color: 'text-green-600', bgColor: 'bg-green-100 dark:bg-green-900/30' },
  { key: 'updated', label: 'Updated', icon: RefreshCw, color: 'text-blue-600', bgColor: 'bg-blue-100 dark:bg-blue-900/30' },
  { key: 'removed', label: 'Removed', icon: Minus, color: 'text-red-600', bgColor: 'bg-red-100 dark:bg-red-900/30' },
  { key: 'fixed', label: 'Fixed', icon: Bug, color: 'text-orange-600', bgColor: 'bg-orange-100 dark:bg-orange-900/30' },
];

interface Props {
  open: boolean;
  onClose: () => void;
  editId?: string | null;
}

export function ChangelogFormModal({ open, onClose, editId }: Props) {
  const queryClient = useQueryClient();
  const isEditing = !!editId;

  // Form state
  const [version, setVersion] = useState('');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [releaseDate, setReleaseDate] = useState(new Date().toISOString().split('T')[0]);
  const [sectionBlocks, setSectionBlocks] = useState<Record<ChangelogCategory, LocalContentBlock[]>>({
    added: [],
    updated: [],
    removed: [],
    fixed: [],
  });
  const [contributors, setContributors] = useState<ChangelogContributorRequest[]>([]);
  const [initialUserMap, setInitialUserMap] = useState<
    Record<string, { id: string; username: string; nama: string; avatar_url?: string }>
  >({});

  // Fetch existing changelog for editing
  const { data: existingData, isLoading: isLoadingExisting } = useQuery({
    queryKey: ['admin-changelog', editId],
    queryFn: () => getAdminChangelogById(editId!),
    enabled: !!editId,
  });

  // Populate form when editing
  useEffect(() => {
    if (existingData?.data?.data) {
      const cl = existingData.data.data;
      setVersion(cl.version);
      setTitle(cl.title);
      setDescription(cl.description || '');
      setReleaseDate(cl.release_date);

      // Parse sections into blocks
      const newSectionBlocks: Record<ChangelogCategory, LocalContentBlock[]> = {
        added: [],
        updated: [],
        removed: [],
        fixed: [],
      };

      (cl.sections || []).forEach(
        (section: {
          category: ChangelogCategory;
          blocks: { id: string; block_type: string; payload: Record<string, unknown> }[];
        }) => {
          newSectionBlocks[section.category] = (section.blocks || []).map((block, index) => ({
            id: block.id,
            block_type: block.block_type as LocalContentBlock['block_type'],
            block_order: index,
            payload: block.payload,
          }));
        }
      );
      setSectionBlocks(newSectionBlocks);

      // Parse contributors and build user map for display
      const userMapData: Record<
        string,
        { id: string; username: string; nama: string; avatar_url?: string }
      > = {};
      const contributorsData = (cl.contributors || []).map(
        (c: {
          user: { id: string; username: string; nama: string; avatar_url?: string };
          contribution: string;
        }) => {
          if (c.user) {
            userMapData[c.user.id] = {
              id: c.user.id,
              username: c.user.username,
              nama: c.user.nama,
              avatar_url: c.user.avatar_url,
            };
          }
          return {
            user_id: c.user?.id || '',
            contribution: c.contribution,
          };
        }
      );
      setInitialUserMap(userMapData);
      setContributors(contributorsData);
    }
  }, [existingData]);

  // Reset form when closing
  useEffect(() => {
    if (!open) {
      setVersion('');
      setTitle('');
      setDescription('');
      setReleaseDate(new Date().toISOString().split('T')[0]);
      setSectionBlocks({ added: [], updated: [], removed: [], fixed: [] });
      setContributors([]);
      setInitialUserMap({});
    }
  }, [open]);

  const createMutation = useMutation({
    mutationFn: (data: CreateChangelogRequest) => createChangelog(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-changelogs'] });
      toast.success('Changelog berhasil dibuat');
      onClose();
    },
    onError: () => toast.error('Gagal membuat changelog'),
  });

  const updateMutation = useMutation({
    mutationFn: (data: CreateChangelogRequest) => updateChangelog(editId!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-changelogs'] });
      queryClient.invalidateQueries({ queryKey: ['admin-changelog', editId] });
      toast.success('Changelog berhasil diupdate');
      onClose();
    },
    onError: () => toast.error('Gagal mengupdate changelog'),
  });

  const handleSubmit = () => {
    // Validation
    if (!version.trim()) {
      toast.error('Version wajib diisi');
      return;
    }
    if (!title.trim()) {
      toast.error('Title wajib diisi');
      return;
    }

    // Check at least one section has blocks
    const filledSections = Object.entries(sectionBlocks).filter(
      ([, blocks]) => blocks.length > 0
    );
    if (filledSections.length === 0) {
      toast.error('Minimal 1 kategori harus memiliki content');
      return;
    }

    // Check at least one contributor
    if (contributors.length === 0) {
      toast.error('Minimal 1 contributor wajib diisi');
      return;
    }

    // Build sections data
    const sectionsData: ChangelogSectionRequest[] = filledSections.map(([category, blocks]) => ({
      category: category as ChangelogCategory,
      blocks: (blocks as LocalContentBlock[]).map((block) => ({
        block_type: block.block_type,
        payload: block.payload,
      })) as ChangelogBlockRequest[],
    }));

    const data: CreateChangelogRequest = {
      version: version.trim(),
      title: title.trim(),
      description: description.trim() || undefined,
      release_date: releaseDate,
      sections: sectionsData,
      contributors,
    };

    if (isEditing) {
      updateMutation.mutate(data);
    } else {
      createMutation.mutate(data);
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  const updateSectionBlocks = (category: ChangelogCategory, blocks: LocalContentBlock[]) => {
    setSectionBlocks((prev) => ({ ...prev, [category]: blocks }));
  };

  return (
    <Dialog open={open} onOpenChange={() => !isPending && onClose()}>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Changelog' : 'Buat Changelog Baru'}</DialogTitle>
        </DialogHeader>

        {isLoadingExisting ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="space-y-6 py-4">
            {/* Basic Info */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="version">Version *</Label>
                <Input
                  id="version"
                  placeholder="1.0.0"
                  value={version}
                  onChange={(e) => setVersion(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="release_date">Tanggal Rilis</Label>
                <Input
                  id="release_date"
                  type="date"
                  value={releaseDate}
                  onChange={(e) => setReleaseDate(e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="title">Judul *</Label>
              <Input
                id="title"
                placeholder="Judul changelog"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Deskripsi (opsional)</Label>
              <Textarea
                id="description"
                placeholder="Deskripsi singkat tentang update ini..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
              />
            </div>

            {/* Sections with Content Blocks */}
            <div className="space-y-2">
              <Label>Perubahan * (minimal 1 kategori)</Label>
              <Accordion type="multiple" className="w-full">
                {categories.map(({ key, label, icon: Icon, color, bgColor }) => {
                  const hasBlocks = sectionBlocks[key].length > 0;
                  return (
                    <AccordionItem key={key} value={key}>
                      <AccordionTrigger className="hover:no-underline">
                        <div className="flex items-center gap-2">
                          <div className={cn('rounded-md p-1.5', bgColor)}>
                            <Icon className={cn('h-4 w-4', color)} />
                          </div>
                          <span className="font-medium">{label}</span>
                          {hasBlocks && (
                            <span className="ml-2 rounded-full bg-green-500 px-2 py-0.5 text-xs text-white">
                              {sectionBlocks[key].length}
                            </span>
                          )}
                        </div>
                      </AccordionTrigger>
                      <AccordionContent>
                        <div className="pt-2">
                          <ContentBlockEditor
                            blocks={sectionBlocks[key]}
                            onChange={(blocks) => updateSectionBlocks(key, blocks)}
                            disabled={isPending}
                          />
                        </div>
                      </AccordionContent>
                    </AccordionItem>
                  );
                })}
              </Accordion>
            </div>

            {/* Contributors */}
            <div className="space-y-2">
              <Label>Contributors * (minimal 1)</Label>
              <ContributorPicker
                value={contributors}
                onChange={setContributors}
                initialUserMap={initialUserMap}
              />
            </div>
          </div>
        )}

        <div className="flex justify-end gap-2 border-t pt-4">
          <Button variant="outline" onClick={onClose} disabled={isPending}>
            Batal
          </Button>
          <Button onClick={handleSubmit} disabled={isPending || isLoadingExisting}>
            {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isEditing ? 'Simpan' : 'Buat Changelog'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
