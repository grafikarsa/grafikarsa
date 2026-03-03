'use client';

import { useState } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { User, generateBgColor } from '@/lib/types';
import { useAuthStore } from '@/lib/stores/auth-store';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { usersApi } from '@/lib/api';
import { toast } from 'sonner';
import {
  Github,
  Instagram,
  Linkedin,
  Twitter,
  Globe,
  Youtube,
  Facebook,
  Loader2,
  Edit,
  Share2,
  Link2,
  MessageCircle,
  Mail,
} from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { FollowModal } from './follow-modal';

const socialIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  github: Github,
  instagram: Instagram,
  linkedin: Linkedin,
  twitter: Twitter,
  personal_website: Globe,
  youtube: Youtube,
  facebook: Facebook,
};

interface UserProfileProps {
  profile: User;
}

export function UserProfile({ profile }: UserProfileProps) {
  const { user: currentUser, isAuthenticated } = useAuthStore();
  const queryClient = useQueryClient();
  const isOwner = currentUser?.username === profile.username;
  const isAdmin = currentUser?.role === 'admin';
  const [followModalType, setFollowModalType] = useState<'followers' | 'following' | null>(null);

  const followMutation = useMutation({
    mutationFn: () =>
      profile.is_following
        ? usersApi.unfollow(profile.username)
        : usersApi.follow(profile.username),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user', profile.username] });
      toast.success(profile.is_following ? 'Berhasil unfollow' : 'Berhasil follow');
    },
    onError: () => {
      toast.error('Gagal. Silakan coba lagi.');
    },
  });

  const profileUrl = typeof window !== 'undefined' ? `${window.location.origin}/${profile.username}` : '';

  const handleCopyLink = () => {
    navigator.clipboard.writeText(profileUrl);
    toast.success('Link profil berhasil disalin!');
  };

  const handleShareWhatsApp = () => {
    const text = `Halo! Cek portofolio ${profile.nama} di Grafikarsa: ${profileUrl}`;
    window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, '_blank');
  };

  const handleCopyEmail = () => {
    if (profile.email) {
      navigator.clipboard.writeText(profile.email);
      toast.success('Email berhasil disalin!');
    } else {
      toast.error('User ini tidak mencantumkan email publik');
    }
  };

  return (
    <>
      {/* Banner - Full width, edge-to-edge */}
      {/* Counteracting MainLayout padding: p-4 (mobile) and p-6 (desktop) */}
      <div className="-mx-4 -mt-4 relative w-[calc(100%+2rem)] aspect-[3/1] md:aspect-auto md:h-80 bg-gradient-to-r from-primary/20 to-primary/10 md:-mx-6 md:-mt-6 md:w-[calc(100%+3rem)]">
        {profile.banner_url && (
          <Image
            src={profile.banner_url ?? ''}
            alt="Banner"
            fill
            className="object-cover"
            priority
            unoptimized
          />
        )}
      </div>

      {/* Profile Content */}
      <div className="container mx-auto max-w-5xl px-6 pb-4 md:px-12 lg:px-16">
        {/* Avatar & Actions Row */}
        <div className="relative flex items-end justify-between">
          {/* Avatar - overlapping banner */}
          <Avatar className="-mt-10 h-20 w-20 border-4 border-background shadow-sm md:-mt-16 md:h-32 md:w-32">
            <AvatarImage src={profile.avatar_url} alt={profile.nama} />
            <AvatarFallback className="text-2xl md:text-4xl">
              {profile.nama?.charAt(0).toUpperCase()}
            </AvatarFallback>
          </Avatar>

          {/* Actions */}
          <div className="flex items-center gap-2 pb-0 md:pb-2">
            {/* Share Button - Global */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="h-8 w-8 p-0 md:h-9 md:w-9">
                  <Share2 className="h-3.5 w-3.5 md:h-4 md:w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem onClick={handleCopyLink} className="cursor-pointer">
                  <Link2 className="mr-2 h-4 w-4" />
                  Salin Link
                </DropdownMenuItem>
                <DropdownMenuItem onClick={handleCopyEmail} className="cursor-pointer">
                  <Mail className="mr-2 h-4 w-4" />
                  Salin Email
                </DropdownMenuItem>
                <DropdownMenuItem onClick={handleShareWhatsApp} className="cursor-pointer">
                  <MessageCircle className="mr-2 h-4 w-4" />
                  WhatsApp
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {isOwner ? (
              <Link href={`/${profile.username}/edit`}>
                <Button variant="outline" size="sm" className="h-8 text-xs md:h-9 md:px-4 md:text-sm">
                  <Edit className="mr-2 h-3 w-3 md:h-4 md:w-4" />
                  Edit Profil
                </Button>
              </Link>
            ) : isAuthenticated && !isAdmin ? (
              <div className="flex gap-2">
                <Button
                  variant={profile.is_following ? 'outline' : 'default'}
                  size="sm"
                  className="h-8 text-xs md:h-9 md:px-4 md:text-sm"
                  onClick={() => followMutation.mutate()}
                  disabled={followMutation.isPending}
                >
                  {followMutation.isPending && <Loader2 className="mr-2 h-3 w-3 animate-spin md:h-4 md:w-4" />}
                  {profile.is_following ? 'Unfollow' : 'Follow'}
                </Button>
                {/* <Button
                  variant="secondary"
                  size="sm"
                  className="h-8 text-xs md:h-9 md:px-4 md:text-sm active:scale-95 transition-transform"
                  onClick={async () => {
                    // Start conversation and redirect
                    try {
                      const { dmApi } = await import('@/lib/api/dm');
                      // backend version expects message content, making it optional in next step
                      await dmApi.startConversation(profile.id, 'Halo!');
                      window.location.href = '/messages';
                    } catch (e) {
                      toast.error('Gagal memulai percakapan');
                    }
                  }}
                >
                  Kirim Pesan
                </Button> */}
              </div>
            ) : null}
          </div>
        </div>

        {/* Name & Username */}
        <div className="mt-3 md:mt-4">
          <div className="flex flex-col gap-1 md:flex-row md:items-center md:gap-2">
            <h1 className="text-xl font-bold md:text-2xl">{profile.nama}</h1>
            {/* Special Role Badges */}
            {profile.special_roles && profile.special_roles.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {profile.special_roles.map((sr) => (
                  <Badge
                    key={sr.id}
                    className="px-1.5 py-0 text-[10px] font-medium md:px-2 md:py-0.5 md:text-xs"
                    style={{
                      backgroundColor: generateBgColor(sr.color),
                      color: sr.color,
                      borderColor: generateBgColor(sr.color, 0.3),
                    }}
                    variant="outline"
                  >
                    {sr.nama}
                  </Badge>
                ))}
              </div>
            )}
            {/* Dev Badge for testing/demo if needed, or based on role */}
            {profile.role === 'admin' && (
              <Badge variant="secondary" className="w-fit text-[10px] md:text-xs">Administrator</Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground md:text-base">@{profile.username}</p>
        </div>

        {/* Info Badges */}
        <div className="mt-3 flex flex-wrap gap-2 md:mt-3">
          <Badge variant="outline" className="capitalize text-xs md:text-sm">
            {profile.role}
          </Badge>
          {profile.kelas && <Badge variant="secondary" className="text-xs md:text-sm">{profile.kelas.nama}</Badge>}
          {profile.jurusan && <Badge variant="secondary" className="text-xs md:text-sm">{profile.jurusan.nama}</Badge>}
          {profile.tahun_masuk && (
            <Badge variant="outline" className="text-xs md:text-sm">
              {profile.tahun_masuk} - {profile.tahun_lulus || 'Sekarang'}
            </Badge>
          )}
        </div>

        {/* Stats */}
        <div className="mt-4 flex gap-4 text-xs md:mt-5 md:gap-6 md:text-sm">
          <button
            onClick={() => setFollowModalType('followers')}
            className="hover:underline"
          >
            <span className="font-bold text-foreground">{profile.follower_count || 0}</span>{' '}
            <span className="text-muted-foreground">Followers</span>
          </button>
          <button
            onClick={() => setFollowModalType('following')}
            className="hover:underline"
          >
            <span className="font-bold text-foreground">{profile.following_count || 0}</span>{' '}
            <span className="text-muted-foreground">Following</span>
          </button>
          <span>
            <span className="font-bold text-foreground">{profile.portfolio_count || 0}</span>{' '}
            <span className="text-muted-foreground">Portofolio</span>
          </span>
        </div>

        {/* Bio */}
        {profile.bio && <p className="mt-3 max-w-3xl text-sm leading-relaxed text-muted-foreground md:mt-4 md:text-sm">{profile.bio}</p>}

        {/* Social Links */}
        {profile.social_links && profile.social_links.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2 md:mt-4">
            {profile.social_links.map((link) => {
              const Icon = socialIcons[link.platform] || Globe;
              return (
                <a
                  key={link.platform}
                  href={link.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="rounded-full p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                  title={link.platform}
                >
                  <Icon className="h-4 w-4 md:h-5 md:w-5" />
                </a>
              );
            })}
          </div>
        )}

        {/* Follow Modal */}
        <FollowModal
          username={profile.username}
          type={followModalType || 'followers'}
          open={followModalType !== null}
          onOpenChange={(open) => !open && setFollowModalType(null)}
        />

        {/* Divider */}
        <div className="mb-6 mt-6 border-t md:mb-8 md:mt-8" />
      </div>
    </>
  );
}
