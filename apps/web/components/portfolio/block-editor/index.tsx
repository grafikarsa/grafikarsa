'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, GripVertical, Trash2, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ContentBlock, ContentBlockType } from '@/lib/types';
import { portfoliosApi } from '@/lib/api';
import { TextBlockEditor } from './text-block';
import { ImageBlockEditor } from './image-block';
import { TableBlockEditor } from './table-block';
import { YoutubeBlockEditor } from './youtube-block';
import { ButtonBlockEditor } from './button-block';
import { EmbedBlockEditor } from './embed-block';

interface BlockEditorProps {
  portfolioId: string;
  blocks: ContentBlock[];
  setBlocks: React.Dispatch<React.SetStateAction<ContentBlock[]>>;
}

const blockTypes: { type: ContentBlockType; label: string; icon: string }[] = [
  { type: 'text', label: 'Teks', icon: '📝' },
  { type: 'image', label: 'Gambar', icon: '🖼️' },
  { type: 'table', label: 'Tabel', icon: '📊' },
  { type: 'youtube', label: 'YouTube', icon: '▶️' },
  { type: 'button', label: 'Tombol', icon: '🔘' },
  { type: 'embed', label: 'Embed', icon: '📦' },
];

export function BlockEditor({ portfolioId, blocks, setBlocks }: BlockEditorProps) {
  const queryClient = useQueryClient();
  const [editingBlockId, setEditingBlockId] = useState<string | null>(null);

  const addBlockMutation = useMutation({
    mutationFn: (blockType: ContentBlockType) => {
      const defaultPayload = getDefaultPayload(blockType);
      return portfoliosApi.addBlock(portfolioId, {
        block_type: blockType,
        block_order: blocks.length,
        payload: defaultPayload,
      });
    },
    onSuccess: (response) => {
      if (response.data) {
        setBlocks((prev) => [...prev, response.data!]);
        setEditingBlockId(response.data.id);
        toast.success('Block ditambahkan');
      }
    },
    onError: () => {
      toast.error('Gagal menambahkan block');
    },
  });

  const updateBlockMutation = useMutation({
    mutationFn: ({ blockId, data }: { blockId: string; data: Partial<ContentBlock> }) =>
      portfoliosApi.updateBlock(portfolioId, blockId, data),
    onSuccess: (response, { blockId }) => {
      if (response.data) {
        setBlocks((prev) => prev.map((b) => (b.id === blockId ? response.data! : b)));
        setEditingBlockId(null);
        toast.success('Block diperbarui');
      }
    },
    onError: () => {
      toast.error('Gagal memperbarui block');
    },
  });

  const deleteBlockMutation = useMutation({
    mutationFn: (blockId: string) => portfoliosApi.deleteBlock(portfolioId, blockId),
    onSuccess: (_, blockId) => {
      setBlocks((prev) => prev.filter((b) => b.id !== blockId));
      toast.success('Block dihapus');
    },
    onError: () => {
      toast.error('Gagal menghapus block');
    },
  });

  const sortedBlocks = [...blocks].sort((a, b) => a.block_order - b.block_order);

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Konten</CardTitle>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm" disabled={addBlockMutation.isPending}>
              {addBlockMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Plus className="mr-2 h-4 w-4" />
              )}
              Tambah Block
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            {blockTypes.map((bt) => (
              <DropdownMenuItem key={bt.type} onClick={() => addBlockMutation.mutate(bt.type)}>
                <span className="mr-2">{bt.icon}</span>
                {bt.label}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </CardHeader>
      <CardContent>
        {sortedBlocks.length === 0 ? (
          <p className="py-8 text-center text-muted-foreground">
            Belum ada konten. Klik &quot;Tambah Block&quot; untuk memulai.
          </p>
        ) : (
          <div className="space-y-4">
            {sortedBlocks.map((block) => (
              <BlockItem
                key={block.id}
                block={block}
                isEditing={editingBlockId === block.id}
                onEdit={() => setEditingBlockId(block.id)}
                onSave={(payload) =>
                  updateBlockMutation.mutate({ blockId: block.id, data: { payload } })
                }
                onCancel={() => setEditingBlockId(null)}
                onDelete={() => {
                  if (confirm('Yakin ingin menghapus block ini?')) {
                    deleteBlockMutation.mutate(block.id);
                  }
                }}
                isSaving={updateBlockMutation.isPending}
                isDeleting={deleteBlockMutation.isPending}
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

interface BlockItemProps {
  block: ContentBlock;
  isEditing: boolean;
  onEdit: () => void;
  onSave: (payload: ContentBlock['payload']) => void;
  onCancel: () => void;
  onDelete: () => void;
  isSaving: boolean;
  isDeleting: boolean;
}

function BlockItem({ block, isEditing, onEdit, onSave, onCancel, onDelete, isSaving, isDeleting }: BlockItemProps) {
  const blockLabel = blockTypes.find((bt) => bt.type === block.block_type)?.label || block.block_type;

  return (
    <div className="group rounded-lg border p-4">
      <div className="mb-2 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <GripVertical className="h-4 w-4 cursor-move text-muted-foreground" />
          <span className="text-sm font-medium">{blockLabel}</span>
        </div>
        <div className="flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
          {!isEditing && (
            <Button variant="ghost" size="sm" onClick={onEdit}>
              Edit
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={onDelete} disabled={isDeleting}>
            {isDeleting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
          </Button>
        </div>
      </div>
      <BlockContent
        block={block}
        isEditing={isEditing}
        onSave={onSave}
        onCancel={onCancel}
        isSaving={isSaving}
      />
    </div>
  );
}

function BlockContent({
  block,
  isEditing,
  onSave,
  onCancel,
  isSaving,
}: {
  block: ContentBlock;
  isEditing: boolean;
  onSave: (payload: ContentBlock['payload']) => void;
  onCancel: () => void;
  isSaving: boolean;
}) {
  const props = { payload: block.payload, isEditing, onSave, onCancel, isSaving };

  switch (block.block_type) {
    case 'text':
      return <TextBlockEditor {...props} />;
    case 'image':
      return <ImageBlockEditor {...props} />;
    case 'table':
      return <TableBlockEditor {...props} />;
    case 'youtube':
      return <YoutubeBlockEditor {...props} />;
    case 'button':
      return <ButtonBlockEditor {...props} />;
    case 'embed':
      return <EmbedBlockEditor {...props} />;
    default:
      return <p className="text-muted-foreground">Block type tidak dikenal</p>;
  }
}

function getDefaultPayload(type: ContentBlockType): ContentBlock['payload'] {
  switch (type) {
    case 'text':
      return { content: '<p>Tulis konten di sini...</p>' };
    case 'image':
      return { url: '', caption: '' };
    case 'table':
      return { headers: ['Kolom 1', 'Kolom 2'], rows: [['', '']] };
    case 'youtube':
      return { video_id: '', title: '' };
    case 'button':
      return { text: 'Klik di sini', url: '' };
    case 'embed':
      return { html: '', title: '' };
  }
}

export { TextBlockEditor } from './text-block';
export { ImageBlockEditor } from './image-block';
export { TableBlockEditor } from './table-block';
export { YoutubeBlockEditor } from './youtube-block';
export { ButtonBlockEditor } from './button-block';
export { EmbedBlockEditor } from './embed-block';
