'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2, Send } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { feedbackApi, FeedbackKategori } from '@/lib/api/feedback';

interface FeedbackFormProps {
    onSuccess?: () => void;
    onCancel?: () => void;
}

export function FeedbackForm({ onSuccess, onCancel }: FeedbackFormProps) {
    const [kategori, setKategori] = useState<FeedbackKategori>('saran');
    const [pesan, setPesan] = useState('');

    const mutation = useMutation({
        mutationFn: feedbackApi.createFeedback,
        onSuccess: () => {
            toast.success('Terima kasih atas masukan Anda!');
            setPesan('');
            setKategori('saran');
            onSuccess?.();
        },
        onError: (error: any) => {
            const msg = error?.response?.data?.error?.message || 'Gagal mengirim feedback';
            toast.error(msg);
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (pesan.length < 10) {
            toast.error('Pesan minimal 10 karakter');
            return;
        }
        mutation.mutate({ kategori, pesan });
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-4 py-2">
            <div className="space-y-2">
                <Label htmlFor="kategori">Kategori</Label>
                <Select
                    value={kategori}
                    onValueChange={(v) => setKategori(v as FeedbackKategori)}
                >
                    <SelectTrigger id="kategori">
                        <SelectValue placeholder="Pilih kategori" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="saran">💡 Saran Fitur</SelectItem>
                        <SelectItem value="bug">🐛 Lapor Bug</SelectItem>
                        <SelectItem value="lainnya">📝 Lainnya</SelectItem>
                    </SelectContent>
                </Select>
            </div>
            <div className="space-y-2">
                <Label htmlFor="pesan">Pesan</Label>
                <Textarea
                    id="pesan"
                    placeholder="Ceritakan detail masukan Anda..."
                    rows={4}
                    value={pesan}
                    onChange={(e) => setPesan(e.target.value)}
                    className="resize-none"
                />
                <p className="text-xs text-muted-foreground text-right">
                    {pesan.length}/2000 karakter (min. 10)
                </p>
            </div>
            <div className="flex justify-end gap-2 pt-2">
                {onCancel && (
                    <Button type="button" variant="outline" onClick={onCancel}>
                        Batal
                    </Button>
                )}
                <Button type="submit" disabled={mutation.isPending || pesan.length < 10}>
                    {mutation.isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Mengirim...
                        </>
                    ) : (
                        <>
                            <Send className="mr-2 h-4 w-4" />
                            Kirim
                        </>
                    )}
                </Button>
            </div>
        </form>
    );
}
