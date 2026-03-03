'use client';

import { useState, useEffect, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Plus,
  Search,
  Loader2,
  UserCheck,
  UserX,
  Eye,
  EyeOff,
  Pencil,
  Trash2,
  Users,
  GraduationCap,
  Shield,
  Mail,
  Calendar,
  MapPin,
  ExternalLink,
  X,
  Check,
  AlertCircle,
  Upload,
  ImageIcon,
} from 'lucide-react';
import Image from 'next/image';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ConfirmDialog } from '@/components/admin/confirm-dialog';
import { adminUsersApi, adminMajorsApi, adminClassesApi, uploadsApi, adminSpecialRolesApi } from '@/lib/api/admin';
import { User, UserRole, SpecialRole, generateBgColor } from '@/lib/types';
import { useDebounce } from '@/lib/hooks/use-debounce';
import { formatDate } from '@/lib/utils/format';
import { Checkbox } from '@/components/ui/checkbox';

const roleOptions = [
  { value: 'all', label: 'Semua Role' },
  { value: 'student', label: 'Siswa' },
  { value: 'alumni', label: 'Alumni' },
  { value: 'admin', label: 'Admin' },
];

const statusOptions = [
  { value: 'all', label: 'Semua Status' },
  { value: 'active', label: 'Aktif' },
  { value: 'inactive', label: 'Nonaktif' },
];

const roleStyles: Record<string, string> = {
  student: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  alumni: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  admin: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
};

const roleIcons: Record<string, React.ReactNode> = {
  student: <GraduationCap className="h-3 w-3" />,
  alumni: <Users className="h-3 w-3" />,
  admin: <Shield className="h-3 w-3" />,
};

