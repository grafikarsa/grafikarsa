'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Loader2 } from 'lucide-react';
import { TextBlockPayload } from '@/lib/types';

interface TextBlockEditorProps {
  payload: TextBlockPayload;
  isEditing: boolean;
  onSave: (payload: TextBlockPayload) => void;
  onCancel: () => void;
  isSaving: boolean;
}

export function TextBlockEditor({ payload, isEditing, onSave, onCancel, isSaving }: TextBlockEditorProps) {
  const [content, setContent] = useState(payload.content);

  if (!isEditing) {
    return (
      <div
        className="prose prose-neutral dark:prose-invert max-w-none"
        dangerouslySetInnerHTML={{ __html: payload.content }}
      />
    );
  }

  return (
    <div className="space-y-4">
      <Textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        rows={8}
        placeholder="Tulis konten HTML di sini..."
        className="font-mono text-sm"
      />
      <p className="text-xs text-muted-foreground">
        Mendukung HTML dasar: &lt;p&gt;, &lt;strong&gt;, &lt;em&gt;, &lt;ul&gt;, &lt;ol&gt;, &lt;li&gt;, &lt;h1&gt;-&lt;h6&gt;, &lt;a&gt;
      </p>
      <div className="flex gap-2">
        <Button onClick={() => onSave({ content })} disabled={isSaving}>
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
