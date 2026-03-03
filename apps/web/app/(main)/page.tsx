'use client';

import { useAuthStore } from '@/lib/stores/auth-store';
import { HeroSection, AboutSection, FaqSection, TopStudentsSection, TopProjectsSection } from '@/components/landing';
import { SmartFeedList } from '@/components/feed/smart-feed-list';
import { MaintenancePage } from '@/components/maintenance/maintenance-page';

export default function HomePage() {
  const { isAuthenticated } = useAuthStore();

  // Check for maintenance mode
  const isMaintenanceMode = process.env.NEXT_PUBLIC_MAINTENANCE_MODE === 'true';

  if (isMaintenanceMode) {
    return <MaintenancePage />;
  }

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
