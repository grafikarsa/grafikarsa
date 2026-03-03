import { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getPortfolioDetail } from '@/lib/api/server-fetch';
import { PortfolioClient } from './portfolio-client';

interface PortfolioDetailPageProps {
  params: Promise<{ username: string; slug: string }>;
}

export async function generateMetadata({ params }: PortfolioDetailPageProps): Promise<Metadata> {
  const { username, slug } = await params;
  const portfolio = await getPortfolioDetail(username, slug);

  if (!portfolio) {
    return {
      title: 'Portfolio Not Found',
    };
  }

  const title = `${portfolio.judul} | Grafikarsa`;

  // Try to find first text block for description
  const firstTextBlock = portfolio.content_blocks?.find((b: any) => b.block_type === 'text');
  const description = firstTextBlock?.payload?.content?.substring(0, 160) ||
    `Karya portofolio oleh ${portfolio.user?.nama} di Grafikarsa - Platform Portofolio SMKN 4 Malang.`;

  return {
    title,
    description,
    openGraph: {
      title,
      description,
      type: 'article',
      authors: [portfolio.user?.nama],
      images: [
        {
          url: `/${username}/${slug}/opengraph-image`,
          width: 1200,
          height: 630,
          alt: portfolio.judul,
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

export default async function PortfolioDetailPage({ params }: PortfolioDetailPageProps) {
  const { username, slug } = await params;
  const portfolio = await getPortfolioDetail(username, slug);

  return <PortfolioClient username={username} slug={slug} initialData={portfolio as any} />;
}
