'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Loader2 } from 'lucide-react';
import { EmbedBlockPayload } from '@/lib/types';

interface EmbedBlockEditorProps {
  payload: EmbedBlockPayload;
  isEditing: boolean;
  onSave: (payload: EmbedBlockPayload) => void;
  onCancel: () => void;
  isSaving: boolean;
}

export function EmbedBlockEditor({ payload, isEditing, onSave, onCancel, isSaving }: EmbedBlockEditorProps) {
  const [html, setHtml] = useState(payload.html);
  const [title, setTitle] = useState(payload.title || '');

  if (!isEditing) {
    return (
      <div className="space-y-2">
        {payload.html ? (
          <div
            className="overflow-hidden rounded-lg"
            dangerouslySetInnerHTML={{ __html: payload.html }}
          />
        ) : (
          <div className="flex aspect-video items-center justify-center rounded-lg bg-muted">
            <p className="text-muted-foreground">Belum ada embed</p>
          </div>
        )}
        {payload.title && <p className="text-center text-sm text-muted-foreground">{payload.title}</p>}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="embed-html">Kode Embed (HTML/iframe)</Label>
        <Textarea
          id="embed-html"
          value={html}
          onChange={(e) => setHtml(e.target.value)}
          rows={6}
          placeholder='<iframe src="..." width="100%" height="400"></iframe>'
          className="font-mono text-sm"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="embed-title">Judul (opsional)</Label>
        <Input
          id="embed-title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Deskripsi embed"
        />
      </div>
      {html && (
        <div className="space-y-2">
          <p className="text-sm font-medium">Preview:</p>
          <div
            className="overflow-hidden rounded-lg border"
            dangerouslySetInnerHTML={{ __html: html }}
          />
        </div>
      )}
      <div className="flex gap-2">
        <Button onClick={() => onSave({ html, title })} disabled={isSaving || !html}>
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
