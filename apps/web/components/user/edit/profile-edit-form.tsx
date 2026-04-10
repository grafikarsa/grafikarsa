'use client';

import { useState, useRef } from 'react';
import Image from 'next/image';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2, Camera, ImageIcon, AlertCircle, ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { User } from '@/lib/types';
import { profileApi } from '@/lib/api';
import { uploadsApi } from '@/lib/api/admin';
import { useAuthStore } from '@/lib/stores/auth-store';
import { cn } from '@/lib/utils';
import { ImageCropper } from '@/components/common/image-cropper';

const profileSchema = z.object({
  nama: z.string().min(2, 'Nama minimal 2 karakter').max(100),
  bio: z.string().max(500, 'Bio maksimal 500 karakter').optional(),
});

type ProfileFormData = z.infer<typeof profileSchema>;

interface ProfileEditFormProps {
  user: User;
}

export function ProfileEditForm({ user }: ProfileEditFormProps) {
  const queryClient = useQueryClient();
  const { setUser } = useAuthStore();
  
  // Avatar & Banner state
  const [avatarUrl, setAvatarUrl] = useState<string | null>(user.avatar_url || null);
  const [bannerUrl, setBannerUrl] = useState<string | null>(user.banner_url || null);
  const [avatarUploading, setAvatarUploading] = useState(false);
  const [bannerUploading, setBannerUploading] = useState(false);
  const avatarInputRef = useRef<HTMLInputElement>(null);
  const bannerInputRef = useRef<HTMLInputElement>(null);
  const [imageGuideOpen, setImageGuideOpen] = useState(false);

  // Cropper state
  const [cropperOpen, setCropperOpen] = useState(false);
  const [cropperImage, setCropperImage] = useState<string | null>(null);
  const [cropType, setCropType] = useState<'avatar' | 'banner'>('avatar');
  const [pendingFile, setPendingFile] = useState<File | null>(null);

  const profileForm = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      nama: user.nama,
      bio: user.bio || '',
    },
  });

  const updateProfileMutation = useMutation({
    mutationFn: (data: ProfileFormData) => profileApi.updateProfile(data),
    onSuccess: (response) => {
      if (response.data) {
        setUser(response.data);
        queryClient.invalidateQueries({ queryKey: ['user', user.username] });
        queryClient.invalidateQueries({ queryKey: ['profile', 'me'] });
        toast.success('Profil berhasil diperbarui');
      }
    },
    onError: () => {
      toast.error('Gagal memperbarui profil');
    },
  });

  const handleImageSelect = (type: 'avatar' | 'banner', file: File) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      setCropperImage(e.target?.result as string);
      setCropType(type);
      setPendingFile(file);
      setCropperOpen(true);
    };
    reader.readAsDataURL(file);
  };

  const handleCropComplete = async (croppedBlob: Blob) => {
    if (!pendingFile) return;

    const extension = pendingFile.name.split('.').pop() || 'jpg';
    const croppedFile = new File([croppedBlob], `${cropType}.${extension}`, {
      type: croppedBlob.type,
    });

    if (cropType === 'avatar') {
      setAvatarUploading(true);
      try {
        const response = await uploadsApi.uploadAvatar(croppedFile);
        if (response.data?.url) {
          setAvatarUrl(response.data.url);
          await profileApi.updateProfile({ avatar_url: response.data.url });
          setUser({ ...user, avatar_url: response.data.url });
          queryClient.invalidateQueries({ queryKey: ['user', user.username] });
          queryClient.invalidateQueries({ queryKey: ['profile', 'me'] });
          toast.success('Avatar berhasil diperbarui');
        }
      } catch (error) {
        toast.error('Gagal mengupload avatar');
      } finally {
        setAvatarUploading(false);
      }
    } else {
      setBannerUploading(true);
      try {
        const response = await uploadsApi.uploadBanner(croppedFile);
        if (response.data?.url) {
          setBannerUrl(response.data.url);
          await profileApi.updateProfile({ banner_url: response.data.url });
          setUser({ ...user, banner_url: response.data.url });
          queryClient.invalidateQueries({ queryKey: ['user', user.username] });
          queryClient.invalidateQueries({ queryKey: ['profile', 'me'] });
          toast.success('Banner berhasil diperbarui');
        }
      } catch (error) {
        toast.error('Gagal mengupload banner');
      } finally {
        setBannerUploading(false);
      }
    }

    setCropperOpen(false);
    setPendingFile(null);
  };

  return (
    <div className="space-y-6">
      {/* Image Upload Guide - Collapsible */}
      <Collapsible open={imageGuideOpen} onOpenChange={setImageGuideOpen}>
        <CollapsibleTrigger asChild>
          <button className="flex w-full items-center justify-between rounded-lg border bg-muted/50 px-4 py-3 text-left hover:bg-muted">
            <div className="flex items-center gap-2">
              <ImageIcon className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">Panduan Ukuran Gambar</span>
            </div>
            <ChevronDown className={cn("h-4 w-4 text-muted-foreground transition-transform", imageGuideOpen && "rotate-180")} />
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent className="mt-2 rounded-lg border bg-card p-4">
          <div className="space-y-3 text-sm">
            <div>
              <div className="font-medium">Avatar (Foto Profil)</div>
              <div className="text-muted-foreground">Rasio 1:1 • Rekomendasi: 800x800px</div>
            </div>
            <div>
              <div className="font-medium">Banner</div>
              <div className="text-muted-foreground">Rasio 3:1 • Rekomendasi: 1500x500px</div>
            </div>
            <div className="flex items-start gap-2 rounded-md bg-blue-500/10 p-2 text-blue-600 dark:text-blue-400">
              <AlertCircle className="h-4 w-4 flex-shrink-0 mt-0.5" />
              <span className="text-xs">
                Gambar akan otomatis di-crop sesuai rasio yang direkomendasikan
              </span>
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* Banner & Avatar Preview */}
      <div className="relative overflow-hidden rounded-xl border bg-card">
        {/* Banner - 3:1 aspect ratio */}
        <div className="relative aspect-[3/1] w-full bg-gradient-to-r from-primary/20 to-primary/10">
          {bannerUrl && (
            <Image
              src={bannerUrl}
              alt="Banner"
              fill
              className="object-cover object-center"
              unoptimized
            />
          )}
          <button
            onClick={() => bannerInputRef.current?.click()}
            disabled={bannerUploading}
            className="absolute right-4 top-4 flex items-center gap-2 rounded-lg bg-background/90 px-3 py-2 text-sm font-medium shadow-lg backdrop-blur-sm transition-colors hover:bg-background disabled:opacity-50"
          >
            {bannerUploading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Camera className="h-4 w-4" />
            )}
            {bannerUploading ? 'Uploading...' : 'Ubah Banner'}
          </button>
          <input
            ref={bannerInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) handleImageSelect('banner', file);
              e.target.value = '';
            }}
          />
        </div>

        {/* Avatar - overlapping banner */}
        <div className="relative px-6 pb-6">
          <div className="relative -mt-12 inline-block">
            <Avatar className="h-24 w-24 border-4 border-background md:h-32 md:w-32 md:border-[6px]">
              <AvatarImage src={avatarUrl || undefined} alt={user.nama} />
              <AvatarFallback className="text-2xl md:text-4xl">
                {user.nama?.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <button
              onClick={() => avatarInputRef.current?.click()}
              disabled={avatarUploading}
              className="absolute bottom-0 right-0 rounded-full bg-primary p-2 text-primary-foreground shadow-lg transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {avatarUploading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Camera className="h-4 w-4" />
              )}
            </button>
            <input
              ref={avatarInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) handleImageSelect('avatar', file);
                e.target.value = '';
              }}
            />
          </div>
        </div>
      </div>

      {/* Profile Form */}
      <form onSubmit={profileForm.handleSubmit((data) => updateProfileMutation.mutate(data))} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="nama">Nama Lengkap</Label>
          <Input
            id="nama"
            {...profileForm.register('nama')}
            placeholder="Nama lengkap kamu"
          />
          {profileForm.formState.errors.nama && (
            <p className="text-sm text-destructive">{profileForm.formState.errors.nama.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="bio">Bio</Label>
          <Textarea
            id="bio"
            {...profileForm.register('bio')}
            placeholder="Ceritakan sedikit tentang diri kamu..."
            rows={4}
            className="resize-none"
          />
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>{profileForm.formState.errors.bio?.message || ''}</span>
            <span>{profileForm.watch('bio')?.length || 0}/500</span>
          </div>
        </div>

        <Button
          type="submit"
          disabled={updateProfileMutation.isPending || !profileForm.formState.isDirty}
          className="w-full"
        >
          {updateProfileMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Simpan Perubahan
        </Button>
      </form>

      {/* Image Cropper Modal */}
      {cropperOpen && cropperImage && (
        <ImageCropper
          image={cropperImage}
          aspect={cropType === 'avatar' ? 1 : 3}
          onCropComplete={handleCropComplete}
          onCancel={() => {
            setCropperOpen(false);
            setPendingFile(null);
          }}
        />
      )}
    </div>
  );
}
