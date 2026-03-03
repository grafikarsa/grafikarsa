'use client';

import { useState } from 'react';
import {
  Plus,
  Trash2,
  GripVertical,
  Type,
  Image as ImageIcon,
  Youtube,
  Link2,
  Upload,
  Loader2,
  ChevronUp,
  ChevronDown,
} from 'lucide-react';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card } from '@/components/ui/card';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ContentBlockType } from '@/lib/types';
import { toast } from 'sonner';

export interface LocalContentBlock {
  id: string;
  block_type: ContentBlockType;
  block_order: number;
  payload: Record<string, unknown>;
  isNew?: boolean;
  file?: File; // For image blocks that need upload
}

interface ContentBlockEditorProps {
  blocks: LocalContentBlock[];
  onChange: (blocks: LocalContentBlock[]) => void;
  disabled?: boolean;
}

const blockTypeLabels: Record<ContentBlockType, { label: string; icon: React.ReactNode }> = {
  text: { label: 'Teks', icon: <Type className="h-4 w-4" /> },
  image: { label: 'Gambar', icon: <ImageIcon className="h-4 w-4" /> },
  youtube: { label: 'YouTube', icon: <Youtube className="h-4 w-4" /> },
  button: { label: 'Tombol', icon: <Link2 className="h-4 w-4" /> },
  table: { label: 'Tabel', icon: <Type className="h-4 w-4" /> },
  embed: { label: 'Embed', icon: <Type className="h-4 w-4" /> },
};

