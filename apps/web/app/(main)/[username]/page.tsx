import { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getUserProfile } from '@/lib/api/server-fetch';
import { ProfileClient } from './profile-client';

interface ProfilePageProps {
  params: Promise<{ username: string }>;
}

export async function generateMetadata({ params }: ProfilePageProps): Promise<Metadata> {
  const { username } = await params;
  const user = await getUserProfile(username);

  if (!user) {
    return {
      title: 'User Not Found',
    };
  }

  const title = `${user.nama} (@${user.username}) | Grafikarsa`;
  const description = user.bio || `Cek portofolio dan karya ${user.nama} di Grafikarsa - Platform Portofolio SMKN 4 Malang.`;

  return {
    title,
    description,
    openGraph: {
      title,
      description,
      type: 'profile',
      username: user.username,
      images: [
        {
          url: `/${user.username}/opengraph-image`,
          width: 1200,
          height: 630,
          alt: user.nama,
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title,
      description,
    },
  };
}

export default async function ProfilePage({ params }: ProfilePageProps) {
  const { username } = await params;
  const user = await getUserProfile(username);

  return <ProfileClient username={username} initialData={user as any} />;
}
