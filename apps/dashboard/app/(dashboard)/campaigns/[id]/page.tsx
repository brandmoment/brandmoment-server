import { CampaignDetail } from "@/components/campaign/CampaignDetail";

interface CampaignDetailPageProps {
  params: Promise<{ id: string }>;
}

export default async function CampaignDetailPage({ params }: CampaignDetailPageProps) {
  const { id } = await params;
  return <CampaignDetail campaignId={id} />;
}
