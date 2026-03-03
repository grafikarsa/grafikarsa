'use client';

import { useState } from 'react';
import { usePathname } from 'next/navigation';
import { MessageSquarePlus } from 'lucide-react';
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
                    size="icon"
                    className="fixed bottom-6 right-6 z-50 hidden h-12 w-12 rounded-full shadow-lg transition-transform hover:scale-105 md:flex"
                    aria-label="Kirim Feedback"
                >
                    <MessageSquarePlus className="h-6 w-6" />
                </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Kirim Masukan</DialogTitle>
                    <DialogDescription>
                        Bantu kami meningkatkan Grafikarsa. Laporkan bug atau berikan saran fitur baru.
                    </DialogDescription>
                </DialogHeader>
                <FeedbackForm onSuccess={() => setOpen(false)} onCancel={() => setOpen(false)} />
            </DialogContent>
        </Dialog>
    );
}
