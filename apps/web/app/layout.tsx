import type { Metadata } from 'next';
import { Inter, JetBrains_Mono } from 'next/font/google';
import './globals.css';
import { QueryProvider } from '@/components/providers/query-provider';
import { AuthProvider } from '@/components/providers/auth-provider';
import { DMProvider } from '@/components/providers/dm-provider';
import { Toaster } from '@/components/ui/sonner';
import { FeedbackButton } from '@/components/layout/feedback-button';

const inter = Inter({
  variable: '--font-sans',
  subsets: ['latin'],
});

const jetbrainsMono = JetBrains_Mono({
  variable: '--font-mono',
  subsets: ['latin'],
});

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'),
  title: {
    default: 'Grafikarsa - Platform Portofolio SMKN 4 Malang',
    template: '%s | Grafikarsa',
  },
  description: 'Platform Katalog Portofolio & Social Network Warga SMKN 4 Malang',
  icons: {
    icon: [
      {
        media: '(prefers-color-scheme: light)',
        url: '/images/logos/logo_black.svg',
        href: '/images/logos/logo_black.svg',
      },
      {
        media: '(prefers-color-scheme: dark)',
        url: '/images/logos/logo_white.svg',
        href: '/images/logos/logo_white.svg',
      },
    ],
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="id" suppressHydrationWarning>
      <body className={`${inter.variable} ${jetbrainsMono.variable} font-sans antialiased`}>
        <QueryProvider>
          <AuthProvider>
            <DMProvider>
              {children}
            </DMProvider>
            <FeedbackButton />
            <Toaster position="top-center" richColors />
          </AuthProvider>
        </QueryProvider>
      </body>
    </html>
  );
}