export default function AdminUsersPage() {
  const queryClient = useQueryClient();
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [roleFilter, setRoleFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [page, setPage] = useState(1);

  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null);

  const handleSearch = () => {
    setSearchQuery(searchInput);
    setPage(1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  // Fetch users
  const { data, isLoading } = useQuery({
    queryKey: ['admin-users', searchQuery, roleFilter, statusFilter, page],
    queryFn: () =>
      adminUsersApi.getUsers({
        search: searchQuery || undefined,
        role: roleFilter === 'all' ? undefined : roleFilter,
        is_active: statusFilter === 'all' ? undefined : statusFilter === 'active',
        page,
        limit: 15,
      }),
  });

  // Fetch user detail
  const { data: userDetail, isLoading: detailLoading } = useQuery({
    queryKey: ['admin-user-detail', selectedUser?.id],
    queryFn: () => adminUsersApi.getUser(selectedUser!.id),
    enabled: !!selectedUser && showDetailModal,
  });

  const toggleActiveMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) => 
      adminUsersApi.toggleActive(id, isActive),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      toast.success('Status user berhasil diubah');
    },
    onError: () => toast.error('Gagal mengubah status user'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminUsersApi.deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      toast.success('User berhasil dihapus');
      setDeleteTarget(null);
    },
    onError: () => toast.error('Gagal menghapus user'),
  });

  const users = data?.data || [];
  const pagination = data?.meta;

  const handleViewDetail = (user: User) => {
    setSelectedUser(user);
    setShowDetailModal(true);
  };

  const handleEdit = (user: User) => {
    setEditingUser(user);
    setShowEditModal(true);
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-9 w-32" />
          <Skeleton className="h-10 w-36" />
        </div>
        <div className="flex gap-4">
          <Skeleton className="h-10 flex-1" />
          <Skeleton className="h-10 w-40" />
          <Skeleton className="h-10 w-40" />
        </div>
        <Card className="p-0">
          <div className="space-y-4 p-6">
            {[...Array(8)].map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Search, Filters, and Action Button in same row */}
      <div className="flex flex-wrap gap-3">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari nama, username, email..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={handleKeyDown}
            className="pl-9"
          />
        </div>
        <Select
          value={roleFilter}
          onValueChange={(v) => {
            setRoleFilter(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Filter role" />
          </SelectTrigger>
          <SelectContent>
            {roleOptions.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={statusFilter}
          onValueChange={(v) => {
            setStatusFilter(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Filter status" />
          </SelectTrigger>
          <SelectContent>
            {statusOptions.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="secondary" onClick={handleSearch}>
          <Search className="mr-2 h-4 w-4" />
          Cari
        </Button>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Tambah User
        </Button>
      </div>

      {/* Table */}
      {users.length === 0 ? (
        <Card className="border-dashed py-16">
          <div className="flex flex-col items-center justify-center">
            <div className="rounded-full bg-muted p-4">
              <Users className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-lg font-semibold">Tidak ada user</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {searchQuery || roleFilter !== 'all' || statusFilter !== 'all'
                ? 'Tidak ada user yang sesuai filter'
                : 'Belum ada user terdaftar'}
            </p>
          </div>
        </Card>
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/50">
                <TableHead className="w-12 text-center">#</TableHead>
                <TableHead className="w-[300px]">User</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Kelas</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Terdaftar</TableHead>
                <TableHead className="w-[100px] text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((user, index) => (
                <TableRow key={user.id} className="group">
                  <TableCell className="text-center text-muted-foreground">
                    {(page - 1) * 15 + index + 1}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <Avatar className="h-10 w-10 border">
                        <AvatarImage src={user.avatar_url} alt={user.nama} />
                        <AvatarFallback className="text-sm font-medium">
                          {user.nama?.charAt(0)}
                        </AvatarFallback>
                      </Avatar>
                      <div className="min-w-0">
                        <p className="truncate font-medium">{user.nama}</p>
                        <p className="truncate text-sm text-muted-foreground">
                          @{user.username}
                          {user.email && ` · ${user.email}`}
                        </p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge className={`gap-1 capitalize ${roleStyles[user.role] || ''}`}>
                      {roleIcons[user.role]}
                      {user.role === 'student' ? 'Siswa' : user.role === 'alumni' ? 'Alumni' : 'Admin'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm">
                      {user.kelas?.nama || (user.role === 'alumni' ? 'Lulus' : '-')}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={user.is_active ? 'default' : 'destructive'}
                      className={user.is_active ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : ''}
                    >
                      {user.is_active ? 'Aktif' : 'Nonaktif'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {formatDate(user.created_at)}
                    </span>
                  </TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                          <span className="sr-only">Menu</span>
                          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                          </svg>
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => handleViewDetail(user)}>
                          <Eye className="mr-2 h-4 w-4" />
                          Lihat Detail
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleEdit(user)}>
                          <Pencil className="mr-2 h-4 w-4" />
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => toggleActiveMutation.mutate({ id: user.id, isActive: user.is_active ?? false })}
                          disabled={toggleActiveMutation.isPending}
                        >
                          {user.is_active ? (
                            <>
                              <UserX className="mr-2 h-4 w-4" />
                              Nonaktifkan
                            </>
                          ) : (
                            <>
                              <UserCheck className="mr-2 h-4 w-4" />
                              Aktifkan
                            </>
                          )}
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => setDeleteTarget(user)}
                          className="text-destructive focus:text-destructive"
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          Hapus
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      {/* Pagination */}
      {pagination && pagination.total_pages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Menampilkan {users.length} dari {pagination.total_count} user
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
            >
              Previous
            </Button>
            <span className="text-sm">
              Halaman {page} dari {pagination.total_pages}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
              disabled={page === pagination.total_pages}
            >
              Next
            </Button>
          </div>
        </div>
      )}

      {/* User Detail Modal */}
      <UserDetailModal
        user={selectedUser}
        userDetail={userDetail?.data}
        isLoading={detailLoading}
        open={showDetailModal}
        onClose={() => {
          setShowDetailModal(false);
          setSelectedUser(null);
        }}
        onEdit={() => {
          setShowDetailModal(false);
          if (selectedUser) handleEdit(selectedUser);
        }}
      />

      {/* Create Modal */}
      <UserFormModal
        open={showCreateModal}
        onOpenChange={setShowCreateModal}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['admin-users'] })}
      />

      {/* Edit Modal */}
      <UserFormModal
        open={showEditModal}
        onOpenChange={setShowEditModal}
        user={editingUser}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['admin-users'] })}
      />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Hapus User"
        description={
          <>
            Yakin ingin menghapus user <strong>{deleteTarget?.nama}</strong>? Tindakan ini tidak dapat dibatalkan.
          </>
        }
        confirmText="Hapus"
        variant="destructive"
        isLoading={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  );
}


