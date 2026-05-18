'use client';

import { useState, useCallback, useEffect } from 'react';
import { authApi } from '@/lib/api';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { ShieldAlert, RefreshCw, CheckCircle2, XCircle, Loader2 } from 'lucide-react';

interface CaptchaDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onVerified: (captchaId: string, answer: number) => void;
}

export function CaptchaDialog({ open, onOpenChange, onVerified }: CaptchaDialogProps) {
  const [captchaId, setCaptchaId] = useState('');
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [feedback, setFeedback] = useState<'success' | 'error' | null>(null);

  const loadCaptcha = useCallback(async () => {
    setIsLoading(true);
    setFeedback(null);
    setAnswer('');
    try {
      const res = await authApi.getCaptcha();
      if (res.success && res.data) {
        setCaptchaId(res.data.id);
        setQuestion(res.data.question);
      }
    } catch {
      // Fallback: still allow retry
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      loadCaptcha();
    }
  }, [open, loadCaptcha]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!answer.trim() || !captchaId) return;

    setFeedback(null);
    const num = parseInt(answer, 10);
    if (isNaN(num)) return;

    onVerified(captchaId, num);
    setFeedback('success');
  };

  const handleClose = () => {
    onOpenChange(false);
    setFeedback(null);
    setAnswer('');
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md" showCloseButton={false}>
        <DialogHeader className="gap-3">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/40">
            <ShieldAlert className="h-6 w-6 text-amber-600 dark:text-amber-400" />
          </div>
          <div className="text-center">
            <DialogTitle>Verifikasi Keamanan</DialogTitle>
            <DialogDescription>
              Terlalu banyak percobaan login. Selesaikan verifikasi untuk melanjutkan.
            </DialogDescription>
          </div>
        </DialogHeader>

        {isLoading ? (
          <div className="flex flex-col items-center justify-center py-6">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <p className="mt-3 text-sm text-muted-foreground">Memuat CAPTCHA...</p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-5">
            <div className="flex flex-col items-center gap-3 rounded-lg border bg-secondary/50 p-5">
              <p className="text-2xl font-bold tracking-wider tabular-nums">
                {question}
              </p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-7 gap-1.5 text-xs"
                onClick={loadCaptcha}
                disabled={isLoading}
              >
                <RefreshCw className={cn('h-3 w-3', isLoading && 'animate-spin')} />
                Soal baru
              </Button>
            </div>

            <div className="flex items-center gap-3">
              <Input
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                placeholder="Jawaban"
                value={answer}
                onChange={(e) => {
                  setAnswer(e.target.value.replace(/[^0-9]/g, ''));
                }}
                maxLength={2}
                className={cn(
                  'h-12 text-center text-xl font-semibold tabular-nums',
                  feedback === 'error' && 'border-destructive focus-visible:ring-destructive',
                  feedback === 'success' && 'border-green-500 focus-visible:ring-green-500'
                )}
                autoFocus
              />
            </div>

            {feedback === 'error' && (
              <div className="flex items-center justify-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 py-2 text-sm text-destructive">
                <XCircle className="h-4 w-4" />
                <span>Jawaban salah. Coba lagi.</span>
              </div>
            )}
            {feedback === 'success' && (
              <div className="flex items-center justify-center gap-2 rounded-lg border border-green-500/30 bg-green-500/10 py-2 text-sm text-green-600 dark:text-green-400">
                <CheckCircle2 className="h-4 w-4" />
                <span>Verifikasi berhasil!</span>
              </div>
            )}

            <Button type="submit" className="w-full" size="lg" disabled={!answer.trim()}>
              Verifikasi &amp; Lanjutkan
            </Button>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
