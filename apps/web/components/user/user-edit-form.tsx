'use client';

import { useState, useRef, useEffect } from 'react';
import Image from 'next/image';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2, Check, X, Camera, Eye, EyeOff, ChevronDown, Link as LinkIcon, Lock, User as UserIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Separator } from '@/components/ui/separator';
import { User, SocialLink, SocialPlatform } from '@/lib/types';
import { profileApi } from '@/lib/api';
import { uploadsApi } from '@/lib/api/admin';
import { useAuthStore } from '@/lib/stores/auth-store';
import { useDebounce } from '@/lib/hooks/use-debounce';
import { cn } from '@/lib/utils';
import { ImageCropper } from '@/components/common/image-cropper';

const profileSchema = z.object({
  nama: z.string().min(2, 'Nama minimal 2 karakter').max(100),
  username: z.string().min(3, 'Username minimal 3 karakter').max(30).regex(/^[a-z0-9_]+$/, 'Username hanya boleh huruf kecil, angka, dan underscore'),
  bio: z.string().max(500, 'Bio maksimal 500 karakter').optional(),
  email: z.string().email('Email tidak valid').optional(),
});

const passwordSchema = z.object({
  current_password: z.string().min(1, 'Password saat ini wajib diisi'),
  new_password: z.string().min(8, 'Password minimal 8 karakter'),
  new_password_confirmation: z.string(),
}).refine((data) => data.new_password === data.new_password_confirmation, {
  message: 'Konfirmasi password tidak cocok',
  path: ['new_password_confirmation'],
});

type ProfileFormData = z.infer<typeof profileSchema>;
type PasswordFormData = z.infer<typeof passwordSchema>;

const socialPlatforms: { value: SocialPlatform; label: string; placeholder: string }[] = [
  { value: 'instagram', label: 'Instagram', placeholder: 'https://instagram.com/username' },
  { value: 'github', label: 'GitHub', placeholder: 'https://github.com/username' },
  { value: 'linkedin', label: 'LinkedIn', placeholder: 'https://linkedin.com/in/username' },
  { value: 'twitter', label: 'Twitter/X', placeholder: 'https://x.com/username' },
  { value: 'youtube', label: 'YouTube', placeholder: 'https://youtube.com/@channel' },
  { value: 'tiktok', label: 'TikTok', placeholder: 'https://tiktok.com/@username' },
  { value: 'behance', label: 'Behance', placeholder: 'https://behance.net/username' },
  { value: 'dribbble', label: 'Dribbble', placeholder: 'https://dribbble.com/username' },
  { value: 'personal_website', label: 'Website', placeholder: 'https://yourwebsite.com' },
];

interface UserEditFormProps {
  user: User;
}

