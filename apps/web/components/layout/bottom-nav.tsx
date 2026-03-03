'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { Home, Search, PlusSquare, User, History, Users, Image as ImageIcon, X, MessageSquare } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/lib/stores/auth-store';
import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Button } from '@/components/ui/button';

export function BottomNav() {
    const pathname = usePathname();
    const router = useRouter();
    const { user } = useAuthStore();
    const [isSearchOpen, setIsSearchOpen] = useState(false);

    if (!user) return null;

    const isActive = (path: string) => pathname === path || pathname.startsWith(path + '/');

    const handleSearchSelect = (type: 'users' | 'portfolios') => {
        setIsSearchOpen(false);
        router.push(`/${type}`);
    };

    return (
        <>
            <div className="fixed bottom-0 left-0 right-0 z-[100] border-t bg-background px-2 pb-safe-bottom pt-2 md:hidden">
                <nav className="flex items-center justify-around">
                    <Link
                        href="/"
                        className={cn(
                            "flex flex-col items-center gap-0.5 p-2 min-w-[56px]",
                            isActive('/') && pathname === '/' ? "text-primary" : "text-muted-foreground"
                        )}
                    >
                        <Home className="h-5 w-5" />
                        <span className="text-[10px]">Home</span>
                    </Link>

                    {/* <Link
                        href="/messages"
                        className={cn(
                            "flex flex-col items-center gap-0.5 p-2 min-w-[56px]",
                            isActive('/messages') ? "text-primary" : "text-muted-foreground"
                        )}
                    >
                        <MessageSquare className="h-5 w-5" />
                        <span className="text-[10px]">Pesan</span>
                    </Link> */}

                    <button
                        onClick={() => setIsSearchOpen(true)}
                        className={cn(
                            "flex flex-col items-center gap-0.5 p-2 min-w-[56px] focus:outline-none",
                            isSearchOpen || isActive('/portfolios') || isActive('/users') ? "text-primary" : "text-muted-foreground"
                        )}
                    >
                        <Search className="h-5 w-5" />
                        <span className="text-[10px]">Cari</span>
                    </button>

                    {/* Upload FAB-like item */}
                    <Link
                        href={`/${user.username}/portfolios/new`}
                        className="flex flex-col items-center gap-0.5 -mt-3 min-w-[56px]"
                    >
                        <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg ring-2 ring-background">
                            <PlusSquare className="h-5 w-5" />
                        </div>
                        <span className="text-[10px] font-medium">Upload</span>
                    </Link>

                    <Link
                        href={`/${user.username}`}
                        className={cn(
                            "flex flex-col items-center gap-0.5 p-2 min-w-[56px]",
                            isActive(`/${user.username}`) ? "text-primary" : "text-muted-foreground"
                        )}
                    >
                        <User className="h-5 w-5" />
                        <span className="text-[10px]">Profil</span>
                    </Link>

                    <Link
                        href="/changelog"
                        className={cn(
                            "flex flex-col items-center gap-0.5 p-2 min-w-[56px]",
                            isActive('/changelog') ? "text-primary" : "text-muted-foreground"
                        )}
                    >
                        <History className="h-5 w-5" />
                        <span className="text-[10px]">Updates</span>
                    </Link>
                </nav>
            </div>

            {/* Search Drawer Overlay */}
            <AnimatePresence>
                {isSearchOpen && (
                    <>
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            onClick={() => setIsSearchOpen(false)}
                            className="fixed inset-0 z-[110] bg-black/60 backdrop-blur-sm md:hidden"
                        />
                        <motion.div
                            initial={{ y: "100%" }}
                            animate={{ y: 0 }}
                            exit={{ y: "100%" }}
                            transition={{ type: "spring", damping: 25, stiffness: 300 }}
                            className="fixed bottom-0 left-0 right-0 z-[120] rounded-t-[20px] bg-background p-6 md:hidden"
                            drag="y"
                            dragConstraints={{ top: 0 }}
                            dragElastic={0.2}
                            onDragEnd={(_, info) => {
                                if (info.offset.y > 100) setIsSearchOpen(false);
                            }}
                        >
                            {/* Drag Handle */}
                            <div className="mx-auto mb-6 h-1.5 w-12 rounded-full bg-muted" />

                            <div className="mb-6 flex items-center justify-between">
                                <h3 className="text-lg font-semibold">Cari Apa?</h3>
                                <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full" onClick={() => setIsSearchOpen(false)}>
                                    <X className="h-4 w-4" />
                                </Button>
                            </div>

                            <div className="grid gap-3">
                                <button
                                    onClick={() => handleSearchSelect('users')}
                                    className="flex w-full items-center gap-4 rounded-xl border bg-card p-4 text-left transition-colors hover:bg-accent active:scale-[0.98]"
                                >
                                    <div className="rounded-full bg-blue-100 p-3 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">
                                        <Users className="h-6 w-6" />
                                    </div>
                                    <div>
                                        <div className="font-semibold">Cari User</div>
                                        <div className="text-xs text-muted-foreground">Temukan kreator berbakat</div>
                                    </div>
                                </button>

                                <button
                                    onClick={() => handleSearchSelect('portfolios')}
                                    className="flex w-full items-center gap-4 rounded-xl border bg-card p-4 text-left transition-colors hover:bg-accent active:scale-[0.98]"
                                >
                                    <div className="rounded-full bg-purple-100 p-3 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400">
                                        <ImageIcon className="h-6 w-6" />
                                    </div>
                                    <div>
                                        <div className="font-semibold">Cari Portfolio</div>
                                        <div className="text-xs text-muted-foreground">Jelajahi karya inspiratif</div>
                                    </div>
                                </button>
                            </div>

                            {/* Bottom safe area spacer */}
                            <div className="h-6" />
                        </motion.div>
                    </>
                )}
            </AnimatePresence>
        </>
    );
}
