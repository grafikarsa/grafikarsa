'use client';

import { useState } from 'react';
import { Info, X, Trophy, TrendingUp, Users, Star, ThumbsUp, Activity, Sparkles, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog';
import {
    Drawer,
    DrawerClose,
    DrawerContent,
    DrawerDescription,
    DrawerFooter,
    DrawerHeader,
    DrawerTitle,
    DrawerTrigger,
} from '@/components/ui/drawer';
import { useMediaQuery } from '@/lib/hooks/use-media-query'; // You might need to check if this hook exists, if not use a simple implementation
import { cn } from '@/lib/utils';
import { Separator } from '@/components/ui/separator';

interface RankingInfoProps {
    type: 'students' | 'projects' | 'all';
    triggerClassName?: string;
    variant?: 'default' | 'ghost' | 'outline' | 'secondary';
    side?: 'left' | 'right' | 'top' | 'bottom';
}

export function RankingInfo({ type, triggerClassName, variant = 'ghost' }: RankingInfoProps) {
    const [open, setOpen] = useState(false);
    const isDesktop = useMediaQuery('(min-width: 768px)');

    const StudentRankingContent = () => (
        <div className="space-y-4">
            <div className="flex items-center gap-2">
                <Trophy className="h-5 w-5" />
                <h3 className="font-semibold">Top Students Algorithm</h3>
            </div>
            <p className="text-sm text-muted-foreground">
                Siswa diperingkat berdasarkan skor meritokratis yang menghargai konsistensi dan kualitas.
            </p>

            <div className="space-y-3">
                <ScoreItem
                    icon={<Activity className="h-4 w-4" />}
                    label="Produktivitas"
                    value="30%"
                    desc="Jumlah portfolio yang dipublikasikan. Konsistensi berkarya sangat dihargai."
                    color="bg-blue-500"
                />
                <ScoreItem
                    icon={<Star className="h-4 w-4" />}
                    label="Kualitas (Nilai Guru)"
                    value="30%"
                    desc="Rata-rata nilai assessment dari guru. Kualitas teknis & artistik adalah kunci."
                    color="bg-purple-500"
                />
                <ScoreItem
                    icon={<ThumbsUp className="h-4 w-4" />}
                    label="Apresiasi Publik"
                    value="25%"
                    desc="Total likes yang didapatkan dari seluruh karya."
                    color="bg-pink-500"
                />
                <ScoreItem
                    icon={<Users className="h-4 w-4" />}
                    label="Pengaruh (Followers)"
                    value="15%"
                    desc="Jumlah pengikut yang terinspirasi oleh karyamu."
                    color="bg-indigo-500"
                />
            </div>
        </div>
    );

    const TrendingProjectContent = () => (
        <div className="space-y-4">
            <div className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5" />
                <h3 className="font-semibold">Trending Projects Algorithm</h3>
            </div>
            <p className="text-sm text-muted-foreground">
                Project ditampilkan berdasarkan kombinasi kualitas karya, popularitas, dan faktor kebaruan.
            </p>

            <div className="space-y-3">
                <ScoreItem
                    icon={<Star className="h-4 w-4" />}
                    label="Kualitas (Nilai Guru)"
                    value="50%"
                    desc="Faktor terpenting. Penilaian profesional guru terhadap teknis & estetika."
                    color="bg-purple-500"
                />
                <ScoreItem
                    icon={<ThumbsUp className="h-4 w-4" />}
                    label="Popularitas (Likes)"
                    value="30%"
                    desc="Seberapa banyak komunitas menyukai karya ini."
                    color="bg-pink-500"
                />
                <ScoreItem
                    icon={<Activity className="h-4 w-4" />}
                    label="Kesegaran (Recency)"
                    value="20%"
                    desc="Karya baru (< 7 hari) mendapat boost nilai agar feed selalu segar."
                    color="bg-green-500"
                />
            </div>
        </div>
    );

    const SmartFeedContent = () => (
        <div className="space-y-4">
            <div className="flex items-center gap-2">
                <Sparkles className="h-5 w-5" />
                <h3 className="font-semibold">FYP (For You Page) Algorithm</h3>
            </div>
            <p className="text-sm text-muted-foreground">
                Feed personal yang dikurasi khusus untukmu berdasarkan interaksi dan minat.
            </p>

            <div className="space-y-3">
                <ScoreItem
                    icon={<Users className="h-4 w-4" />}
                    label="Koneksi (Following)"
                    value="30%"
                    desc="Prioritas konten dari teman yang kamu follow (Mutual Follow dapat nilai lebih)."
                    color="bg-blue-500"
                />
                <ScoreItem
                    icon={<Clock className="h-4 w-4" />}
                    label="Kesegaran (Recency)"
                    value="25%"
                    desc="Konten terbaru (terutama < 24 jam) mendapat prioritas tinggi."
                    color="bg-green-500"
                />
                <ScoreItem
                    icon={<ThumbsUp className="h-4 w-4" />}
                    label="Interaksi (Engagement)"
                    value="20%"
                    desc="Kombinasi antara Likes (60%) dan Views (40%)."
                    color="bg-pink-500"
                />
                <ScoreItem
                    icon={<Activity className="h-4 w-4" />}
                    label="Relevansi (Minat)"
                    value="15%"
                    desc="Kesesuaian dengan Jurusanmu dan Tags yang sering kamu like."
                    color="bg-amber-500"
                />
                <ScoreItem
                    icon={<Star className="h-4 w-4" />}
                    label="Kualitas"
                    value="10%"
                    desc="Kelengkapan konten & nilai guru."
                    color="bg-purple-500"
                />
            </div>
        </div>
    );

    const Content = () => (
        <div className="grid gap-6 py-4">
            {/* 1. FYP Algorithm (Priority for logged in users) */}
            {type === 'all' && (
                <>
                    <SmartFeedContent />
                    <Separator />
                </>
            )}

            {/* 2. Top Students */}
            {(type === 'students' || type === 'all') && (
                <>
                    <StudentRankingContent />
                    {type === 'all' && <Separator />}
                </>
            )}

            {/* 3. Top Projects */}
            {(type === 'projects' || type === 'all') && <TrendingProjectContent />}

            <div className="rounded-lg bg-muted/50 p-3 text-xs text-muted-foreground">
                <p>
                    <strong>Note:</strong> Sistem ini didesain agar adil (fair). Anda tidak bisa menjadi Top Student hanya dengan viral (likes), tapi harus rajin upload dan menjaga kualitas nilai dari guru.
                </p>
            </div>
        </div>
    );

    if (isDesktop) {
        return (
            <Dialog open={open} onOpenChange={setOpen}>
                <DialogTrigger asChild>
                    <Button variant={variant} size="icon" className={cn("h-6 w-6 rounded-full text-muted-foreground hover:text-foreground", triggerClassName)}>
                        <Info className="h-4 w-4" />
                        <span className="sr-only">Info Ranking</span>
                    </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-[425px]">
                    <DialogHeader>
                        <DialogTitle>Bagaimana Ranking Dihitung?</DialogTitle>
                        <DialogDescription>
                            Transparansi algoritma penilaian Grafikarsa.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="max-h-[70vh] overflow-y-auto pr-2">
                        <Content />
                    </div>
                </DialogContent>
            </Dialog>
        );
    }

    return (
        <Drawer open={open} onOpenChange={setOpen} snapPoints={[0.5, 1]}>
            <DrawerTrigger asChild>
                <Button variant={variant} size="icon" className={cn("h-6 w-6 rounded-full text-muted-foreground hover:text-foreground", triggerClassName)}>
                    <Info className="h-4 w-4" />
                    <span className="sr-only">Info Ranking</span>
                </Button>
            </DrawerTrigger>
            <DrawerContent>
                <DrawerHeader className="text-left">
                    <DrawerTitle>Bagaimana Ranking Dihitung?</DrawerTitle>
                    <DrawerDescription>
                        Transparansi algoritma penilaian Grafikarsa.
                    </DrawerDescription>
                </DrawerHeader>
                <div className="px-4 overflow-y-auto max-h-[85vh]">
                    <Content />
                </div>
                <DrawerFooter className="pt-2">
                    <DrawerClose asChild>
                        <Button variant="outline">Tutup</Button>
                    </DrawerClose>
                </DrawerFooter>
            </DrawerContent>
        </Drawer>
    );
}

function ScoreItem({ icon, label, value, desc }: { icon: React.ReactNode, label: string, value: string, desc: string, color?: string }) {
    return (
        <div className="flex items-start gap-3 rounded-lg border p-3">
            <div className="relative mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
                {icon}
            </div>
            <div className="flex-1 space-y-1">
                <div className="flex items-center justify-between">
                    <span className="font-medium text-sm">{label}</span>
                    <span className="text-xs font-bold px-2 py-0.5 rounded-full bg-primary/10 text-primary">{value}</span>
                </div>
                <p className="text-xs text-muted-foreground leading-relaxed">{desc}</p>

                {/* Visual Progress Bar */}
                <div className="h-1.5 w-full rounded-full bg-muted mt-2 overflow-hidden">
                    <div className="h-full rounded-full bg-primary" style={{ width: value }}></div>
                </div>
            </div>
        </div>
    );
}
