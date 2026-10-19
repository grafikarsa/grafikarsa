import FollowListPage from "@/components/user/follow-list";

export const metadata = { title: "Mengikuti — Grafikarsa" };

export default function FollowingPage() {
  return <FollowListPage kind="following" />;
}