export function UserEditForm({ user }: UserEditFormProps) {
  const queryClient = useQueryClient();
  const { setUser, user: authUser } = useAuthStore();
  const [socialLinks, setSocialLinks] = useState<SocialLink[]>(user.social_links || []);
  const [usernameAvailable, setUsernameAvailable] = useState<boolean | null>(null);
  const [checkingUsername, setCheckingUsername] = useState(false);
  const [socialLinksOpen, setSocialLinksOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [showCurrentPassword, setShowCurrentPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  // Avatar & Banner state
  const [avatarUrl, setAvatarUrl] = useState<string | null>(user.avatar_url || null);
  const [bannerUrl, setBannerUrl] = useState<string | null>(user.banner_url || null);
  const [avatarUploading, setAvatarUploading] = useState(false);
  const [bannerUploading, setBannerUploading] = useState(false);
  const avatarInputRef = useRef<HTMLInputElement>(null);
  const bannerInputRef = useRef<HTMLInputElement>(null);

  // Cropper state
  const [cropperOpen, setCropperOpen] = useState(false);
  const [cropperImage, setCropperImage] = useState<string | null>(null);
  const [cropType, setCropType] = useState<'avatar' | 'banner'>('avatar');
  const [pendingFile, setPendingFile] = useState<File | null>(null); // Keep reference strictly for extension check if needed

  const profileForm = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      nama: user.nama,
      username: user.username,
      bio: user.bio || '',
      email: user.email || '',
    },
  });

  const passwordForm = useForm<PasswordFormData>({
    resolver: zodResolver(passwordSchema),
    defaultValues: {
      current_password: '',
      new_password: '',
      new_password_confirmation: '',
    },
  });

  const watchedUsername = profileForm.watch('username');
  const debouncedUsername = useDebounce(watchedUsername, 500);
  const watchedBio = profileForm.watch('bio');

  useEffect(() => {
    const checkUsername = async () => {
      if (!debouncedUsername || debouncedUsername === user.username) {
        setUsernameAvailable(null);
        return;
      }
      setCheckingUsername(true);
      try {
        const response = await profileApi.checkUsername(debouncedUsername);
        setUsernameAvailable(response.data?.available ?? false);
      } catch {
        setUsernameAvailable(false);
      } finally {
        setCheckingUsername(false);
      }
    };
    checkUsername();
  }, [debouncedUsername, user.username]);

  const processFileSort = (file: File, type: 'avatar' | 'banner') => {
    const allowedTypes = type === 'banner'
      ? ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
      : ['image/jpeg', 'image/png', 'image/webp'];

    if (!allowedTypes.includes(file.type)) {
      toast.error(type === 'banner'
        ? 'Format file harus JPG, PNG, WebP, atau GIF'
        : 'Format file harus JPG, PNG, atau WebP');
      return;
    }

    const maxSize = type === 'banner' ? 10 * 1024 * 1024 : 10 * 1024 * 1024;
    if (file.size > maxSize) {
      toast.error('Ukuran file terlalu besar (max 10MB)');
      return;
    }

    // Bypass cropping for GIFs to preserve animation
    if (file.type === 'image/gif') {
      handleCropComplete(file, type);
      return;
    }

    const reader = new FileReader();
    reader.addEventListener('load', () => {
      setCropperImage(reader.result?.toString() || null);
      setCropType(type);
      setPendingFile(file);
      setCropperOpen(true);
    });
    reader.readAsDataURL(file);
  };

  const handleAvatarSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) processFileSort(file, 'avatar');
    // Reset input
    if (e.target) e.target.value = '';
  };

  const handleBannerSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) processFileSort(file, 'banner');
    if (e.target) e.target.value = '';
  };

  const handleCropComplete = async (croppedFile: File, typeOverride?: 'avatar' | 'banner') => {
    const type = typeOverride || cropType;

    if (type === 'avatar') {
      if (croppedFile.size > 2 * 1024 * 1024) {
        toast.error('File terlalu besar (>2MB).');
        return;
      }
      setAvatarUploading(true);
      try {
        const url = await uploadsApi.uploadFile(croppedFile, 'avatar');
        setAvatarUrl(url);
        if (authUser) setUser({ ...authUser, avatar_url: url });
        queryClient.invalidateQueries({ queryKey: ['user', user.username] });
        toast.success('Foto profil berhasil diperbarui');
      } catch (error) {
        console.error(error);
        toast.error('Gagal upload foto profil');
      } finally {
        setAvatarUploading(false);
      }
    } else {
      if (croppedFile.size > 10 * 1024 * 1024) { // Increased limit for banner/GIF
        toast.error('File terlalu besar (>10MB).');
        return;
      }
      setBannerUploading(true);
      try {
        const url = await uploadsApi.uploadFile(croppedFile, 'banner');
        setBannerUrl(url);
        if (authUser) setUser({ ...authUser, banner_url: url });
        queryClient.invalidateQueries({ queryKey: ['user', user.username] });
        toast.success('Banner berhasil diperbarui');
      } catch (error) {
        console.error(error);
        toast.error('Gagal upload banner');
      } finally {
        setBannerUploading(false);
      }
    }
    setCropperOpen(false);
    setCropperImage(null);
  };

  const updateProfileMutation = useMutation({
    mutationFn: (data: ProfileFormData) => profileApi.updateMe(data),
    onSuccess: (response) => {
      if (response.data) {
        setUser({ ...authUser, ...response.data } as User);
        queryClient.invalidateQueries({ queryKey: ['user', user.username] });
        queryClient.invalidateQueries({ queryKey: ['profile', 'me'] });
        toast.success('Profil berhasil diperbarui');
      }
    },
    onError: () => {
      toast.error('Gagal memperbarui profil');
    },
  });

  const updatePasswordMutation = useMutation({
    mutationFn: (data: PasswordFormData) => profileApi.updatePassword(data),
    onSuccess: () => {
      toast.success('Password berhasil diperbarui');
      passwordForm.reset();
      setPasswordOpen(false);
    },
    onError: () => {
      toast.error('Gagal memperbarui password. Pastikan password lama benar.');
    },
  });

  const updateSocialLinksMutation = useMutation({
    mutationFn: (links: SocialLink[]) => profileApi.updateSocialLinks(links),
    onSuccess: (response) => {
      if (response.data) {
        setSocialLinks(response.data.social_links);
        queryClient.invalidateQueries({ queryKey: ['user', user.username] });
        toast.success('Social links berhasil diperbarui');
      }
    },
    onError: () => {
      toast.error('Gagal memperbarui social links');
    },
  });

  const handleSocialLinkChange = (platform: SocialPlatform, url: string) => {
    setSocialLinks((prev) => {
      const existing = prev.find((l) => l.platform === platform);
      if (existing) {
        if (!url) return prev.filter((l) => l.platform !== platform);
        return prev.map((l) => (l.platform === platform ? { ...l, url } : l));
      }
      if (url) return [...prev, { platform, url }];
      return prev;
    });
  };

  const filledSocialLinksCount = socialLinks.filter(l => l.url).length;

  return (
    <div className="space-y-8">
      {/* Hero Section - Banner & Avatar Preview */}
      <div className="relative overflow-hidden rounded-xl border bg-card">
        {/* Banner */}
        <div
          className="relative h-36 cursor-pointer bg-gradient-to-r from-primary/20 via-primary/10 to-primary/5 sm:h-44"
          onClick={() => !bannerUploading && bannerInputRef.current?.click()}
        >
          {bannerUrl ? (
            <Image src={bannerUrl} alt="Banner" fill className="object-cover" unoptimized />
          ) : null}
          {/* Banner Overlay */}
          <div className="absolute inset-0 flex items-center justify-center bg-black/0 transition-all hover:bg-black/40">
            <div className="flex items-center gap-2 rounded-full bg-black/60 px-4 py-2 text-sm text-white opacity-0 transition-opacity hover:opacity-100 group-hover:opacity-100 [div:hover>&]:opacity-100">
              {bannerUploading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Camera className="h-4 w-4" />
              )}
              <span>{bannerUploading ? 'Mengupload...' : 'Ubah Banner'}</span>
            </div>
          </div>
          <input
            ref={bannerInputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp,image/gif"
            className="hidden"
            onChange={handleBannerSelect}
            disabled={bannerUploading}
          />
        </div>

        {/* Avatar & Quick Info */}
        <div className="relative px-4 pb-4 sm:px-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:gap-6">
            {/* Avatar */}
            <div
              className="-mt-12 relative cursor-pointer sm:-mt-14"
              onClick={() => !avatarUploading && avatarInputRef.current?.click()}
            >
              <Avatar className="h-24 w-24 border-4 border-card sm:h-28 sm:w-28">
                <AvatarImage src={avatarUrl || undefined} alt={user.nama} />
                <AvatarFallback className="text-2xl sm:text-3xl">{user.nama?.charAt(0).toUpperCase()}</AvatarFallback>
              </Avatar>
              {/* Avatar Overlay */}
              <div className="absolute inset-0 flex items-center justify-center rounded-full bg-black/0 transition-all hover:bg-black/40">
                <div className="rounded-full bg-black/60 p-2 opacity-0 transition-opacity [div:hover>&]:opacity-100">
                  {avatarUploading ? (
                    <Loader2 className="h-5 w-5 animate-spin text-white" />
                  ) : (
                    <Camera className="h-5 w-5 text-white" />
                  )}
                </div>
              </div>
              <input
                ref={avatarInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                className="hidden"
                onChange={handleAvatarSelect}
                disabled={avatarUploading}
              />
            </div>

            {/* Name Preview */}
            <div className="flex-1 pb-1">
              <p className="text-lg font-semibold sm:text-xl">{profileForm.watch('nama') || user.nama}</p>
              <p className="text-sm text-muted-foreground">@{profileForm.watch('username') || user.username}</p>
            </div>
          </div>
          <p className="mt-2 text-xs text-muted-foreground">
            Klik pada foto profil atau banner untuk mengubahnya.
          </p>
          <div className="mt-2 rounded-md bg-muted/50 p-3 text-xs text-muted-foreground">
            <span className="font-semibold text-foreground">Catatan untuk Banner GIF:</span>
            <ul className="mt-1 list-inside list-disc space-y-1">
              <li>File GIF akan diupload langsung <strong>tanpa cropping</strong> agar animasi tetap berjalan.</li>
              <li>Disarankan menggunakan resolusi <strong>1500x500 piksel</strong> untuk hasil terbaik.</li>
            </ul>
          </div>
        </div>
      </div>

      {/* Profile Information Section */}
      <section className="space-y-4">
        <div className="flex items-center gap-2">
          <UserIcon className="h-5 w-5 text-muted-foreground" />
          <h2 className="text-lg font-semibold">Informasi Profil</h2>
        </div>

        <form onSubmit={profileForm.handleSubmit((data) => updateProfileMutation.mutate(data))} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="nama">Nama Lengkap</Label>
              <Input id="nama" {...profileForm.register('nama')} placeholder="Nama lengkap kamu" />
              {profileForm.formState.errors.nama && (
                <p className="text-sm text-destructive">{profileForm.formState.errors.nama.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <div className="relative">
                <Input
                  id="username"
                  {...profileForm.register('username')}
                  className={cn(
                    usernameAvailable === true && 'border-green-500 focus-visible:ring-green-500',
                    usernameAvailable === false && 'border-destructive focus-visible:ring-destructive'
                  )}
                />
                <div className="absolute right-3 top-2.5">
                  {checkingUsername && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
                  {!checkingUsername && usernameAvailable === true && <Check className="h-4 w-4 text-green-500" />}
                  {!checkingUsername && usernameAvailable === false && <X className="h-4 w-4 text-destructive" />}
                </div>
              </div>
              {profileForm.formState.errors.username && (
                <p className="text-sm text-destructive">{profileForm.formState.errors.username.message}</p>
              )}
              {usernameAvailable === false && (
                <p className="text-sm text-destructive">Username sudah digunakan</p>
              )}
              {usernameAvailable === true && (
                <p className="text-sm text-green-600">Username tersedia</p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input id="email" type="email" {...profileForm.register('email')} placeholder="email@example.com" />
            {profileForm.formState.errors.email && (
              <p className="text-sm text-destructive">{profileForm.formState.errors.email.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="bio">Bio</Label>
              <span className="text-xs text-muted-foreground">{watchedBio?.length || 0}/500</span>
            </div>
            <Textarea
              id="bio"
              rows={3}
              {...profileForm.register('bio')}
              placeholder="Ceritakan sedikit tentang dirimu..."
              className="resize-none"
            />
            {profileForm.formState.errors.bio && (
              <p className="text-sm text-destructive">{profileForm.formState.errors.bio.message}</p>
            )}
          </div>

          <Button type="submit" disabled={updateProfileMutation.isPending || usernameAvailable === false}>
            {updateProfileMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Simpan Perubahan
          </Button>
        </form>
      </section>

      <Separator />

      {/* Social Links Section - Collapsible */}
      <Collapsible open={socialLinksOpen} onOpenChange={setSocialLinksOpen}>
        <CollapsibleTrigger asChild>
          <button className="flex w-full items-center justify-between py-2 text-left">
            <div className="flex items-center gap-2">
              <LinkIcon className="h-5 w-5 text-muted-foreground" />
              <h2 className="text-lg font-semibold">Social Links</h2>
              {filledSocialLinksCount > 0 && (
                <span className="rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
                  {filledSocialLinksCount} link
                </span>
              )}
            </div>
            <ChevronDown className={cn("h-5 w-5 text-muted-foreground transition-transform", socialLinksOpen && "rotate-180")} />
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-4">
          <p className="text-sm text-muted-foreground">
            Tambahkan link ke profil sosial media kamu agar orang lain bisa terhubung denganmu.
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            {socialPlatforms.map((platform) => (
              <div key={platform.value} className="space-y-1.5">
                <Label htmlFor={platform.value} className="text-sm">{platform.label}</Label>
                <Input
                  id={platform.value}
                  type="url"
                  placeholder={platform.placeholder}
                  value={socialLinks.find((l) => l.platform === platform.value)?.url || ''}
                  onChange={(e) => handleSocialLinkChange(platform.value, e.target.value)}
                  className="h-9 text-sm"
                />
              </div>
            ))}
          </div>
          <Button
            onClick={() => updateSocialLinksMutation.mutate(socialLinks)}
            disabled={updateSocialLinksMutation.isPending}
            variant="outline"
          >
            {updateSocialLinksMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Simpan Social Links
          </Button>
        </CollapsibleContent>
      </Collapsible>

      <Separator />

      {/* Password Section - Collapsible */}
      <Collapsible open={passwordOpen} onOpenChange={setPasswordOpen}>
        <CollapsibleTrigger asChild>
          <button className="flex w-full items-center justify-between py-2 text-left">
            <div className="flex items-center gap-2">
              <Lock className="h-5 w-5 text-muted-foreground" />
              <h2 className="text-lg font-semibold">Ubah Password</h2>
            </div>
            <ChevronDown className={cn("h-5 w-5 text-muted-foreground transition-transform", passwordOpen && "rotate-180")} />
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-4">
          <p className="text-sm text-muted-foreground">
            Pastikan password baru kamu kuat dan mudah diingat.
          </p>
          <form onSubmit={passwordForm.handleSubmit((data) => updatePasswordMutation.mutate(data))} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="current_password">Password Saat Ini</Label>
              <div className="relative">
                <Input
                  id="current_password"
                  type={showCurrentPassword ? 'text' : 'password'}
                  {...passwordForm.register('current_password')}
                />
                <button
                  type="button"
                  onClick={() => setShowCurrentPassword(!showCurrentPassword)}
                  className="absolute right-3 top-2.5 text-muted-foreground hover:text-foreground"
                >
                  {showCurrentPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {passwordForm.formState.errors.current_password && (
                <p className="text-sm text-destructive">{passwordForm.formState.errors.current_password.message}</p>
              )}
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="new_password">Password Baru</Label>
                <div className="relative">
                  <Input
                    id="new_password"
                    type={showNewPassword ? 'text' : 'password'}
                    {...passwordForm.register('new_password')}
                  />
                  <button
                    type="button"
                    onClick={() => setShowNewPassword(!showNewPassword)}
                    className="absolute right-3 top-2.5 text-muted-foreground hover:text-foreground"
                  >
                    {showNewPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                {passwordForm.formState.errors.new_password && (
                  <p className="text-sm text-destructive">{passwordForm.formState.errors.new_password.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="new_password_confirmation">Konfirmasi Password</Label>
                <div className="relative">
                  <Input
                    id="new_password_confirmation"
                    type={showConfirmPassword ? 'text' : 'password'}
                    {...passwordForm.register('new_password_confirmation')}
                  />
                  <button
                    type="button"
                    onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                    className="absolute right-3 top-2.5 text-muted-foreground hover:text-foreground"
                  >
                    {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                {passwordForm.formState.errors.new_password_confirmation && (
                  <p className="text-sm text-destructive">{passwordForm.formState.errors.new_password_confirmation.message}</p>
                )}
              </div>
            </div>
            <Button type="submit" disabled={updatePasswordMutation.isPending} variant="outline">
              {updatePasswordMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Ubah Password
            </Button>
          </form>
        </CollapsibleContent>
      </Collapsible>

      <ImageCropper
        open={cropperOpen}
        onOpenChange={setCropperOpen}
        imageSrc={cropperImage}
        aspect={cropType === 'avatar' ? 1 : 3}
        onCropComplete={handleCropComplete}
        title={cropType === 'avatar' ? 'Sesuaikan Foto Profil' : 'Sesuaikan Banner Profil'}
        description={cropType === 'avatar' ? 'Geser dan zoom untuk menyesuaikan foto profil.' : 'Geser dan zoom untuk menyesuaikan banner.'}
      />
    </div>
  );
}
