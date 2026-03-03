'use client';

import { useState } from 'react';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Loader2 } from 'lucide-react';
import { ImageBlockPayload } from '@/lib/types';

interface ImageBlockEditorProps {
  payload: ImageBlockPayload;
  isEditing: boolean;
  onSave: (payload: ImageBlockPayload) => void;
  onCancel: () => void;
  isSaving: boolean;
}

export function ImageBlockEditor({ payload, isEditing, onSave, onCancel, isSaving }: ImageBlockEditorProps) {
  const [url, setUrl] = useState(payload.url);
  const [caption, setCaption] = useState(payload.caption || '');

  if (!isEditing) {
    return (
      <figure className="space-y-2">
        {payload.url ? (
          <div className="relative aspect-video overflow-hidden rounded-lg bg-muted">
            <Image src={payload.url} alt={payload.caption || 'Image'} fill className="object-contain" />
          </div>
        ) : (
          <div className="flex aspect-video items-center justify-center rounded-lg bg-muted">
            <p className="text-muted-foreground">Belum ada gambar</p>
          </div>
        )}
        {payload.caption && (
          <figcaption className="text-center text-sm text-muted-foreground">{payload.caption}</figcaption>
        )}
      </figure>
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="image-url">URL Gambar</Label>
        <Input
          id="image-url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/image.jpg"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="image-caption">Caption (opsional)</Label>
        <Input
          id="image-caption"
          value={caption}
          onChange={(e) => setCaption(e.target.value)}
          placeholder="Deskripsi gambar"
        />
      </div>
      {url && (
        <div className="relative aspect-video overflow-hidden rounded-lg bg-muted">
          <Image src={url} alt={caption || 'Preview'} fill className="object-contain" />
        </div>
      )}
      <div className="flex gap-2">
        <Button onClick={() => onSave({ url, caption })} disabled={isSaving || !url}>
          {isSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Simpan
        </Button>
        <Button variant="outline" onClick={onCancel}>
          Batal
        </Button>
      </div>
    </div>
  );
}
