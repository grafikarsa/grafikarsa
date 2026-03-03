'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { commentsApi } from '@/lib/api';
import { CreateCommentRequest } from '@/lib/types';
import { toast } from 'sonner';
import { Loader2, Send } from 'lucide-react';
import { useAuthStore } from '@/lib/stores/auth-store';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';

interface CommentInputProps {
    portfolioId: string;
    parentId?: string;
    onCancel?: () => void;
    onSuccess?: () => void;
    placeholder?: string;
    autoFocus?: boolean;
}

export function CommentInput({
    portfolioId,
    parentId,
    onCancel,
    onSuccess,
    placeholder = 'Tulis komentar...',
    autoFocus = false,
}: CommentInputProps) {
    const { user } = useAuthStore();
    const queryClient = useQueryClient();
    const { register, handleSubmit, reset, formState: { isSubmitting, isValid } } = useForm<{ content: string }>();

    const createCommentMutation = useMutation({
        mutationFn: (data: CreateCommentRequest) => commentsApi.create(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['comments', portfolioId] });
            reset();
            toast.success('Komentar berhasil dikirim');
            onSuccess?.();
        },
        onError: (error: any) => {
            toast.error('Gagal mengirim komentar: ' + (error.response?.data?.message || error.message));
        },
    });

    const onSubmit = (data: { content: string }) => {
        createCommentMutation.mutate({
            portfolio_id: portfolioId,
            parent_id: parentId,
            content: data.content,
        });
    };

    if (!user) return null;

    return (
        <div className="flex gap-4">
            <Avatar className="h-8 w-8 shrink-0">
                <AvatarImage src={user.avatar_url || ''} />
                <AvatarFallback>{user.nama?.[0]?.toUpperCase()}</AvatarFallback>
            </Avatar>
            <form onSubmit={handleSubmit(onSubmit)} className="flex-1 space-y-2">
                <Textarea
                    {...register('content', { required: true })}
                    placeholder={placeholder}
                    className="min-h-[80px] resize-none text-sm"
                    autoFocus={autoFocus}
                />
                <div className="flex justify-end gap-2">
                    {onCancel && (
                        <Button type="button" variant="ghost" size="sm" onClick={onCancel}>
                            Batal
                        </Button>
                    )}
                    <Button type="submit" size="sm" disabled={isSubmitting || !isValid}>
                        {createCommentMutation.isPending ? (
                            <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                        ) : (
                            <Send className="mr-2 h-3 w-3" />
                        )}
                        Kirim
                    </Button>
                </div>
            </form>
        </div>
    );
}
