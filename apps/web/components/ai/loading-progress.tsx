'use client';

import { useEffect, useState } from 'react';
import { Sparkles, Brain, Lightbulb, Rocket } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';

const LOADING_STAGES = [
  { icon: Brain, message: 'Menganalisis minat dan jurusan...', duration: 3000 },
  { icon: Lightbulb, message: 'Mencari teknologi yang sesuai...', duration: 4000 },
  { icon: Rocket, message: 'Membuat ide proyek kreatif...', duration: 5000 },
  { icon: Sparkles, message: 'Menyelesaikan detail...', duration: 3000 },
];

export function LoadingProgress() {
  const [currentStage, setCurrentStage] = useState(0);
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    const totalDuration = LOADING_STAGES.reduce((sum, stage) => sum + stage.duration, 0);
    let elapsed = 0;

    const interval = setInterval(() => {
      elapsed += 100;
      const newProgress = Math.min((elapsed / totalDuration) * 100, 95);
      setProgress(newProgress);

      // Update stage based on elapsed time
      let cumulativeDuration = 0;
      for (let i = 0; i < LOADING_STAGES.length; i++) {
        cumulativeDuration += LOADING_STAGES[i].duration;
        if (elapsed < cumulativeDuration) {
          setCurrentStage(i);
          break;
        }
      }

      if (elapsed >= totalDuration) {
        clearInterval(interval);
      }
    }, 100);

    return () => clearInterval(interval);
  }, []);

  const CurrentIcon = LOADING_STAGES[currentStage].icon;

  return (
    <Card className="border-2 p-8 md:p-12">
      <div className="mx-auto max-w-md space-y-8 text-center">
        {/* Animated Icon */}
        <div className="flex justify-center">
          <div className="relative">
            <div className="absolute inset-0 animate-ping rounded-full bg-primary/20" />
            <div className="relative flex h-20 w-20 items-center justify-center rounded-full bg-primary/10">
              <CurrentIcon className="h-10 w-10 animate-pulse text-primary" />
            </div>
          </div>
        </div>

        {/* Progress Bar */}
        <div className="space-y-3">
          <Progress value={progress} className="h-2" />
          <p className="text-sm font-medium text-muted-foreground">
            {Math.round(progress)}% selesai
          </p>
        </div>

        {/* Current Stage Message */}
        <div className="space-y-4">
          <p className="text-lg font-semibold">{LOADING_STAGES[currentStage].message}</p>

          {/* Stage Indicators */}
          <div className="flex justify-center gap-2">
            {LOADING_STAGES.map((stage, index) => {
              const Icon = stage.icon;
              return (
                <div
                  key={index}
                  className={cn(
                    'flex h-8 w-8 items-center justify-center rounded-full transition-all',
                    index < currentStage && 'bg-primary/20 text-primary',
                    index === currentStage && 'bg-primary text-primary-foreground',
                    index > currentStage && 'bg-muted text-muted-foreground'
                  )}
                >
                  <Icon className="h-4 w-4" />
                </div>
              );
            })}
          </div>
        </div>

        {/* Estimated Time */}
        <p className="text-xs text-muted-foreground">
          Estimasi waktu: 10-15 detik
        </p>
      </div>
    </Card>
  );
}
