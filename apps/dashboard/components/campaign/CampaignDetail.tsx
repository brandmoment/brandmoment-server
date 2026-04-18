"use client";

import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { useCampaign } from "@/hooks/useCampaign";
import { CampaignStatusBadge } from "@/components/campaign/CampaignStatusBadge";
import { CampaignOverviewTab } from "@/components/campaign/CampaignOverviewTab";
import { CreativesList } from "@/components/campaign/CreativesList";

interface CampaignDetailProps {
  campaignId: string;
}

export function CampaignDetail({ campaignId }: CampaignDetailProps) {
  const { data: campaign, isLoading, isError } = useCampaign(campaignId);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-10 w-80" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (isError || !campaign) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" asChild>
          <Link href="/campaigns">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Campaigns
          </Link>
        </Button>
        <p className="text-sm text-destructive">
          Campaign not found or failed to load.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-4">
        <Button variant="ghost" size="sm" asChild>
          <Link href="/campaigns">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Campaigns
          </Link>
        </Button>
      </div>

      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">{campaign.name}</h1>
        <CampaignStatusBadge status={campaign.status} />
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="creatives">Creatives</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-6">
          <CampaignOverviewTab campaign={campaign} />
        </TabsContent>

        <TabsContent value="creatives" className="mt-6">
          <CreativesList campaignId={campaignId} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
