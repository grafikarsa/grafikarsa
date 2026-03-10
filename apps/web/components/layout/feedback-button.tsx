'use client';

import { useState } from 'react';
import { usePathname } from 'next/navigation';
import { MessageCircleMore } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog';
import { useAuthStore } from '@/lib/stores/auth-store';
import { FeedbackForm } from '@/components/shared/feedback-form';

export function FeedbackButton() {
    const pathname = usePathname();
    const { isAuthenticated } = useAuthStore();
    const [open, setOpen] = useState(false);

    // Hide on auth pages
    if (pathname?.startsWith('/login') || pathname?.startsWith('/register') || pathname?.startsWith('/forgot-password')) {
        return null;
    }

    // Hide on landing page (only if not authenticated)
    if (pathname === '/' && !isAuthenticated) {
        return null;
    }

    return (
        <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
                <Button
                    variant="default"
                    size="sm"
                    className="fixed bottom-6 right-6 z-50 hidden h-10 gap-2 rounded-full px-4 shadow-lg transition-all hover:scale-105 hover:shadow-xl md:flex"
                    aria-label="Kirim Feedback"
                >
                    <MessageCircleMore className="h-4 w-4" />
                    <span className="text-sm font-medium">Feedback</span>
                </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-lg">
                <DialogHeader className="space-y-3">
                    <DialogTitle className="text-xl">Kirim Masukan</DialogTitle>
                    <DialogDescription className="text-base">
                        Bantu kami meningkatkan Grafikarsa. Laporkan bug atau berikan saran fitur baru.
                    </DialogDescription>
                </DialogHeader>
                <FeedbackForm onSuccess={() => setOpen(false)} onCancel={() => setOpen(false)} />
            </DialogContent>
        </Dialog>
    );
}