// User Detail Modal
function UserDetailModal({
  user,
  userDetail,
  isLoading,
  open,
  onClose,
  onEdit,
}: {
  user: User | null;
  userDetail?: User;
  isLoading: boolean;
  open: boolean;
  onClose: () => void;
  onEdit: () => void;
}) {
  if (!user) return null;

  const detail = userDetail || user;

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto p-0">
        <DialogTitle className="sr-only">Detail User: {detail.nama}</DialogTitle>
        {/* Banner & Avatar */}
        <div className="relative">
          <div className="h-32 bg-gradient-to-r from-primary/20 to-primary/5">
            {detail.banner_url && (
              <Image
                src={detail.banner_url}
                alt="Banner"
                fill
                className="object-cover"
              />
            )}
          </div>
          <div className="absolute -bottom-12 left-6">
            <Avatar className="h-24 w-24 border-4 border-background">
              <AvatarImage src={detail.avatar_url} alt={detail.nama} />
              <AvatarFallback className="text-2xl font-bold">
                {detail.nama?.charAt(0)}
              </AvatarFallback>
            </Avatar>
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="absolute right-2 top-2 bg-background/80 hover:bg-background"
            onClick={onClose}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Content */}
        <div className="px-6 pb-6 pt-14">
          {isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-8 w-48" />
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-20 w-full" />
            </div>
          ) : (
            <>
              {/* Header Info */}
              <div className="flex items-start justify-between">
                <div>
                  <h2 className="text-xl font-bold">{detail.nama}</h2>
                  <p className="text-muted-foreground">@{detail.username}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Badge className={`gap-1 capitalize ${roleStyles[detail.role] || ''}`}>
                    {roleIcons[detail.role]}
                    {detail.role === 'student' ? 'Siswa' : detail.role === 'alumni' ? 'Alumni' : 'Admin'}
                  </Badge>
                  <Badge
                    className={detail.is_active ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : ''}
                    variant={detail.is_active ? 'default' : 'destructive'}
                  >
                    {detail.is_active ? 'Aktif' : 'Nonaktif'}
                  </Badge>
                </div>
              </div>

              {/* Bio */}
              {detail.bio && (
                <p className="mt-3 text-sm text-muted-foreground">{detail.bio}</p>
              )}

              {/* Info Grid */}
              <div className="mt-6 grid gap-4 sm:grid-cols-2">
                {detail.email && (
                  <div className="flex items-center gap-3 rounded-lg border p-3">
                    <div className="rounded-full bg-muted p-2">
                      <Mail className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Email</p>
                      <p className="text-sm font-medium">{detail.email}</p>
                    </div>
                  </div>
                )}

                {detail.kelas && (
                  <div className="flex items-center gap-3 rounded-lg border p-3">
                    <div className="rounded-full bg-muted p-2">
                      <GraduationCap className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Kelas</p>
                      <p className="text-sm font-medium">{detail.kelas.nama}</p>
                    </div>
                  </div>
                )}

                {detail.jurusan && (
                  <div className="flex items-center gap-3 rounded-lg border p-3">
                    <div className="rounded-full bg-muted p-2">
                      <MapPin className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Jurusan</p>
                      <p className="text-sm font-medium">{detail.jurusan.nama}</p>
                    </div>
                  </div>
                )}

                {detail.tahun_masuk && (
                  <div className="flex items-center gap-3 rounded-lg border p-3">
                    <div className="rounded-full bg-muted p-2">
                      <Calendar className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Tahun Masuk</p>
                      <p className="text-sm font-medium">{detail.tahun_masuk}</p>
                    </div>
                  </div>
                )}

                {detail.nisn && (
                  <div className="flex items-center gap-3 rounded-lg border p-3">
                    <div className="rounded-full bg-muted p-2">
                      <Users className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">NISN</p>
                      <p className="text-sm font-medium">{detail.nisn}</p>
                    </div>
                  </div>
                )}

                {detail.nis && (
                  <div className="flex items-center gap-3 rounded-lg border p-3">
                    <div className="rounded-full bg-muted p-2">
                      <Users className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">NIS</p>
                      <p className="text-sm font-medium">{detail.nis}</p>
                    </div>
                  </div>
                )}
              </div>

              {/* Class History */}
              {detail.class_history && detail.class_history.length > 0 && (
                <div className="mt-6">
                  <h3 className="mb-3 text-sm font-semibold">Riwayat Kelas</h3>
                  <div className="flex flex-wrap gap-2">
                    {detail.class_history.map((ch, i) => (
                      <Badge key={i} variant="outline">
                        {ch.kelas_nama} ({ch.tahun_ajaran})
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* Social Links */}
              {detail.social_links && detail.social_links.length > 0 && (
                <div className="mt-6">
                  <h3 className="mb-3 text-sm font-semibold">Social Links</h3>
                  <div className="flex flex-wrap gap-2">
                    {detail.social_links.map((link, i) => (
                      <a
                        key={i}
                        href={link.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 rounded-full bg-muted px-3 py-1 text-sm hover:bg-muted/80"
                      >
                        {link.platform}
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    ))}
                  </div>
                </div>
              )}

              {/* Timestamps */}
              <div className="mt-6 flex flex-wrap gap-4 border-t pt-4 text-xs text-muted-foreground">
                <span>Terdaftar: {formatDate(detail.created_at)}</span>
                {detail.last_login_at && (
                  <span>Login terakhir: {formatDate(detail.last_login_at)}</span>
                )}
              </div>

              {/* Actions */}
              <div className="mt-6 flex justify-between border-t pt-4">
                <Link href={`/${detail.username}`}>
                  <Button variant="outline" size="sm">
                    <Eye className="mr-2 h-4 w-4" />
                    Lihat Profil
                  </Button>
                </Link>
                <Button size="sm" onClick={onEdit}>
                  <Pencil className="mr-2 h-4 w-4" />
                  Edit User
                </Button>
              </div>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

// User Form Modal (Create/Edit)
function UserFormModal({
  open,
  onOpenChange,
  user,
  onSuccess,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user?: User | null;
  onSuccess: () => void;
}) {
  const isEdit = !!user;
  const [formData, setFormData] = useState({
    nama: '',
    username: '',
    email: '',
    password: '',
    role: 'student' as UserRole,
    nisn: '',
    nis: '',
    kelas_id: '',
    tahun_masuk: new Date().getFullYear(),
    tahun_lulus: null as number | null,
    avatar_url: '',
    banner_url: '',
  });
  const [selectedSpecialRoles, setSelectedSpecialRoles] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [usernameStatus, setUsernameStatus] = useState<'idle' | 'checking' | 'available' | 'taken'>('idle');
  const [emailStatus, setEmailStatus] = useState<'idle' | 'checking' | 'available' | 'taken'>('idle');
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [uploadingBanner, setUploadingBanner] = useState(false);
  const debouncedUsername = useDebounce(formData.username, 500);
  const debouncedEmail = useDebounce(formData.email, 500);

  // Fetch active special roles
  const { data: specialRolesData } = useQuery({
    queryKey: ['admin-special-roles-active'],
    queryFn: () => adminSpecialRolesApi.getActiveSpecialRoles(),
    enabled: open,
  });

  const specialRoles = specialRolesData?.data || [];

  // Fetch user's current special roles when editing
  const { data: userSpecialRolesData } = useQuery({
    queryKey: ['admin-user-special-roles', user?.id],
    queryFn: () => adminSpecialRolesApi.getUserSpecialRoles(user!.id),
    enabled: open && isEdit && !!user?.id,
  });

  // Check username availability
  useEffect(() => {
    const checkUsername = async () => {
      if (!debouncedUsername || debouncedUsername.length < 3) {
        setUsernameStatus('idle');
        return;
      }
      // Skip check if username hasn't changed from original
      if (isEdit && user && debouncedUsername === user.username) {
        setUsernameStatus('idle');
        return;
      }
      setUsernameStatus('checking');
      try {
        const result = await adminUsersApi.checkUsername(debouncedUsername, user?.id);
        setUsernameStatus(result.data?.available ? 'available' : 'taken');
      } catch {
        setUsernameStatus('idle');
      }
    };
    checkUsername();
  }, [debouncedUsername, isEdit, user]);

  // Check email availability
  useEffect(() => {
    const checkEmail = async () => {
      if (!debouncedEmail || !debouncedEmail.includes('@')) {
        setEmailStatus('idle');
        return;
      }
      // Skip check if email hasn't changed from original
      if (isEdit && user && debouncedEmail === user.email) {
        setEmailStatus('idle');
        return;
      }
      setEmailStatus('checking');
      try {
        const result = await adminUsersApi.checkEmail(debouncedEmail, user?.id);
        setEmailStatus(result.data?.available ? 'available' : 'taken');
      } catch {
        setEmailStatus('idle');
      }
    };
    checkEmail();
  }, [debouncedEmail, isEdit, user]);

  // Fetch classes for dropdown
  const { data: classesData } = useQuery({
    queryKey: ['admin-classes-dropdown'],
    queryFn: () => adminClassesApi.getClasses({ limit: 100 }),
    enabled: open,
  });

  const classes = classesData?.data || [];

  // Reset form when modal opens/closes or user changes
  useEffect(() => {
    if (open) {
      setUsernameStatus('idle');
      setEmailStatus('idle');
      setShowPassword(false);
      if (user) {
        setFormData({
          nama: user.nama || '',
          username: user.username || '',
          email: user.email || '',
          password: '',
          role: user.role || 'student',
          nisn: user.nisn || '',
          nis: user.nis || '',
          kelas_id: user.kelas?.id || '',
          tahun_masuk: user.tahun_masuk || new Date().getFullYear(),
          tahun_lulus: user.tahun_lulus || null,
          avatar_url: user.avatar_url || '',
          banner_url: user.banner_url || '',
        });
      } else {
        setFormData({
          nama: '',
          username: '',
          email: '',
          password: '',
          role: 'student',
          nisn: '',
          nis: '',
          kelas_id: '',
          tahun_masuk: new Date().getFullYear(),
          tahun_lulus: null,
          avatar_url: '',
          banner_url: '',
        });
        setSelectedSpecialRoles([]);
      }
    }
  }, [open, user]);

  // Update selected special roles when user data is loaded
  useEffect(() => {
    if (userSpecialRolesData?.data) {
      setSelectedSpecialRoles(userSpecialRolesData.data.map((sr: SpecialRole) => sr.id));
    }
  }, [userSpecialRolesData]);

  // Handle avatar upload
  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
      toast.error('File harus berupa gambar');
      return;
    }

    if (file.size > 5 * 1024 * 1024) {
      toast.error('Ukuran file maksimal 5MB');
      return;
    }

    setUploadingAvatar(true);
    try {
      const url = await uploadsApi.uploadFile(file, 'avatar');
      setFormData(prev => ({ ...prev, avatar_url: url }));
      toast.success('Avatar berhasil diupload');
    } catch {
      toast.error('Gagal mengupload avatar');
    } finally {
      setUploadingAvatar(false);
    }
  };

  // Handle banner upload
  const handleBannerUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
      toast.error('File harus berupa gambar');
      return;
    }

    if (file.size > 10 * 1024 * 1024) {
      toast.error('Ukuran file maksimal 10MB');
      return;
    }

    setUploadingBanner(true);
    try {
      const url = await uploadsApi.uploadFile(file, 'banner');
      setFormData(prev => ({ ...prev, banner_url: url }));
      toast.success('Banner berhasil diupload');
    } catch {
      toast.error('Gagal mengupload banner');
    } finally {
      setUploadingBanner(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Prevent submission if username is taken
    if (usernameStatus === 'taken') {
      toast.error('Username sudah digunakan');
      return;
    }
    
    // Prevent submission if email is taken
    if (emailStatus === 'taken') {
      toast.error('Email sudah digunakan');
      return;
    }
    
    setIsSubmitting(true);

    try {
      if (isEdit && user) {
        const updateData: Record<string, unknown> = {
          nama: formData.nama,
          role: formData.role,
          nisn: formData.nisn || null,
          nis: formData.nis || null,
          kelas_id: formData.role === 'alumni' ? null : (formData.kelas_id || null),
          tahun_masuk: formData.tahun_masuk,
          tahun_lulus: formData.role === 'alumni' ? formData.tahun_lulus : null,
          avatar_url: formData.avatar_url || null,
          banner_url: formData.banner_url || null,
        };
        // Include username if changed
        if (formData.username !== user.username) {
          updateData.username = formData.username;
        }
        // Include email if changed
        if (formData.email !== user.email) {
          updateData.email = formData.email;
        }
        await adminUsersApi.updateUser(user.id, updateData);
        
        // Reset password separately if provided
        if (formData.password) {
          await adminUsersApi.resetPassword(user.id, formData.password);
        }
        
        // Update special roles
        await adminSpecialRolesApi.updateUserSpecialRoles(user.id, { special_role_ids: selectedSpecialRoles });
        
        toast.success('User berhasil diperbarui');
      } else {
        await adminUsersApi.createUser({
          nama: formData.nama,
          username: formData.username,
          email: formData.email,
          password: formData.password,
          role: formData.role,
          nisn: formData.nisn || undefined,
          nis: formData.nis || undefined,
          tahun_masuk: formData.tahun_masuk,
          avatar_url: formData.avatar_url || undefined,
          banner_url: formData.banner_url || undefined,
        } as Parameters<typeof adminUsersApi.createUser>[0]);
        toast.success('User berhasil dibuat');
      }
      onSuccess();
      onOpenChange(false);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: { message?: string } } } };
      toast.error(err.response?.data?.error?.message || (isEdit ? 'Gagal memperbarui user' : 'Gagal membuat user'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit User' : 'Tambah User Baru'}</DialogTitle>
          <DialogDescription>
            {isEdit ? 'Perbarui informasi user' : 'Buat akun user baru'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="nama">Nama Lengkap *</Label>
              <Input
                id="nama"
                value={formData.nama}
                onChange={(e) => setFormData({ ...formData, nama: e.target.value })}
                placeholder="John Doe"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="username">Username *</Label>
              <div className="relative">
                <Input
                  id="username"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '') })}
                  placeholder="johndoe"
                  required
                  className="pr-10"
                />
                {formData.username.length >= 3 && (
                  <div className="absolute right-3 top-1/2 -translate-y-1/2">
                    {usernameStatus === 'checking' && (
                      <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                    )}
                    {usernameStatus === 'available' && (
                      <Check className="h-4 w-4 text-green-500" />
                    )}
                    {usernameStatus === 'taken' && (
                      <AlertCircle className="h-4 w-4 text-destructive" />
                    )}
                  </div>
                )}
              </div>
              {usernameStatus === 'taken' && (
                <p className="text-xs text-destructive">Username sudah digunakan</p>
              )}
              {usernameStatus === 'available' && (
                <p className="text-xs text-green-600">Username tersedia</p>
              )}
              {formData.username.length > 0 && formData.username.length < 3 && (
                <p className="text-xs text-muted-foreground">Minimal 3 karakter</p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="email">Email *</Label>
            <div className="relative">
              <Input
                id="email"
                type="email"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                placeholder="john@example.com"
                required
                className="pr-10"
              />
              {formData.email.includes('@') && (
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  {emailStatus === 'checking' && (
                    <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                  )}
                  {emailStatus === 'available' && (
                    <Check className="h-4 w-4 text-green-500" />
                  )}
                  {emailStatus === 'taken' && (
                    <AlertCircle className="h-4 w-4 text-destructive" />
                  )}
                </div>
              )}
            </div>
            {emailStatus === 'taken' && (
              <p className="text-xs text-destructive">Email sudah digunakan</p>
            )}
            {emailStatus === 'available' && (
              <p className="text-xs text-green-600">Email tersedia</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="password">
              Password {isEdit ? '(kosongkan jika tidak diubah)' : '*'}
            </Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                placeholder="••••••••"
                required={!isEdit}
                minLength={8}
                className="pr-10"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showPassword ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            </div>
            {!isEdit && (
              <p className="text-xs text-muted-foreground">Minimal 8 karakter</p>
            )}
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="role">Role *</Label>
              <Select
                value={formData.role}
                onValueChange={(v) => setFormData({ ...formData, role: v as UserRole })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="student">Siswa</SelectItem>
                  <SelectItem value="alumni">Alumni</SelectItem>
                  <SelectItem value="admin">Admin</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="kelas">Kelas</Label>
              <Select
                value={formData.role === 'alumni' ? 'none' : (formData.kelas_id || 'none')}
                onValueChange={(v) => setFormData({ ...formData, kelas_id: v === 'none' ? '' : v })}
                disabled={formData.role === 'alumni'}
              >
                <SelectTrigger className={formData.role === 'alumni' ? 'bg-muted' : ''}>
                  <SelectValue placeholder="Pilih kelas" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Tidak ada</SelectItem>
                  {classes.map((kelas) => (
                    <SelectItem key={kelas.id} value={kelas.id}>
                      {kelas.nama}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {formData.role === 'alumni' && (
                <p className="text-xs text-muted-foreground">Alumni tidak memiliki kelas aktif</p>
              )}
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="nisn">NISN</Label>
              <Input
                id="nisn"
                value={formData.nisn}
                onChange={(e) => setFormData({ ...formData, nisn: e.target.value })}
                placeholder="0098115881"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="nis">NIS</Label>
              <Input
                id="nis"
                value={formData.nis}
                onChange={(e) => setFormData({ ...formData, nis: e.target.value })}
                placeholder="25491/02000"
              />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="tahun_masuk">Tahun Masuk</Label>
              <Input
                id="tahun_masuk"
                type="number"
                value={formData.tahun_masuk}
                onChange={(e) => setFormData({ ...formData, tahun_masuk: parseInt(e.target.value) || new Date().getFullYear() })}
                min={2000}
                max={2100}
              />
            </div>
            {formData.role === 'alumni' && (
              <div className="space-y-2">
                <Label htmlFor="tahun_lulus">Tahun Lulus</Label>
                <Input
                  id="tahun_lulus"
                  type="number"
                  value={formData.tahun_lulus || ''}
                  onChange={(e) => setFormData({ ...formData, tahun_lulus: e.target.value ? parseInt(e.target.value) : null })}
                  placeholder={String(formData.tahun_masuk + 3)}
                  min={2000}
                  max={2100}
                />
                <p className="text-xs text-muted-foreground">
                  Default: tahun masuk + 3 ({formData.tahun_masuk + 3})
                </p>
              </div>
            )}
          </div>

          {/* Special Roles (only for edit mode and non-admin users) */}
          {isEdit && formData.role !== 'admin' && specialRoles.length > 0 && (
            <div className="space-y-3 rounded-lg border p-4">
              <div>
                <h4 className="text-sm font-medium">Special Roles</h4>
                <p className="text-xs text-muted-foreground">
                  Berikan akses admin terbatas sesuai role yang dipilih
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                {specialRoles.map((role) => {
                  const isSelected = selectedSpecialRoles.includes(role.id);
                  return (
                    <button
                      key={role.id}
                      type="button"
                      onClick={() => {
                        setSelectedSpecialRoles(prev =>
                          isSelected
                            ? prev.filter(id => id !== role.id)
                            : [...prev, role.id]
                        );
                      }}
                      className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium transition-all ${
                        isSelected
                          ? 'ring-2 ring-offset-2'
                          : 'opacity-60 hover:opacity-100'
                      }`}
                      style={{
                        backgroundColor: generateBgColor(role.color),
                        color: role.color,
                        ...(isSelected && { ringColor: role.color }),
                      }}
                    >
                      {isSelected && <Check className="h-3 w-3" />}
                      {role.nama}
                    </button>
                  );
                })}
              </div>
              {selectedSpecialRoles.length > 0 && (
                <p className="text-xs text-muted-foreground">
                  {selectedSpecialRoles.length} role dipilih - User akan mendapat akses ke Admin Panel
                </p>
              )}
            </div>
          )}

          {/* Avatar & Banner Upload */}
          <div className="space-y-4 rounded-lg border p-4">
            <h4 className="text-sm font-medium">Foto Profil</h4>
            <div className="grid gap-4 sm:grid-cols-2">
              {/* Avatar */}
              <div className="space-y-2">
                <Label>Avatar</Label>
                <div className="flex items-center gap-3">
                  <Avatar className="h-16 w-16 border">
                    <AvatarImage src={formData.avatar_url} />
                    <AvatarFallback>
                      <ImageIcon className="h-6 w-6 text-muted-foreground" />
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex-1">
                    <label className="cursor-pointer">
                      <input
                        type="file"
                        accept="image/*"
                        onChange={handleAvatarUpload}
                        className="hidden"
                        disabled={uploadingAvatar}
                      />
                      <Button type="button" variant="outline" size="sm" asChild disabled={uploadingAvatar}>
                        <span>
                          {uploadingAvatar ? (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          ) : (
                            <Upload className="mr-2 h-4 w-4" />
                          )}
                          Upload
                        </span>
                      </Button>
                    </label>
                    {formData.avatar_url && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="ml-2 text-destructive"
                        onClick={() => setFormData(prev => ({ ...prev, avatar_url: '' }))}
                      >
                        Hapus
                      </Button>
                    )}
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">Maks 5MB, format JPG/PNG</p>
              </div>

              {/* Banner */}
              <div className="space-y-2">
                <Label>Banner</Label>
                <div className="relative h-16 w-full overflow-hidden rounded-md border bg-muted">
                  {formData.banner_url ? (
                    <Image
                      src={formData.banner_url}
                      alt="Banner"
                      fill
                      className="object-cover"
                    />
                  ) : (
                    <div className="flex h-full items-center justify-center">
                      <ImageIcon className="h-6 w-6 text-muted-foreground" />
                    </div>
                  )}
                </div>
                <div className="flex gap-2">
                  <label className="cursor-pointer">
                    <input
                      type="file"
                      accept="image/*"
                      onChange={handleBannerUpload}
                      className="hidden"
                      disabled={uploadingBanner}
                    />
                    <Button type="button" variant="outline" size="sm" asChild disabled={uploadingBanner}>
                      <span>
                        {uploadingBanner ? (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                          <Upload className="mr-2 h-4 w-4" />
                        )}
                        Upload
                      </span>
                    </Button>
                  </label>
                  {formData.banner_url && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="text-destructive"
                      onClick={() => setFormData(prev => ({ ...prev, banner_url: '' }))}
                    >
                      Hapus
                    </Button>
                  )}
                </div>
                <p className="text-xs text-muted-foreground">Maks 10MB, format JPG/PNG</p>
              </div>
            </div>
          </div>

          <DialogFooter className="gap-2 sm:gap-0">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Batal
            </Button>
            <Button 
              type="submit" 
              disabled={isSubmitting || usernameStatus === 'checking' || usernameStatus === 'taken' || emailStatus === 'checking' || emailStatus === 'taken' || uploadingAvatar || uploadingBanner}
            >
              {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isEdit ? 'Simpan Perubahan' : 'Buat User'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
