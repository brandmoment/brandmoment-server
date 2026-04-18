import { AppDetail } from "@/components/publisher/AppDetail";

interface AppDetailPageProps {
  params: Promise<{ id: string }>;
}

export default async function AppDetailPage({ params }: AppDetailPageProps) {
  const { id } = await params;
  return <AppDetail appId={id} />;
}
