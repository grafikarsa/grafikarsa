'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2, Eye, EyeOff } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { profileApi } from '@/lib/api';

const passwordSchema = z.object({
  current_password: z.string().min(1, 'Password saat ini wajib diisi'),
  new_password: z.string().min(8, 'Password minimal 8 karakter'),
  new_password_confirmation: z.string(),
}).refine((data) => data.new_password === data.new_password_confirmation, {
  message: 'Konfirmasi password tidak cocok',
  path: ['new_password_confirmation'],
});

type PasswordFormData = z.infer<typeof passwordSchema>;

export function PasswordEditForm() {
  const [showCurrentPassword, setShowCurrentPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  const passwordForm = useForm<PasswordFormData>({
    resolver: zodResolver(passwordSchema),
    defaultValues: {
      current_password: '',
      new_password: '',
      new_password_confirmation: '',
    },
  });

  const updatePasswordMutation = useMutation({
    mutationFn: (data: PasswordFormData) => profileApi.updatePassword(data),
    onSuccess: () => {
      toast.success('Password berhasil diperbarui');
      passwordForm.reset();
    },
    onError: (error: any) => {
      const message = error.response?.data?.error || 'Gagal memperbarui password';
      toast.error(message);
    },
  });

  return (
    <div className="space-y-6">
      {/* Password Form */}
      <form onSubmit={passwordForm.handleSubmit((data) => updatePasswordMutation.mutate(data))} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="current_password">Password Saat Ini</Label>
          <div className="relative">
            <Input
              id="current_password"
              type={showCurrentPassword ? 'text' : 'password'}
              {...passwordForm.register('current_password')}
              placeholder="Masukkan password saat ini"
              className="pr-10"
            />
            <button
              type="button"
              onClick={() => setShowCurrentPassword(!showCurrentPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            >
              {showCurrentPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {passwordForm.formState.errors.current_password && (
            <p className="text-sm text-destructive">{passwordForm.formState.errors.current_password.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="new_password">Password Baru</Label>
          <div className="relative">
            <Input
              id="new_password"
              type={showNewPassword ? 'text' : 'password'}
              {...passwordForm.register('new_password')}
              placeholder="Masukkan password baru"
              className="pr-10"
            />
            <button
              type="button"
              onClick={() => setShowNewPassword(!showNewPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            >
              {showNewPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {passwordForm.formState.errors.new_password && (
            <p className="text-sm text-destructive">{passwordForm.formState.errors.new_password.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="new_password_confirmation">Konfirmasi Password Baru</Label>
          <div className="relative">
            <Input
              id="new_password_confirmation"
              type={showConfirmPassword ? 'text' : 'password'}
              {...passwordForm.register('new_password_confirmation')}
              placeholder="Konfirmasi password baru"
              className="pr-10"
            />
            <button
              type="button"
              onClick={() => setShowConfirmPassword(!showConfirmPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            >
              {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {passwordForm.formState.errors.new_password_confirmation && (
            <p className="text-sm text-destructive">{passwordForm.formState.errors.new_password_confirmation.message}</p>
          )}
        </div>

        <Button
          type="submit"
          disabled={updatePasswordMutation.isPending || !passwordForm.formState.isDirty}
          className="w-full"
        >
          {updatePasswordMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Ubah Password
        </Button>
      </form>
    </div>
  );
}
