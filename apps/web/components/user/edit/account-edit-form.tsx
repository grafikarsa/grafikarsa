'use client';

import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { motion, AnimatePresence } from 'framer-motion';
import { Loader2, Check, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { User } from '@/lib/types';
import { profileApi } from '@/lib/api';
import { useAuthStore } from '@/lib/stores/auth-store';
import { useDebounce } from '@/lib/hooks/use-debounce';

const accountSchema = z.object({
  username: z.string().min(3, 'Username minimal 3 karakter').max(30).regex(/^[a-z0-9_]+$/, 'Username hanya boleh huruf kecil, angka, dan underscore'),
  email: z.string().email('Email tidak valid').optional(),
  phone: z.string().max(20, 'Nomor telepon maksimal 20 karakter').optional(),
  address: z.string().max(500, 'Alamat maksimal 500 karakter').optional(),
  show_email: z.boolean(),
  show_phone: z.boolean(),
  show_address: z.boolean(),
});

type AccountFormData = z.infer<typeof accountSchema>;

interface AccountEditFormProps {
  user: User;
}

export function AccountEditForm({ user }: AccountEditFormProps) {
  const queryClient = useQueryClient();
  const { setUser } = useAuthStore();
  const [usernameAvailable, setUsernameAvailable] = useState<boolean | null>(null);
  const [checkingUsername, setCheckingUsername] = useState(false);

  const accountForm = useForm<AccountFormData>({
    resolver: zodResolver(accountSchema),
    defaultValues: {
      username: user.username,
      email: user.email || '',
      phone: user.phone || '',
      address: user.address || '',
      show_email: user.show_email || false,
      show_phone: user.show_phone || false,
      show_address: user.show_address || false,
    },
  });

  const watchedUsername = accountForm.watch('username');
  const debouncedUsername = useDebounce(watchedUsername, 500);

  useEffect(() => {
    if (debouncedUsername && debouncedUsername !== user.username) {
      setCheckingUsername(true);
      profileApi
        .checkUsername(debouncedUsername)
        .then((response) => {
          setUsernameAvailable(response.data?.available || false);
        })
        .catch(() => {
          setUsernameAvailable(false);
        })
        .finally(() => {
          setCheckingUsername(false);
        });
    } else {
      setUsernameAvailable(null);
    }
  }, [debouncedUsername, user.username]);

  const updateAccountMutation = useMutation({
    mutationFn: (data: AccountFormData) => profileApi.updateMe(data),
    onSuccess: (response) => {
      if (response.data) {
        setUser(response.data);
        queryClient.invalidateQueries({ queryKey: ['user', user.username] });
        queryClient.invalidateQueries({ queryKey: ['profile', 'me'] });
        toast.success('Akun berhasil diperbarui');
        
        // Redirect if username changed
        if (response.data.username !== user.username) {
          window.location.href = `/${response.data.username}/edit/account`;
        }
      }
    },
    onError: () => {
      toast.error('Gagal memperbarui akun');
    },
  });

  const isUsernameChanged = watchedUsername !== user.username;
  const canSubmit = accountForm.formState.isDirty && (!isUsernameChanged || usernameAvailable === true);

  return (
    <div className="space-y-6">
      {/* Account Form */}
      <form onSubmit={accountForm.handleSubmit((data) => updateAccountMutation.mutate(data))} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="username">Username</Label>
          <div className="relative">
            <Input
              id="username"
              {...accountForm.register('username')}
              placeholder="username"
              className="pr-10"
            />
            <div className="absolute right-3 top-1/2 -translate-y-1/2">
              <AnimatePresence mode="wait">
                {checkingUsername && (
                  <motion.div
                    key="checking"
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.8 }}
                  >
                    <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                  </motion.div>
                )}
                {!checkingUsername && usernameAvailable === true && (
                  <motion.div
                    key="available"
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.8 }}
                  >
                    <Check className="h-4 w-4 text-green-500" />
                  </motion.div>
                )}
                {!checkingUsername && usernameAvailable === false && (
                  <motion.div
                    key="unavailable"
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.8 }}
                  >
                    <X className="h-4 w-4 text-destructive" />
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </div>
          {accountForm.formState.errors.username && (
            <p className="text-sm text-destructive">{accountForm.formState.errors.username.message}</p>
          )}
          {!checkingUsername && usernameAvailable === false && (
            <p className="text-sm text-destructive">Username sudah digunakan</p>
          )}
          {!checkingUsername && usernameAvailable === true && (
            <p className="text-sm text-green-600">Username tersedia</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            type="email"
            {...accountForm.register('email')}
            placeholder="email@example.com"
          />
          {accountForm.formState.errors.email && (
            <p className="text-sm text-destructive">{accountForm.formState.errors.email.message}</p>
          )}
          <div className="flex items-center space-x-2 pt-1">
            <Checkbox
              id="show_email"
              checked={accountForm.watch('show_email')}
              onCheckedChange={(checked) => accountForm.setValue('show_email', checked as boolean, { shouldDirty: true })}
            />
            <label
              htmlFor="show_email"
              className="text-sm text-muted-foreground cursor-pointer"
            >
              Tampilkan email di profil publik
            </label>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="phone">Nomor Telepon</Label>
          <Input
            id="phone"
            type="tel"
            {...accountForm.register('phone')}
            placeholder="+62 812 3456 7890"
          />
          {accountForm.formState.errors.phone && (
            <p className="text-sm text-destructive">{accountForm.formState.errors.phone.message}</p>
          )}
          <div className="flex items-center space-x-2 pt-1">
            <Checkbox
              id="show_phone"
              checked={accountForm.watch('show_phone')}
              onCheckedChange={(checked) => accountForm.setValue('show_phone', checked as boolean, { shouldDirty: true })}
            />
            <label
              htmlFor="show_phone"
              className="text-sm text-muted-foreground cursor-pointer"
            >
              Tampilkan nomor telepon di profil publik
            </label>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="address">Alamat</Label>
          <Textarea
            id="address"
            {...accountForm.register('address')}
            placeholder="Alamat lengkap kamu..."
            rows={3}
            className="resize-none"
          />
          {accountForm.formState.errors.address && (
            <p className="text-sm text-destructive">{accountForm.formState.errors.address.message}</p>
          )}
          <div className="flex items-center space-x-2 pt-1">
            <Checkbox
              id="show_address"
              checked={accountForm.watch('show_address')}
              onCheckedChange={(checked) => accountForm.setValue('show_address', checked as boolean, { shouldDirty: true })}
            />
            <label
              htmlFor="show_address"
              className="text-sm text-muted-foreground cursor-pointer"
            >
              Tampilkan alamat di profil publik
            </label>
          </div>
        </div>

        <Button
          type="submit"
          disabled={updateAccountMutation.isPending || !canSubmit}
          className="w-full"
        >
          {updateAccountMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Simpan Perubahan
        </Button>
      </form>
    </div>
  );
}
