import Link from 'next/link';
import Image from 'next/image';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { UserCard as UserCardType } from '@/lib/types';

interface UserCardProps {
  user: UserCardType;
}

export function UserCard({ user }: UserCardProps) {
  const { username, nama, avatar_url, banner_url, role, kelas, jurusan, tahun_masuk, tahun_lulus } = user;

  const getInitials = (name: string) => {
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  const getRoleLabel = (role: string) => {
    switch (role) {
      case 'student':
        return 'Siswa';
      case 'alumni':
        return 'Alumni';
      case 'admin':
        return 'Admin';
      default:
        return role;
    }
  };

  return (
    <Link href={`/${username}`}>
      <div className="group overflow-hidden rounded-xl border bg-card transition-all hover:shadow-lg hover:-translate-y-1">
        {/* Banner */}
        <div className="relative h-20 w-full bg-gradient-to-br from-primary/20 via-primary/10 to-background sm:h-24">
          {banner_url && (
            <Image
              src={banner_url}
              alt=""
              fill
              className="object-cover"
              unoptimized
            />
          )}
        </div>

        {/* Avatar - overlapping banner */}
        <div className="relative px-3 sm:px-4">
          <Avatar className="-mt-8 h-16 w-16 border-4 border-background shadow-md sm:-mt-10 sm:h-20 sm:w-20">
            <AvatarImage src={avatar_url || undefined} alt={nama} />
            <AvatarFallback className="bg-primary/10 text-lg font-semibold sm:text-xl">
              {getInitials(nama)}
            </AvatarFallback>
          </Avatar>
        </div>

        {/* Content */}
        <div className="p-3 pt-2 sm:p-4 sm:pt-2">
          <h3 className="truncate text-sm font-semibold text-foreground transition-colors group-hover:text-primary sm:text-base">
            {nama}
          </h3>
          <p className="truncate text-xs text-muted-foreground sm:text-sm">@{username}</p>

          {/* Info */}
          <div className="mt-2.5 flex flex-wrap items-center gap-1.5 sm:mt-3">
            <Badge variant="secondary" className="px-1.5 py-0 text-[10px] sm:px-2.5 sm:py-0.5 sm:text-xs">
              {getRoleLabel(role)}
            </Badge>
            {role === 'student' && kelas && (
              <Badge variant="outline" className="px-1.5 py-0 text-[10px] sm:px-2.5 sm:py-0.5 sm:text-xs">
                {kelas.nama}
              </Badge>
            )}
            {role === 'alumni' && tahun_masuk && tahun_lulus && (
              <Badge variant="outline" className="px-1.5 py-0 text-[10px] sm:px-2.5 sm:py-0.5 sm:text-xs">
                {tahun_masuk}-{tahun_lulus}
              </Badge>
            )}
            {jurusan && (
              <Badge variant="outline" className="px-1.5 py-0 text-[10px] sm:px-2.5 sm:py-0.5 sm:text-xs">
                {jurusan.kode || jurusan.nama}
              </Badge>
            )}
          </div>
        </div>
      </div>
    </Link>
  );
}
