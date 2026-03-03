'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { dmApi } from '@/lib/api/dm';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Loader2, UserX } from 'lucide-react';
import { toast } from 'sonner';
import { useState } from 'react';

interface DMSettingsModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export function DMSettingsModal({ open, onOpenChange }: DMSettingsModalProps) {
    const queryClient = useQueryClient();
    const [activeTab, setActiveTab] = useState('privacy');

    // Privacy Settings
    const { data: settings, isLoading: isLoadingSettings } = useQuery({
        queryKey: ['dm-settings'],
        queryFn: dmApi.getSettings,
        enabled: open,
    });

    const updateSettingsMutation = useMutation({
        mutationFn: dmApi.updateSettings,
        onSuccess: (data) => {
            queryClient.setQueryData(['dm-settings'], data);
            toast.success('Privacy settings updated');
        },
        onError: () => toast.error('Failed to update settings'),
    });

    // Blocked Users
    const { data: blockedUsers, isLoading: isLoadingBlocked } = useQuery({
        queryKey: ['dm-blocked-users'],
        queryFn: dmApi.getBlockedUsers,
        enabled: open && activeTab === 'blocked',
    });

    const unblockMutation = useMutation({
        mutationFn: dmApi.unblockUser,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['dm-blocked-users'] });
            toast.success('User unblocked');
        },
        onError: () => toast.error('Failed to unblock user'),
    });

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[500px] h-[500px] flex flex-col p-0 gap-0 overflow-hidden">
                <DialogHeader className="p-4 border-b">
                    <DialogTitle className="text-center font-bold">DM Settings</DialogTitle>
                    <DialogDescription className="sr-only">
                        Manage your direct message privacy and blocked users.
                    </DialogDescription>
                </DialogHeader>

                <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col overflow-hidden">
                    <div className="px-4 pt-2">
                        <TabsList className="w-full grid grid-cols-2">
                            <TabsTrigger value="privacy">Privacy</TabsTrigger>
                            <TabsTrigger value="blocked">Blocked Accounts</TabsTrigger>
                        </TabsList>
                    </div>

                    <TabsContent value="privacy" className="flex-1 p-4 mt-0">
                        {isLoadingSettings ? (
                            <div className="flex h-full items-center justify-center">
                                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                            </div>
                        ) : (
                            <div className="space-y-6">
                                <div className="space-y-4">
                                    <div>
                                        <h3 className="text-base font-semibold">Message Requests</h3>
                                        <p className="text-sm text-muted-foreground">
                                            Decide who can send you direct messages.
                                        </p>
                                    </div>

                                    <RadioGroup
                                        value={settings?.data?.privacy || 'open'}
                                        onValueChange={(value: string) => updateSettingsMutation.mutate({ ...settings?.data, privacy: value as any })}
                                        className="gap-4"
                                    >
                                        <div className="flex items-start space-x-3 space-y-0">
                                            <RadioGroupItem value="open" id="open" />
                                            <div className="grid gap-1.5 leading-none">
                                                <Label htmlFor="open" className="font-medium cursor-pointer">
                                                    Everyone
                                                </Label>
                                                <p className="text-xs text-muted-foreground">
                                                    Anyone can send you a message request.
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-start space-x-3 space-y-0">
                                            <RadioGroupItem value="followers" id="followers" />
                                            <div className="grid gap-1.5 leading-none">
                                                <Label htmlFor="followers" className="font-medium cursor-pointer">
                                                    Followers Only
                                                </Label>
                                                <p className="text-xs text-muted-foreground">
                                                    Only people who track you can message you.
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-start space-x-3 space-y-0">
                                            <RadioGroupItem value="mutual" id="mutual" />
                                            <div className="grid gap-1.5 leading-none">
                                                <Label htmlFor="mutual" className="font-medium cursor-pointer">
                                                    Mutual Follows
                                                </Label>
                                                <p className="text-xs text-muted-foreground">
                                                    Only people you follow back can message you.
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-start space-x-3 space-y-0">
                                            <RadioGroupItem value="closed" id="closed" />
                                            <div className="grid gap-1.5 leading-none">
                                                <Label htmlFor="closed" className="font-medium cursor-pointer">
                                                    Nobody
                                                </Label>
                                                <p className="text-xs text-muted-foreground">
                                                    Receive no new message requests.
                                                </p>
                                            </div>
                                        </div>
                                    </RadioGroup>
                                </div>
                            </div>
                        )}
                    </TabsContent>

                    <TabsContent value="blocked" className="flex-1 p-0 mt-0 flex flex-col overflow-hidden">
                        {isLoadingBlocked ? (
                            <div className="flex h-full items-center justify-center">
                                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                            </div>
                        ) : blockedUsers?.data?.length === 0 ? (
                            <div className="flex flex-col h-full items-center justify-center text-muted-foreground gap-2">
                                <UserX className="h-10 w-10 opacity-20" />
                                <p>No blocked users.</p>
                            </div>
                        ) : (
                            <ScrollArea className="flex-1">
                                <div className="p-4 space-y-2">
                                    {blockedUsers?.data?.map((user: any) => (
                                        <div key={user.id} className="flex items-center justify-between p-3 rounded-lg border bg-card">
                                            <div className="flex items-center gap-3">
                                                <Avatar className="h-9 w-9">
                                                    <AvatarImage src={user.avatar_url} />
                                                    <AvatarFallback>{user.nama[0]}</AvatarFallback>
                                                </Avatar>
                                                <div>
                                                    <p className="font-medium text-sm">{user.nama}</p>
                                                    <p className="text-xs text-muted-foreground">@{user.username}</p>
                                                </div>
                                            </div>
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={() => unblockMutation.mutate(user.id)}
                                                disabled={unblockMutation.isPending}
                                            >
                                                Unblock
                                            </Button>
                                        </div>
                                    ))}
                                </div>
                            </ScrollArea>
                        )}
                    </TabsContent>
                </Tabs>
            </DialogContent>
        </Dialog>
    );
}
