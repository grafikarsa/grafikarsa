'use client';

import { useState } from 'react';
import { ChevronLeft, ChevronRight, ThumbsUp, ThumbsDown, Save, Trash2, Clock, Target, Lightbulb, Sparkles } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import type { ProjectIdea } from '@/lib/api/ai';

interface IdeaCarouselProps {
  ideas: ProjectIdea[];
  onLike?: (index: number) => void;
  onSkip?: (index: number) => void;
  onSave?: (index: number) => void;
  onDelete?: (index: number) => void;
}

const DIFFICULTY_LEVELS = [
  { value: 'beginner', label: 'Pemula', color: 'bg-green-500/10 text-green-700 dark:text-green-400 border-green-500/20' },
  { value: 'intermediate', label: 'Menengah', color: 'bg-yellow-500/10 text-yellow-700 dark:text-yellow-400 border-yellow-500/20' },
  { value: 'advanced', label: 'Lanjutan', color: 'bg-red-500/10 text-red-700 dark:text-red-400 border-red-500/20' },
];

export function IdeaCarousel({ ideas, onLike, onSkip, onSave, onDelete }: IdeaCarouselProps) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [showDetails, setShowDetails] = useState(false);

  const currentIdea = ideas[currentIndex];

  const handlePrevious = () => {
    setCurrentIndex((prev) => (prev > 0 ? prev - 1 : ideas.length - 1));
    setShowDetails(false);
  };

  const handleNext = () => {
    setCurrentIndex((prev) => (prev < ideas.length - 1 ? prev + 1 : 0));
    setShowDetails(false);
  };

  const getDifficultyColor = (difficulty: string) => {
    return DIFFICULTY_LEVELS.find((d) => d.value === difficulty)?.color || '';
  };

  const getDifficultyLabel = (difficulty: string) => {
    return DIFFICULTY_LEVELS.find((d) => d.value === difficulty)?.label || difficulty;
  };

  if (!currentIdea) return null;

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Sparkles className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold md:text-xl">
            Ide {currentIndex + 1} dari {ideas.length}
          </h2>
        </div>
        {onDelete && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onDelete(currentIndex)}
            className="gap-2 text-muted-foreground hover:text-destructive"
          >
            <Trash2 className="h-4 w-4" />
            <span className="hidden sm:inline">Hapus</span>
          </Button>
        )}
      </div>

      {/* Main Card */}
      <Card className="overflow-hidden border-2 shadow-lg">
        <div className="p-6 md:p-8">
          {/* Title & Difficulty */}
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
            <div className="flex items-start gap-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <span className="text-lg font-bold">{currentIndex + 1}</span>
              </div>
              <h3 className="text-xl font-bold leading-tight md:text-2xl">{currentIdea.title}</h3>
            </div>
            <Badge className={cn('shrink-0 border', getDifficultyColor(currentIdea.difficulty))}>
              {getDifficultyLabel(currentIdea.difficulty)}
            </Badge>
          </div>

          {/* Description */}
          <p className="mb-4 text-sm leading-relaxed text-muted-foreground md:text-base">
            {currentIdea.description}
          </p>

          {/* Quick Info */}
          <div className="mb-4 flex flex-wrap gap-3 text-sm">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-primary" />
              <span className="font-medium">Estimasi:</span>
              <span className="text-muted-foreground">{currentIdea.estimated_time}</span>
            </div>
            <div className="flex items-center gap-2">
              <Lightbulb className="h-4 w-4 text-primary" />
              <span className="font-medium">{currentIdea.technologies.length} Teknologi</span>
            </div>
            <div className="flex items-center gap-2">
              <Target className="h-4 w-4 text-primary" />
              <span className="font-medium">{currentIdea.learning_goals.length} Tujuan</span>
            </div>
          </div>

          {/* Toggle Details */}
          <Button
            variant="outline"
            onClick={() => setShowDetails(!showDetails)}
            className="mb-4 w-full"
          >
            {showDetails ? 'Sembunyikan Detail' : 'Lihat Detail Lengkap'}
          </Button>

          {/* Detailed Info */}
          {showDetails && (
            <div className="space-y-4 border-t pt-4">
              {/* Technologies */}
              <div>
                <div className="mb-2 flex items-center gap-2 text-sm font-semibold">
                  <Lightbulb className="h-4 w-4 text-primary" />
                  <span>Teknologi yang Digunakan</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {currentIdea.technologies.map((tech, i) => (
                    <Badge key={i} variant="secondary" className="font-normal">
                      {tech}
                    </Badge>
                  ))}
                </div>
              </div>

              {/* Learning Goals */}
              <div>
                <div className="mb-2 flex items-center gap-2 text-sm font-semibold">
                  <Target className="h-4 w-4 text-primary" />
                  <span>Tujuan Pembelajaran</span>
                </div>
                <ul className="space-y-1.5 text-sm text-muted-foreground">
                  {currentIdea.learning_goals.map((goal, i) => (
                    <li key={i} className="flex gap-2">
                      <span className="text-primary">•</span>
                      <span className="flex-1">{goal}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}
        </div>

        {/* Navigation & Actions */}
        <div className="border-t bg-muted/30 p-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {/* Navigation */}
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="icon"
                onClick={handlePrevious}
                disabled={ideas.length <= 1}
                className="h-9 w-9"
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>

              <div className="flex gap-1.5 px-2">
                {ideas.map((_, index) => (
                  <button
                    key={index}
                    onClick={() => {
                      setCurrentIndex(index);
                      setShowDetails(false);
                    }}
                    className={cn(
                      'h-2 rounded-full transition-all',
                      index === currentIndex ? 'w-8 bg-primary' : 'w-2 bg-muted-foreground/30'
                    )}
                    aria-label={`Go to idea ${index + 1}`}
                  />
                ))}
              </div>

              <Button
                variant="outline"
                size="icon"
                onClick={handleNext}
                disabled={ideas.length <= 1}
                className="h-9 w-9"
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>

            {/* Action Buttons */}
            <div className="flex items-center justify-center gap-2">
              {onSkip && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onSkip(currentIndex)}
                  className="gap-1.5"
                >
                  <ThumbsDown className="h-4 w-4" />
                  <span className="hidden sm:inline">Skip</span>
                </Button>
              )}
              {onLike && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onLike(currentIndex)}
                  className="gap-1.5"
                >
                  <ThumbsUp className="h-4 w-4" />
                  <span className="hidden sm:inline">Suka</span>
                </Button>
              )}
              {onSave && (
                <Button
                  size="sm"
                  onClick={() => onSave(currentIndex)}
                  className="gap-1.5"
                >
                  <Save className="h-4 w-4" />
                  <span className="hidden sm:inline">Simpan</span>
                </Button>
              )}
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}
