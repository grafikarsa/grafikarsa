'use client';

import { useState, useRef } from 'react';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2, Send, Paperclip, X, FileText, Image as ImageIcon, Bug, Lightbulb, MessageSquare } from 'lucide-react';
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
import { uploadsApi } from '@/lib/api/admin';
import { cn } from '@/lib/utils';

interface FeedbackFormProps {
    onSuccess?: () => void;
    onCancel?: () => void;
}

export function FeedbackForm({ onSuccess, onCancel }: FeedbackFormProps) {
    const [kategori, setKategori] = useState<FeedbackKategori>('saran');
    const [pesan, setPesan] = useState('');
    const [attachmentFile, setAttachmentFile] = useState<File | null>(null);
    const [attachmentUrl, setAttachmentUrl] = useState<string | null>(null);
    const [uploading, setUploading] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const mutation = useMutation({
        mutationFn: feedbackApi.createFeedback,
        onSuccess: () => {
            toast.success('Terima kasih atas masukanmu!');
            setPesan('');
            setKategori('saran');
            setAttachmentFile(null);
            setAttachmentUrl(null);
            onSuccess?.();
        },
        onError: (error: any) => {
            const msg = error?.response?.data?.error?.message || 'Gagal mengirim feedback';
            toast.error(msg);
        },
    });

    const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        // Validate file size (5MB)
        if (file.size > 5 * 1024 * 1024) {
            toast.error('Ukuran file maksimal 5MB');
            return;
        }

        // Validate file type
        const allowedTypes = [
            'image/jpeg', 'image/png', 'image/webp', 'image/gif',
            'application/pdf',
            'application/msword',
            'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
        ];
        if (!allowedTypes.includes(file.type)) {
            toast.error('Tipe file tidak didukung. Gunakan gambar (JPG, PNG, WebP, GIF), PDF, atau DOC');
            return;
        }

        setAttachmentFile(file);

        // Upload immediately
        setUploading(true);
        try {
            const url = await uploadsApi.uploadFile(file, 'feedback_attachment');
            setAttachmentUrl(url);
            toast.success('File berhasil diupload');
        } catch (error) {
            toast.error('Gagal upload file');
            setAttachmentFile(null);
        } finally {
            setUploading(false);
        }
    };

    const handleRemoveAttachment = () => {
        setAttachmentFile(null);
        setAttachmentUrl(null);
        if (fileInputRef.current) {
            fileInputRef.current.value = '';
        }
    };

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (pesan.length < 10) {
            toast.error('Pesan minimal 10 karakter');
            return;
        }
        mutation.mutate({ 
            kategori, 
            pesan,
            attachment_url: attachmentUrl || undefined
        });
    };

    const getFileIcon = (file: File) => {
        if (file.type.startsWith('image/')) {
            return <ImageIcon className="h-4 w-4" />;
        }
        return <FileText className="h-4 w-4" />;
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-5 py-2">
            <div className="space-y-2">
                <Label htmlFor="kategori">Kategori</Label>
                <Select
                    value={kategori}
                    onValueChange={(v) => setKategori(v as FeedbackKategori)}
                >
                    <SelectTrigger id="kategori" className="h-11">
                        <SelectValue placeholder="Pilih kategori" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="bug" className="cursor-pointer">
                            <div className="flex items-center gap-2">
                                <Bug className="h-4 w-4 text-red-500" />
                                <span>Lapor Bug</span>
                            </div>
                        </SelectItem>
                        <SelectItem value="saran" className="cursor-pointer">
                            <div className="flex items-center gap-2">
                                <Lightbulb className="h-4 w-4 text-amber-500" />
                                <span>Saran Fitur</span>
                            </div>
                        </SelectItem>
                        <SelectItem value="lainnya" className="cursor-pointer">
                            <div className="flex items-center gap-2">
                                <MessageSquare className="h-4 w-4 text-blue-500" />
                                <span>Lainnya</span>
                            </div>
                        </SelectItem>
                    </SelectContent>
                </Select>
            </div>
            <div className="space-y-2">
                <Label htmlFor="pesan">Pesan</Label>
                <Textarea
                    id="pesan"
                    placeholder="Ceritakan detail masukanmu..."
                    rows={5}
                    value={pesan}
                    onChange={(e) => setPesan(e.target.value)}
                    className="resize-none whitespace-pre-wrap"
                />
                <div className="flex items-center justify-between">
                    <p className="text-xs text-muted-foreground">
                        Minimal 10 karakter
                    </p>
                    <p className={cn(
                        "text-xs transition-colors",
                        pesan.length < 10 ? "text-muted-foreground" : 
                        pesan.length > 1900 ? "text-amber-600" : 
                        "text-green-600"
                    )}>
                        {pesan.length}/2000
                    </p>
                </div>
            </div>

            {/* Attachment Section */}
            <div className="space-y-2">
                <Label className="flex items-center gap-2">
                    <Paperclip className="h-3.5 w-3.5" />
                    Lampiran (Opsional)
                </Label>
                <div className="flex items-center gap-2">
                    <input
                        ref={fileInputRef}
                        type="file"
                        accept="image/*,.pdf,.doc,.docx"
                        onChange={handleFileSelect}
                        className="hidden"
                        disabled={uploading || !!attachmentFile}
                    />
                    {!attachmentFile ? (
                        <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => fileInputRef.current?.click()}
                            disabled={uploading}
                            className="w-full justify-center"
                        >
                            {uploading ? (
                                <>
                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                    Mengupload...
                                </>
                            ) : (
                                <>
                                    <Paperclip className="mr-2 h-4 w-4" />
                                    Pilih File
                                </>
                            )}
                        </Button>
                    ) : (
                        <div className="flex w-full items-center gap-2 rounded-md border border-border bg-muted px-3 py-2.5 text-sm">
                            {getFileIcon(attachmentFile)}
                            <span className="flex-1 truncate font-medium">{attachmentFile.name}</span>
                            <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6 shrink-0"
                                onClick={handleRemoveAttachment}
                            >
                                <X className="h-3.5 w-3.5" />
                            </Button>
                        </div>
                    )}
                </div>
                <p className="text-xs text-muted-foreground">
                    Screenshot, PDF, atau DOC (maks. 5MB)
                </p>
            </div>

            <div className="flex justify-end gap-2 pt-4 border-t">
                {onCancel && (
                    <Button type="button" variant="ghost" onClick={onCancel}>
                        Batal
                    </Button>
                )}
                <Button 
                    type="submit" 
                    disabled={mutation.isPending || pesan.length < 10 || uploading}
                    className="min-w-[100px]"
                >
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
