'use client';

import { useAuthStore } from '@/lib/stores/auth-store';
import { HeroSection, AboutSection, FaqSection, TopStudentsSection, TopProjectsSection } from '@/components/landing';
import { SmartFeedList } from '@/components/feed/smart-feed-list';

export default function HomePage() {
  const { isAuthenticated } = useAuthStore();

  if (!isAuthenticated) {
    return (
      <>
        <HeroSection />
        <AboutSection />
        <TopStudentsSection />
        <TopProjectsSection />
        <FaqSection />
      </>
    );
  }

  return <SmartFeedList />;
}
