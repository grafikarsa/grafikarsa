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
  Loader2,
  Edit,
  Share2,
  Link2,
  MessageCircle,
  Mail,
  Phone,
  MapPin,
} from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { FollowModal } from './follow-modal';
import { SocialChip } from './social-chip';

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
      {/* Banner - Extended width with 3:1 aspect ratio */}
      <div className="w-full px-4 pt-4 md:container md:mx-auto md:max-w-6xl md:px-6 md:pt-6 lg:px-8">
        <div className="relative w-full aspect-[3/1] overflow-hidden rounded-2xl bg-gradient-to-r from-primary/20 to-primary/10">
          {profile.banner_url && (
            <Image
              src={profile.banner_url ?? ''}
              alt="Banner"
              fill
              className="object-cover object-center"
              priority
              unoptimized
            />
          )}
        </div>
      </div>

      {/* Profile Content */}
      <div className="w-full px-4 pb-4 md:container md:mx-auto md:max-w-5xl md:px-12 lg:px-16">
        {/* Avatar & Actions Row */}
        <div className="relative flex items-end justify-between -mt-12 md:-mt-20">
          {/* Avatar - overlapping banner with better positioning */}
          <Avatar className="h-24 w-24 border-4 border-background md:h-36 md:w-36 md:border-[6px]">
            <AvatarImage src={profile.avatar_url} alt={profile.nama} />
            <AvatarFallback className="text-2xl md:text-4xl">
              {profile.nama?.charAt(0).toUpperCase()}
            </AvatarFallback>
          </Avatar>

          {/* Actions */}
          <div className="flex items-center gap-2 pb-2 md:pb-4">
            {isOwner ? (
              <>
                <Link href={`/${profile.username}/edit`}>
                  <Button variant="outline" size="sm" className="h-9 text-xs md:h-10 md:px-4 md:text-sm">
                    <Edit className="mr-2 h-3.5 w-3.5 md:h-4 md:w-4" />
                    Edit Profil
                  </Button>
                </Link>
                {/* Share Button for owner */}
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="sm" className="h-9 w-9 p-0 md:h-10 md:w-10">
                      <Share2 className="h-4 w-4" />
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
                      Share ke WhatsApp
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </>
            ) : isAuthenticated && !isAdmin ? (
              <div className="flex gap-2">
                <Button
                  variant={profile.is_following ? 'outline' : 'default'}
                  size="sm"
                  className="h-9 text-xs md:h-10 md:px-4 md:text-sm"
                  onClick={() => followMutation.mutate()}
                  disabled={followMutation.isPending}
                >
                  {followMutation.isPending && <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin md:h-4 md:w-4" />}
                  {profile.is_following ? 'Unfollow' : 'Follow'}
                </Button>
                {/* Share Button for visitors */}
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="sm" className="h-9 w-9 p-0 md:h-10 md:w-10">
                      <Share2 className="h-4 w-4" />
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
                      Share ke WhatsApp
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            ) : (
              /* Share Button for non-authenticated users */
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="h-9 w-9 p-0 md:h-10 md:w-10">
                    <Share2 className="h-4 w-4" />
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
                    Share ke WhatsApp
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
        </div>

        {/* Name & Special Role Badges */}
        <div className="mt-4 md:mt-5">
          <div className="flex flex-col gap-2 md:flex-row md:items-center md:gap-3">
            <h1 className="text-2xl font-bold md:text-3xl">{profile.nama}</h1>
            {/* Special Role Badges */}
            {profile.special_roles && profile.special_roles.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {profile.special_roles.map((sr) => (
                  <Badge
                    key={sr.id}
                    className="px-2 py-0.5 text-xs font-medium md:px-2.5 md:py-1 md:text-sm"
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
            {/* Admin Badge */}
            {profile.role === 'admin' && (
              <Badge variant="secondary" className="w-fit text-xs md:text-sm">Administrator</Badge>
            )}
          </div>
        </div>

        {/* Grid 2x3 Layout */}
        <div className="mt-4 grid grid-cols-1 gap-4 md:mt-5 md:grid-cols-2 md:gap-x-16 md:gap-y-4">
          {/* Row 1 Col 1: Username */}
          <div className="flex items-center">
            <p className="text-base text-muted-foreground md:text-lg">@{profile.username}</p>
          </div>

          {/* Row 1 Col 2: Email */}
          <div className="flex items-center justify-start md:justify-end">
            {profile.show_email && profile.email ? (
              <div className="flex items-center gap-2 text-base text-muted-foreground">
                <Mail className="h-4 w-4 flex-shrink-0" />
                <a href={`mailto:${profile.email}`} className="break-all hover:text-foreground hover:underline">
                  {profile.email}
                </a>
              </div>
            ) : (
              <span className="text-base text-muted-foreground/50">-</span>
            )}
          </div>

          {/* Row 2 Col 1: Info Badges */}
          <div className="flex items-center">
            <div className="flex flex-wrap gap-2">
              <Badge variant="outline" className="capitalize">
                {profile.role}
              </Badge>
              {profile.kelas && <Badge variant="secondary">{profile.kelas.nama}</Badge>}
              {profile.jurusan && <Badge variant="secondary">{profile.jurusan.nama}</Badge>}
              {profile.tahun_masuk && (
                <Badge variant="outline">
                  {profile.tahun_masuk} - {profile.tahun_lulus || 'Sekarang'}
                </Badge>
              )}
            </div>
          </div>

          {/* Row 2 Col 2: Phone */}
          <div className="flex items-center justify-start md:justify-end">
            {profile.show_phone && profile.phone ? (
              <div className="flex items-center gap-2 text-base text-muted-foreground">
                <Phone className="h-4 w-4 flex-shrink-0" />
                <a href={`tel:${profile.phone}`} className="hover:text-foreground hover:underline">
                  {profile.phone}
                </a>
              </div>
            ) : (
              <span className="text-base text-muted-foreground/50">-</span>
            )}
          </div>

          {/* Row 3 Col 1: Stats */}
          <div className="flex items-center">
            <div className="flex gap-6 text-base">
              <button
                onClick={() => setFollowModalType('followers')}
                className="transition-colors hover:text-primary"
              >
                <span className="font-bold text-foreground">{profile.follower_count || 0}</span>{' '}
                <span className="text-muted-foreground">Followers</span>
              </button>
              <button
                onClick={() => setFollowModalType('following')}
                className="transition-colors hover:text-primary"
              >
                <span className="font-bold text-foreground">{profile.following_count || 0}</span>{' '}
                <span className="text-muted-foreground">Following</span>
              </button>
              <span>
                <span className="font-bold text-foreground">{profile.portfolio_count || 0}</span>{' '}
                <span className="text-muted-foreground">Portofolio</span>
              </span>
            </div>
          </div>

          {/* Row 3 Col 2: Address */}
          <div className="flex items-center justify-start md:justify-end">
            {profile.show_address && profile.address ? (
              <div className="flex items-center gap-2 text-base text-muted-foreground">
                <MapPin className="h-4 w-4 flex-shrink-0" />
                <span className="break-words text-right">{profile.address}</span>
              </div>
            ) : (
              <span className="text-base text-muted-foreground/50">-</span>
            )}
          </div>
        </div>

        {/* Bio */}
        {profile.bio && <p className="mt-4 max-w-3xl text-sm leading-relaxed md:mt-5 md:text-base">{profile.bio}</p>}

        {/* Social Links */}
        {profile.social_links && profile.social_links.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2 md:mt-5">
            {profile.social_links.map((link) => (
              <SocialChip key={link.platform} link={link} />
            ))}
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
        <div className="mb-8 mt-8 border-t md:mb-10 md:mt-10" />
      </div>
    </>
  );
}
