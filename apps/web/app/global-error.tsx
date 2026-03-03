'use client';

import { useEffect } from 'react';
import { Button } from '@/components/ui/button';

export default function GlobalError({
    error,
    reset,
}: {
    error: Error & { digest?: string };
    reset: () => void;
}) {
    useEffect(() => {
        // Log error to console (or reporting service)
        console.error('Global Error:', error);
    }, [error]);

    return (
        <html>
            <body className="flex min-h-screen flex-col items-center justify-center p-4 text-center font-sans">
                <div className="space-y-4">
                    <h2 className="text-2xl font-bold">Terjadi Kesalahan</h2>
                    <p className="text-muted-foreground">
                        Aplikasi mengalami masalah teknis. Coba muat ulang halaman.
                    </p>
                    <div className="flex justify-center gap-4">
                        <Button
                            onClick={() => {
                                // Force hard reload to clear cache
                                window.location.href = window.location.href;
                            }}
                        >
                            Muat Ulang (Refresh)
                        </Button>
                        <Button variant="outline" onClick={() => reset()}>
                            Coba Lagi
                        </Button>
                    </div>
                </div>
            </body>
        </html>
    );
}