export function ContentBlockEditor({ blocks, onChange, disabled }: ContentBlockEditorProps) {
  const addBlock = (type: ContentBlockType) => {
    const newBlock: LocalContentBlock = {
      id: `temp-${Date.now()}`,
      block_type: type,
      block_order: blocks.length,
      payload: getDefaultPayload(type),
      isNew: true,
    };
    onChange([...blocks, newBlock]);
  };

  const updateBlock = (id: string, payload: Record<string, unknown>) => {
    onChange(
      blocks.map((b) => (b.id === id ? { ...b, payload: { ...b.payload, ...payload } } : b))
    );
  };

  const updateBlockFile = (id: string, file: File, previewUrl: string) => {
    onChange(
      blocks.map((b) =>
        b.id === id ? { ...b, file, payload: { ...b.payload, url: previewUrl } } : b
      )
    );
  };

  const removeBlock = (id: string) => {
    const newBlocks = blocks
      .filter((b) => b.id !== id)
      .map((b, i) => ({ ...b, block_order: i }));
    onChange(newBlocks);
  };

  const moveBlock = (id: string, direction: 'up' | 'down') => {
    const index = blocks.findIndex((b) => b.id === id);
    if (
      (direction === 'up' && index === 0) ||
      (direction === 'down' && index === blocks.length - 1)
    ) {
      return;
    }

    const newBlocks = [...blocks];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    [newBlocks[index], newBlocks[targetIndex]] = [newBlocks[targetIndex], newBlocks[index]];
    onChange(newBlocks.map((b, i) => ({ ...b, block_order: i })));
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-medium">Content Blocks</Label>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="sm" disabled={disabled}>
              <Plus className="mr-1.5 h-4 w-4" />
              Tambah Block
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => addBlock('text')}>
              <Type className="mr-2 h-4 w-4" />
              Teks
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => addBlock('image')}>
              <ImageIcon className="mr-2 h-4 w-4" />
              Gambar
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => addBlock('youtube')}>
              <Youtube className="mr-2 h-4 w-4" />
              YouTube
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => addBlock('button')}>
              <Link2 className="mr-2 h-4 w-4" />
              Tombol/Link
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {blocks.length === 0 ? (
        <div className="rounded-lg border-2 border-dashed bg-muted/30 p-6 text-center">
          <p className="text-sm text-muted-foreground">
            Belum ada content block. Klik "Tambah Block" untuk menambahkan konten.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {blocks.map((block, index) => (
            <BlockItem
              key={block.id}
              block={block}
              index={index}
              total={blocks.length}
              onUpdate={updateBlock}
              onUpdateFile={updateBlockFile}
              onRemove={removeBlock}
              onMove={moveBlock}
              disabled={disabled}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function BlockItem({
  block,
  index,
  total,
  onUpdate,
  onUpdateFile,
  onRemove,
  onMove,
  disabled,
}: {
  block: LocalContentBlock;
  index: number;
  total: number;
  onUpdate: (id: string, payload: Record<string, unknown>) => void;
  onUpdateFile: (id: string, file: File, previewUrl: string) => void;
  onRemove: (id: string) => void;
  onMove: (id: string, direction: 'up' | 'down') => void;
  disabled?: boolean;
}) {
  const typeInfo = blockTypeLabels[block.block_type];

  return (
    <Card className="overflow-hidden">
      <div className="flex items-center gap-2 border-b bg-muted/30 px-3 py-2">
        <GripVertical className="h-4 w-4 text-muted-foreground" />
        <div className="flex items-center gap-1.5 text-sm font-medium">
          {typeInfo.icon}
          {typeInfo.label}
        </div>
        <div className="ml-auto flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onMove(block.id, 'up')}
            disabled={disabled || index === 0}
          >
            <ChevronUp className="h-4 w-4" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onMove(block.id, 'down')}
            disabled={disabled || index === total - 1}
          >
            <ChevronDown className="h-4 w-4" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-destructive hover:text-destructive"
            onClick={() => onRemove(block.id)}
            disabled={disabled}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <div className="p-3">
        <BlockEditor
          block={block}
          onUpdate={onUpdate}
          onUpdateFile={onUpdateFile}
          disabled={disabled}
        />
      </div>
    </Card>
  );
}

function BlockEditor({
  block,
  onUpdate,
  onUpdateFile,
  disabled,
}: {
  block: LocalContentBlock;
  onUpdate: (id: string, payload: Record<string, unknown>) => void;
  onUpdateFile: (id: string, file: File, previewUrl: string) => void;
  disabled?: boolean;
}) {
  switch (block.block_type) {
    case 'text':
      return (
        <Textarea
          value={(block.payload.content as string) || ''}
          onChange={(e) => onUpdate(block.id, { content: e.target.value })}
          placeholder="Tulis konten teks di sini..."
          rows={4}
          disabled={disabled}
          className="resize-none"
        />
      );

    case 'image':
      return (
        <ImageBlockEditor
          block={block}
          onUpdate={onUpdate}
          onUpdateFile={onUpdateFile}
          disabled={disabled}
        />
      );

    case 'youtube':
      return (
        <div className="space-y-2">
          <Input
            value={(block.payload.video_id as string) || ''}
            onChange={(e) => onUpdate(block.id, { video_id: extractYoutubeId(e.target.value) })}
            placeholder="URL atau ID video YouTube"
            disabled={disabled}
          />
          {block.payload.video_id && (
            <div className="aspect-video overflow-hidden rounded-lg bg-muted">
              <iframe
                src={`https://www.youtube.com/embed/${block.payload.video_id}`}
                className="h-full w-full"
                allowFullScreen
              />
            </div>
          )}
        </div>
      );

    case 'button':
      return (
        <div className="grid gap-2 sm:grid-cols-2">
          <Input
            value={(block.payload.text as string) || ''}
            onChange={(e) => onUpdate(block.id, { text: e.target.value })}
            placeholder="Teks tombol"
            disabled={disabled}
          />
          <Input
            value={(block.payload.url as string) || ''}
            onChange={(e) => onUpdate(block.id, { url: e.target.value })}
            placeholder="URL tujuan"
            disabled={disabled}
          />
        </div>
      );

    default:
      return <p className="text-sm text-muted-foreground">Block type tidak didukung</p>;
  }
}

function ImageBlockEditor({
  block,
  onUpdate,
  onUpdateFile,
  disabled,
}: {
  block: LocalContentBlock;
  onUpdate: (id: string, payload: Record<string, unknown>) => void;
  onUpdateFile: (id: string, file: File, previewUrl: string) => void;
  disabled?: boolean;
}) {
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

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
      onUpdateFile(block.id, file, reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  const imageUrl = block.payload.url as string;

  return (
    <div className="space-y-2">
      {imageUrl ? (
        <div className="relative aspect-video overflow-hidden rounded-lg border bg-muted">
          <Image src={imageUrl} alt="Preview" fill className="object-contain" />
          <label className="absolute inset-0 flex cursor-pointer items-center justify-center bg-black/50 opacity-0 transition-opacity hover:opacity-100">
            <span className="rounded-md bg-white px-3 py-1.5 text-sm font-medium">
              Ganti Gambar
            </span>
            <input
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleFileChange}
              disabled={disabled}
            />
          </label>
        </div>
      ) : (
        <label className="flex aspect-video cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed bg-muted/30 transition-colors hover:bg-muted/50">
          <Upload className="mb-2 h-8 w-8 text-muted-foreground" />
          <span className="text-sm text-muted-foreground">Klik untuk upload gambar</span>
          <input
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleFileChange}
            disabled={disabled}
          />
        </label>
      )}
      <Input
        value={(block.payload.caption as string) || ''}
        onChange={(e) => onUpdate(block.id, { caption: e.target.value })}
        placeholder="Caption gambar (opsional)"
        disabled={disabled}
      />
    </div>
  );
}

function getDefaultPayload(type: ContentBlockType): Record<string, unknown> {
  switch (type) {
    case 'text':
      return { content: '' };
    case 'image':
      return { url: '', caption: '' };
    case 'youtube':
      return { video_id: '', title: '' };
    case 'button':
      return { text: '', url: '' };
    case 'table':
      return { headers: [], rows: [] };
    case 'embed':
      return { html: '', title: '' };
    default:
      return {};
  }
}

function extractYoutubeId(input: string): string {
  // Handle full URLs
  const urlMatch = input.match(
    /(?:youtube\.com\/(?:watch\?v=|embed\/)|youtu\.be\/)([a-zA-Z0-9_-]{11})/
  );
  if (urlMatch) return urlMatch[1];

  // Handle direct ID (11 characters)
  if (/^[a-zA-Z0-9_-]{11}$/.test(input)) return input;

  return input;
}
