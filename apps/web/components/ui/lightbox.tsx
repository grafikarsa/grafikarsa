'use client';

import * as React from 'react';
import Image from 'next/image';
import { Dialog, DialogContent, DialogClose } from '@/components/ui/dialog';
import { X, ZoomIn } from 'lucide-react';

interface LightboxProps {
    src: string;
    alt: string;
    isOpen: boolean;
    onClose: () => void;
}

export function Lightbox({ src, alt, isOpen, onClose }: LightboxProps) {
    return (
        <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="max-w-screen-xl border-none bg-black/90 p-0 shadow-none sm:max-w-screen-xl">
                <div className="relative flex h-[90vh] w-full items-center justify-center p-4">
                    <DialogClose className="absolute right-4 top-4 z-50 rounded-full bg-black/50 p-2 text-white hover:bg-white/20">
                        <X className="h-6 w-6" />
                        <span className="sr-only">Close</span>
                    </DialogClose>
                    <div className="relative h-full w-full">
                        <Image
                            src={src}
                            alt={alt}
                            fill
                            className="object-contain" // Preserves aspect ratio, fits within container
                            priority
                            unoptimized
                        />
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}

export function ImageWithLightbox({ src, alt, className }: { src: string; alt: string; className?: string }) {
    const [isOpen, setIsOpen] = React.useState(false);

    return (
        <>
            <div
                className={`group relative h-full w-full cursor-zoom-in ${className || ''}`}
                onClick={() => setIsOpen(true)}
            >
                <Image
                    src={src}
                    alt={alt}
                    fill
                    className="object-contain transition-transform duration-300 group-hover:scale-105"
                />
                <div className="absolute inset-0 flex items-center justify-center bg-black/0 transition-colors group-hover:bg-black/10">
                    <ZoomIn className="h-8 w-8 text-white opacity-0 transition-opacity group-hover:opacity-100 drop-shadow-md" />
                </div>
            </div>
            <Lightbox src={src} alt={alt} isOpen={isOpen} onClose={() => setIsOpen(false)} />
        </>
    );
}
