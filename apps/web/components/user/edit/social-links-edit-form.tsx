'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { User, SocialLink, SocialPlatform } from '@/lib/types';
import { profileApi } from '@/lib/api';
import { socialPlatformConfigs } from '@/lib/constants/social-platforms';
import { PlatformIcon } from '../platform-icon';

// Generate social platforms list from config
const socialPlatforms: { value: SocialPlatform; label: string; placeholder: string }[] = [
  'instagram',
  'github',
  'linkedin',
  'twitter',
  'youtube',
  'tiktok',
  'behance',
  'dribbble',
  'personal_website',
].map((platform) => {
  const config = socialPlatformConfigs[platform as SocialPlatform];
  return {
    value: platform as SocialPlatform,
    label: config.label,
    placeholder: config.placeholder,
  };
});

interface SocialLinksEditFormProps {
  user: User;
}

export function SocialLinksEditForm({ user }: SocialLinksEditFormProps) {
  const queryClient = useQueryClient();
  const [socialLinks, setSocialLinks] = useState<SocialLink[]>(user.social_links || []);

  const updateSocialLinksMutation = useMutation({
    mutationFn: (links: SocialLink[]) => profileApi.updateSocialLinks(links),
    onSuccess: (response) => {
      if (response.data) {
        setSocialLinks(response.data.social_links);
        queryClient.invalidateQueries({ queryKey: ['user', user.username] });
        queryClient.invalidateQueries({ queryKey: ['profile', 'me'] });
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
        return prev.map((l) => (l.platform === platform ? { ...l, url } : l));
      }
      return [...prev, { platform, url }];
    });
  };

  const filledSocialLinksCount = socialLinks.filter(l => l.url).length;

  return (
    <div className="space-y-6">
      {/* Social Links Form */}
      <div className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          {socialPlatforms.map((platform) => (
            <div key={platform.value} className="space-y-2">
              <Label htmlFor={platform.value} className="flex items-center gap-2 text-sm">
                <PlatformIcon platform={platform.value} size="sm" />
                {platform.label}
              </Label>
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
          className="w-full"
        >
          {updateSocialLinksMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Simpan Social Links
        </Button>
      </div>
    </div>
  );
}
