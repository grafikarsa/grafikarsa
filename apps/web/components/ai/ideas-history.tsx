'use client';

import { useState } from 'react';
import { History, ChevronDown, ChevronUp, Trash2, Eye } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import type { ProjectIdea } from '@/lib/api/ai';

interface GenerationHistory {
  id: string;
  timestamp: number;
  formData: {
    jurusan: string;
    interests: string[];
    project_type: string;
    difficulty: string;
  };
  ideas: ProjectIdea[];
}

interface IdeasHistoryProps {
  history: GenerationHistory[];
  onLoadHistory: (historyItem: GenerationHistory) => void;
  onDeleteHistory: (id: string) => void;
  onClearAll: () => void;
}

export function IdeasHistory({ history, onLoadHistory, onDeleteHistory, onClearAll }: IdeasHistoryProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  if (history.length === 0) return null;

  const formatDate = (timestamp: number) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Baru saja';
    if (diffMins < 60) return `${diffMins} menit lalu`;
    if (diffHours < 24) return `${diffHours} jam lalu`;
    if (diffDays < 7) return `${diffDays} hari lalu`;
    
    return date.toLocaleDateString('id-ID', { 
      day: 'numeric', 
      month: 'short', 
      year: 'numeric' 
    });
  };

  const PROJECT_TYPES: Record<string, string> = {
    web_app: 'Aplikasi Web',
    mobile_app: 'Aplikasi Mobile',
    desktop_app: 'Aplikasi Desktop',
    design_project: 'Proyek Desain',
    animation: 'Animasi',
    video_editing: 'Video Editing',
    game: 'Game',
    iot: 'IoT Project',
    other: 'Lainnya',
  };

  return (
    <Card className="border-2 p-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <History className="h-5 w-5 text-primary" />
          <h3 className="font-semibold">Riwayat Generate</h3>
          <Badge variant="secondary" className="text-xs">
            {history.length}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          {history.length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onClearAll}
              className="gap-1.5 text-xs text-muted-foreground hover:text-destructive"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Hapus Semua
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsExpanded(!isExpanded)}
            className="gap-1.5"
          >
            {isExpanded ? (
              <>
                <ChevronUp className="h-4 w-4" />
                <span className="text-xs">Sembunyikan</span>
              </>
            ) : (
              <>
                <ChevronDown className="h-4 w-4" />
                <span className="text-xs">Lihat Semua</span>
              </>
            )}
          </Button>
        </div>
      </div>

      {/* History List */}
      {isExpanded && (
        <div className="mt-4 space-y-2">
          {history.map((item) => (
            <div
              key={item.id}
              className="group rounded-lg border bg-muted/30 p-3 transition-colors hover:bg-muted/50"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 space-y-2">
                  {/* Timestamp */}
                  <p className="text-xs text-muted-foreground">
                    {formatDate(item.timestamp)}
                  </p>

                  {/* Info */}
                  <div className="flex flex-wrap items-center gap-2 text-sm">
                    <span className="font-medium">{item.formData.jurusan}</span>
                    <span className="text-muted-foreground">•</span>
                    <span className="text-muted-foreground">
                      {PROJECT_TYPES[item.formData.project_type]}
                    </span>
                    <span className="text-muted-foreground">•</span>
                    <span className="text-muted-foreground">
                      {item.ideas.length} ide
                    </span>
                  </div>

                  {/* Interests */}
                  <div className="flex flex-wrap gap-1.5">
                    {item.formData.interests.slice(0, 3).map((interest, i) => (
                      <Badge key={i} variant="outline" className="text-xs">
                        {interest}
                      </Badge>
                    ))}
                    {item.formData.interests.length > 3 && (
                      <Badge variant="outline" className="text-xs">
                        +{item.formData.interests.length - 3}
                      </Badge>
                    )}
                  </div>
                </div>

                {/* Actions */}
                <div className="flex shrink-0 gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onLoadHistory(item)}
                    className="h-8 gap-1.5 opacity-0 transition-opacity group-hover:opacity-100"
                  >
                    <Eye className="h-3.5 w-3.5" />
                    <span className="text-xs">Lihat</span>
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => onDeleteHistory(item.id)}
                    className="h-8 w-8 opacity-0 transition-opacity hover:text-destructive group-hover:opacity-100"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
