import FollowListPage from "@/components/user/follow-list";

export const metadata = { title: "Pengikut — Grafikarsa" };

export default function FollowersPage() {
  return <FollowListPage kind="followers" />;
}
