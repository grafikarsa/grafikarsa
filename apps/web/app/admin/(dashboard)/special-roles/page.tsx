'use client';

import React, { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, Loader2, Shield, Users, X, Search, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { Textarea } from '@/components/ui/textarea';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { DataTable, Column } from '@/components/admin/data-table';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { adminSpecialRolesApi, adminUsersApi } from '@/lib/api/admin';
import {
  SpecialRole,
  SpecialRoleDetail,
  CapabilityInfo,
  capabilityGroups,
  presetColors,
  generateBgColor,
} from '@/lib/types';
import { useDebounce } from '@/lib/hooks/use-debounce';

export default function AdminSpecialRolesPage() {
  const queryClient = useQueryClient();
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [editRole, setEditRole] = useState<SpecialRole | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteRole, setDeleteRole] = useState<SpecialRole | null>(null);
  const [manageUsersRole, setManageUsersRole] = useState<SpecialRole | null>(null);

  const handleSearch = () => {
    setSearchQuery(searchInput);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  const { data, isLoading } = useQuery({
    queryKey: ['admin-special-roles', searchQuery],
    queryFn: () => adminSpecialRolesApi.getSpecialRoles({ search: searchQuery || undefined, include_inactive: true }),
  });

  const { data: capabilitiesData } = useQuery({
    queryKey: ['capabilities'],
    queryFn: () => adminSpecialRolesApi.getCapabilities(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminSpecialRolesApi.deleteSpecialRole(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-special-roles'] });
      toast.success('Special role berhasil dihapus');
      setDeleteRole(null);
    },
    onError: () => {
      toast.error('Gagal menghapus special role');
    },
  });

  const toggleActiveMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      adminSpecialRolesApi.updateSpecialRole(id, { is_active }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-special-roles'] });
      toast.success('Status special role diperbarui');
    },
    onError: () => {
      toast.error('Gagal memperbarui status');
    },
  });

  const roles = data?.data || [];
  const capabilities = capabilitiesData?.data || [];

  const getCapabilityLabel = (key: string) => {
    const cap = capabilities.find((c) => c.key === key);
    return cap?.label || key;
  };

  const columns: Column<SpecialRole>[] = [
    {
      key: 'nama',
      header: 'Nama Role',
      render: (r) => (
        <div className="flex items-center gap-3">
          <div
            className="flex h-8 w-8 items-center justify-center rounded-full"
            style={{ backgroundColor: generateBgColor(r.color, 0.2) }}
          >
            <Shield className="h-4 w-4" style={{ color: r.color }} />
          </div>
          <div>
            <span className="font-medium">{r.nama}</span>
            {r.description && (
              <p className="text-xs text-muted-foreground line-clamp-1">{r.description}</p>
            )}
          </div>
        </div>
      ),
    },
    {
      key: 'capabilities',
      header: 'Capabilities',
      render: (r) => {
        const caps = r.capabilities || [];
        return (
          <div className="flex flex-wrap gap-1 max-w-xs">
            {caps.slice(0, 3).map((cap) => (
              <Badge key={cap} variant="outline" className="text-xs">
                {getCapabilityLabel(cap)}
              </Badge>
            ))}
            {caps.length > 3 && (
              <Badge variant="secondary" className="text-xs">
                +{caps.length - 3}
              </Badge>
            )}
          </div>
        );
      },
    },
    {
      key: 'user_count',
      header: 'Users',
      render: (r) => (
        <Button
          variant="ghost"
          size="sm"
          className="gap-1.5"
          onClick={(e) => {
            e.stopPropagation();
            setManageUsersRole(r);
          }}
        >
          <Users className="h-3.5 w-3.5" />
          {r.user_count || 0}
        </Button>
      ),
    },
    {
      key: 'is_active',
      header: 'Status',
      render: (r) => (
        <div className="flex items-center gap-2">
          <Switch
            checked={r.is_active}
            onCheckedChange={(checked) =>
              toggleActiveMutation.mutate({ id: r.id, is_active: checked })
            }
          />
          <Badge
            variant={r.is_active ? 'default' : 'secondary'}
            className={r.is_active ? 'bg-green-100 text-green-700 hover:bg-green-100 dark:bg-green-900/30 dark:text-green-400' : ''}
          >
            {r.is_active ? 'Aktif' : 'Nonaktif'}
          </Badge>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Search and Action Button in same row */}
      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Cari special role..."
            className="pl-9"
          />
        </div>
        <Button variant="secondary" onClick={handleSearch}>
          <Search className="mr-2 h-4 w-4" />
          Cari
        </Button>
        <Button onClick={() => setIsCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Tambah Special Role
        </Button>
      </div>

      <DataTable
        data={roles}
        columns={columns}
        isLoading={isLoading}
        onEdit={setEditRole}
        onDelete={setDeleteRole}
      />

      <SpecialRoleFormDialog
        role={editRole}
        capabilities={capabilities}
        open={isCreateOpen || !!editRole}
        onClose={() => {
          setIsCreateOpen(false);
          setEditRole(null);
        }}
      />

      <ManageUsersDialog
        role={manageUsersRole}
        open={!!manageUsersRole}
        onClose={() => setManageUsersRole(null)}
      />

      <ConfirmDialog
        open={!!deleteRole}
        onOpenChange={() => setDeleteRole(null)}
        title="Hapus Special Role"
        description={
          <>
            Yakin ingin menghapus special role <strong>&quot;{deleteRole?.nama}&quot;</strong>?
            Users yang memiliki role ini akan kehilangan akses terkait.
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteRole && deleteMutation.mutate(deleteRole.id)}
      />
    </div>
  );
}


function SpecialRoleFormDialog({
  role,
  capabilities,
  open,
  onClose,
}: {
  role: SpecialRole | null;
  capabilities: CapabilityInfo[];
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const isEdit = !!role;
  const [nama, setNama] = useState('');
  const [description, setDescription] = useState('');
  const [color, setColor] = useState('#6366f1');
  const [selectedCapabilities, setSelectedCapabilities] = useState<string[]>([]);
  const [isActive, setIsActive] = useState(true);

  useEffect(() => {
    if (open) {
      if (role) {
        setNama(role.nama);
        setDescription(role.description || '');
        setColor(role.color);
        setSelectedCapabilities(role.capabilities || []);
        setIsActive(role.is_active);
      } else {
        setNama('');
        setDescription('');
        setColor('#6366f1');
        setSelectedCapabilities([]);
        setIsActive(true);
      }
    }
  }, [role, open]);

  const createMutation = useMutation({
    mutationFn: () =>
      adminSpecialRolesApi.createSpecialRole({
        nama,
        description: description || undefined,
        color,
        capabilities: selectedCapabilities,
        is_active: isActive,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-special-roles'] });
      toast.success('Special role berhasil dibuat');
      onClose();
    },
    onError: () => {
      toast.error('Gagal membuat special role');
    },
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      adminSpecialRolesApi.updateSpecialRole(role!.id, {
        nama,
        description: description || undefined,
        color,
        capabilities: selectedCapabilities,
        is_active: isActive,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-special-roles'] });
      toast.success('Special role berhasil diperbarui');
      onClose();
    },
    onError: () => {
      toast.error('Gagal memperbarui special role');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isEdit) {
      updateMutation.mutate();
    } else {
      createMutation.mutate();
    }
  };

  const toggleCapability = (key: string) => {
    setSelectedCapabilities((prev) =>
      prev.includes(key) ? prev.filter((c) => c !== key) : [...prev, key]
    );
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  // Group capabilities
  const groupedCapabilities = capabilityGroups.map((group) => ({
    ...group,
    items: capabilities.filter((c) => group.capabilities.includes(c.key)),
  }));

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[85vh] p-0 gap-0 flex flex-col">
        <DialogHeader className="px-6 pt-6 pb-4 border-b shrink-0">
          <div className="flex items-center gap-3">
            <div
              className="rounded-full p-2"
              style={{ backgroundColor: generateBgColor(color, 0.2) }}
            >
              <Shield className="h-5 w-5" style={{ color }} />
            </div>
            <div>
              <DialogTitle>{isEdit ? 'Edit Special Role' : 'Tambah Special Role Baru'}</DialogTitle>
              <DialogDescription>
                {isEdit
                  ? 'Ubah detail special role'
                  : 'Buat special role baru dengan capabilities tertentu'}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto px-6 py-4">
          <div className="space-y-6">
            {/* Basic Info */}
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="nama">
                  Nama Role <span className="text-destructive">*</span>
                </Label>
                <Input
                  id="nama"
                  value={nama}
                  onChange={(e) => setNama(e.target.value)}
                  placeholder="Contoh: Moderator Konten"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Deskripsi</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Deskripsi singkat tentang role ini"
                  rows={2}
                />
              </div>
            </div>

            {/* Color Picker */}
            <div className="space-y-3">
              <Label>Warna Badge</Label>
              <div className="flex flex-wrap gap-2">
                {presetColors.map((preset) => (
                  <button
                    key={preset.value}
                    type="button"
                    className="relative h-8 w-8 rounded-full border-2 transition-all hover:scale-110"
                    style={{
                      backgroundColor: preset.value,
                      borderColor: color === preset.value ? 'white' : 'transparent',
                      boxShadow: color === preset.value ? `0 0 0 2px ${preset.value}` : 'none',
                    }}
                    onClick={() => setColor(preset.value)}
                    title={preset.name}
                  >
                    {color === preset.value && (
                      <Check className="absolute inset-0 m-auto h-4 w-4 text-white" />
                    )}
                  </button>
                ))}
              </div>
              <div className="flex items-center gap-3">
                <Label className="text-sm text-muted-foreground">Preview:</Label>
                <Badge
                  style={{
                    backgroundColor: generateBgColor(color, 0.15),
                    color: color,
                    borderColor: generateBgColor(color, 0.3),
                  }}
                  variant="outline"
                >
                  {nama || 'Nama Role'}
                </Badge>
              </div>
            </div>

            {/* Capabilities */}
            <div className="space-y-3">
              <Label>
                Capabilities <span className="text-destructive">*</span>
              </Label>
              <p className="text-xs text-muted-foreground">
                Pilih menu admin yang dapat diakses oleh role ini
              </p>
              <div className="space-y-4 rounded-lg border p-4">
                {groupedCapabilities.map((group) => (
                  <div key={group.title}>
                    <h4 className="mb-2 text-sm font-medium text-muted-foreground">
                      {group.title}
                    </h4>
                    <div className="grid grid-cols-2 gap-2">
                      {group.items.map((cap) => (
                        <label
                          key={cap.key}
                          className="flex cursor-pointer items-center gap-2 rounded-md border p-2 hover:bg-muted/50"
                        >
                          <Checkbox
                            checked={selectedCapabilities.includes(cap.key)}
                            onCheckedChange={() => toggleCapability(cap.key)}
                          />
                          <span className="text-sm">{cap.label}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Status */}
            <div className="flex items-center justify-between rounded-lg border p-4">
              <div>
                <Label>Status Aktif</Label>
                <p className="text-xs text-muted-foreground">
                  Role aktif dapat digunakan dan memberikan akses
                </p>
              </div>
              <Switch checked={isActive} onCheckedChange={setIsActive} />
            </div>
          </div>
        </div>

        <DialogFooter className="px-6 py-4 border-t shrink-0">
          <Button type="button" variant="outline" onClick={onClose}>
            Batal
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={isPending || !nama.trim() || selectedCapabilities.length === 0}
          >
            {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isEdit ? 'Simpan Perubahan' : 'Buat Role'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}


function ManageUsersDialog({
  role,
  open,
  onClose,
}: {
  role: SpecialRole | null;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [isAddingUsers, setIsAddingUsers] = useState(false);
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);
  const debouncedSearch = useDebounce(search, 300);

  // Fetch role detail with users
  const { data: roleDetail, isLoading: isLoadingRole } = useQuery({
    queryKey: ['special-role-detail', role?.id],
    queryFn: () => adminSpecialRolesApi.getSpecialRole(role!.id),
    enabled: !!role?.id && open,
  });

  // Fetch users for adding
  const { data: usersData, isLoading: isLoadingUsers } = useQuery({
    queryKey: ['admin-users-for-role', debouncedSearch],
    queryFn: () => adminUsersApi.getUsers({ search: debouncedSearch || undefined, limit: 20 }),
    enabled: isAddingUsers,
  });

  const assignMutation = useMutation({
    mutationFn: () => adminSpecialRolesApi.assignUsers(role!.id, { user_ids: selectedUserIds }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['special-role-detail', role?.id] });
      queryClient.invalidateQueries({ queryKey: ['admin-special-roles'] });
      toast.success('Users berhasil ditambahkan');
      setIsAddingUsers(false);
      setSelectedUserIds([]);
      setSearch('');
    },
    onError: () => {
      toast.error('Gagal menambahkan users');
    },
  });

  const removeMutation = useMutation({
    mutationFn: (userId: string) => adminSpecialRolesApi.removeUser(role!.id, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['special-role-detail', role?.id] });
      queryClient.invalidateQueries({ queryKey: ['admin-special-roles'] });
      toast.success('User berhasil dihapus dari role');
    },
    onError: () => {
      toast.error('Gagal menghapus user dari role');
    },
  });

  const detail = roleDetail?.data;
  const users = detail?.users || [];
  const availableUsers = usersData?.data || [];

  // Filter out users already in role
  const userIdsInRole = new Set(users.map((u) => u.id));
  const filteredAvailableUsers = availableUsers.filter((u) => !userIdsInRole.has(u.id));

  const toggleUserSelection = (userId: string) => {
    setSelectedUserIds((prev) =>
      prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId]
    );
  };

  const handleClose = () => {
    setIsAddingUsers(false);
    setSelectedUserIds([]);
    setSearch('');
    onClose();
  };

  if (!role) return null;

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg max-h-[80vh] p-0 gap-0 flex flex-col">
        <DialogHeader className="px-6 pt-6 pb-4 border-b shrink-0">
          <div className="flex items-center gap-3">
            <div
              className="rounded-full p-2"
              style={{ backgroundColor: generateBgColor(role.color, 0.2) }}
            >
              <Users className="h-5 w-5" style={{ color: role.color }} />
            </div>
            <div>
              <DialogTitle>Kelola Users - {role.nama}</DialogTitle>
              <DialogDescription>
                {isAddingUsers
                  ? 'Pilih users untuk ditambahkan ke role ini'
                  : `${users.length} user memiliki role ini`}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {isAddingUsers ? (
          // Add Users View
          <>
            <div className="px-6 pt-4 shrink-0">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Cari user..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-9"
                />
              </div>
            </div>
            <div className="flex-1 overflow-y-auto px-6 py-4">
              {isLoadingUsers ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : filteredAvailableUsers.length === 0 ? (
                <p className="text-center text-sm text-muted-foreground py-8">
                  {search ? 'Tidak ada user ditemukan' : 'Semua user sudah memiliki role ini'}
                </p>
              ) : (
                <div className="space-y-2">
                  {filteredAvailableUsers.map((user) => (
                    <label
                      key={user.id}
                      className="flex cursor-pointer items-center gap-3 rounded-md border p-3 hover:bg-muted/50"
                    >
                      <Checkbox
                        checked={selectedUserIds.includes(user.id)}
                        onCheckedChange={() => toggleUserSelection(user.id)}
                      />
                      <Avatar className="h-8 w-8">
                        <AvatarImage src={user.avatar_url} />
                        <AvatarFallback>{user.nama?.charAt(0)}</AvatarFallback>
                      </Avatar>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{user.nama}</p>
                        <p className="text-xs text-muted-foreground truncate">@{user.username}</p>
                      </div>
                    </label>
                  ))}
                </div>
              )}
            </div>
            <DialogFooter className="px-6 py-4 border-t shrink-0">
              <Button variant="outline" onClick={() => setIsAddingUsers(false)}>
                Kembali
              </Button>
              <Button
                onClick={() => assignMutation.mutate()}
                disabled={selectedUserIds.length === 0 || assignMutation.isPending}
              >
                {assignMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Tambahkan ({selectedUserIds.length})
              </Button>
            </DialogFooter>
          </>
        ) : (
          // Users List View
          <>
            <div className="flex-1 overflow-y-auto px-6 py-4">
              {isLoadingRole ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : users.length === 0 ? (
                <p className="text-center text-sm text-muted-foreground py-8">
                  Belum ada user dengan role ini
                </p>
              ) : (
                <div className="space-y-2">
                  {users.map((user) => (
                    <div
                      key={user.id}
                      className="flex items-center gap-3 rounded-md border p-3"
                    >
                      <Avatar className="h-8 w-8">
                        <AvatarImage src={user.avatar_url || undefined} />
                        <AvatarFallback>{user.nama?.charAt(0)}</AvatarFallback>
                      </Avatar>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{user.nama}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          @{user.username}
                          {user.kelas_nama && ` • ${user.kelas_nama}`}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-muted-foreground hover:text-destructive"
                        onClick={() => removeMutation.mutate(user.id)}
                        disabled={removeMutation.isPending}
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <DialogFooter className="px-6 py-4 border-t shrink-0">
              <Button variant="outline" onClick={handleClose}>
                Tutup
              </Button>
              <Button onClick={() => setIsAddingUsers(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Tambah User
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
