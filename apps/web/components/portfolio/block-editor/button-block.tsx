'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Loader2, ExternalLink } from 'lucide-react';
import { ButtonBlockPayload } from '@/lib/types';

interface ButtonBlockEditorProps {
  payload: ButtonBlockPayload;
  isEditing: boolean;
  onSave: (payload: ButtonBlockPayload) => void;
  onCancel: () => void;
  isSaving: boolean;
}

export function ButtonBlockEditor({ payload, isEditing, onSave, onCancel, isSaving }: ButtonBlockEditorProps) {
  const [text, setText] = useState(payload.text);
  const [url, setUrl] = useState(payload.url);

  if (!isEditing) {
    return (
      <div className="flex justify-center">
        {payload.url ? (
          <a href={payload.url} target="_blank" rel="noopener noreferrer">
            <Button className="gap-2">
              {payload.text}
              <ExternalLink className="h-4 w-4" />
            </Button>
          </a>
        ) : (
          <Button disabled className="gap-2">
            {payload.text || 'Tombol'}
            <ExternalLink className="h-4 w-4" />
          </Button>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="button-text">Teks Tombol</Label>
        <Input
          id="button-text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Klik di sini"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="button-url">URL Tujuan</Label>
        <Input
          id="button-url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com"
        />
      </div>
      <div className="flex justify-center">
        <Button className="gap-2" disabled>
          {text || 'Preview'}
          <ExternalLink className="h-4 w-4" />
        </Button>
      </div>
      <div className="flex gap-2">
        <Button onClick={() => onSave({ text, url })} disabled={isSaving || !text || !url}>
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
