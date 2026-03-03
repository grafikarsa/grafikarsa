'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Loader2 } from 'lucide-react';
import { YoutubeBlockPayload } from '@/lib/types';

interface YoutubeBlockEditorProps {
  payload: YoutubeBlockPayload;
  isEditing: boolean;
  onSave: (payload: YoutubeBlockPayload) => void;
  onCancel: () => void;
  isSaving: boolean;
}

function extractVideoId(input: string): string {
  // Handle various YouTube URL formats
  const patterns = [
    /(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([^&\n?#]+)/,
    /^([a-zA-Z0-9_-]{11})$/,
  ];
  for (const pattern of patterns) {
    const match = input.match(pattern);
    if (match) return match[1];
  }
  return input;
}

export function YoutubeBlockEditor({ payload, isEditing, onSave, onCancel, isSaving }: YoutubeBlockEditorProps) {
  const [videoInput, setVideoInput] = useState(payload.video_id);
  const [title, setTitle] = useState(payload.title || '');

  const videoId = extractVideoId(videoInput);

  if (!isEditing) {
    return (
      <div className="space-y-2">
        {payload.video_id ? (
          <div className="relative aspect-video overflow-hidden rounded-lg">
            <iframe
              src={`https://www.youtube.com/embed/${payload.video_id}`}
              title={payload.title || 'YouTube video'}
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
              allowFullScreen
              className="absolute inset-0 h-full w-full"
            />
          </div>
        ) : (
          <div className="flex aspect-video items-center justify-center rounded-lg bg-muted">
            <p className="text-muted-foreground">Belum ada video</p>
          </div>
        )}
        {payload.title && <p className="text-center text-sm text-muted-foreground">{payload.title}</p>}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="youtube-url">URL atau Video ID YouTube</Label>
        <Input
          id="youtube-url"
          value={videoInput}
          onChange={(e) => setVideoInput(e.target.value)}
          placeholder="https://youtube.com/watch?v=... atau video ID"
        />
        {videoInput && videoId && (
          <p className="text-xs text-muted-foreground">Video ID: {videoId}</p>
        )}
      </div>
      <div className="space-y-2">
        <Label htmlFor="youtube-title">Judul (opsional)</Label>
        <Input
          id="youtube-title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Judul video"
        />
      </div>
      {videoId && (
        <div className="relative aspect-video overflow-hidden rounded-lg">
          <iframe
            src={`https://www.youtube.com/embed/${videoId}`}
            title={title || 'Preview'}
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
            allowFullScreen
            className="absolute inset-0 h-full w-full"
          />
        </div>
      )}
      <div className="flex gap-2">
        <Button onClick={() => onSave({ video_id: videoId, title })} disabled={isSaving || !videoId}>
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
