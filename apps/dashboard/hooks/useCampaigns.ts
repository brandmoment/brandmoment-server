"use client";

import { useQuery } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { CampaignListResponse, CampaignStatus } from "@/types/campaign";

interface UseCampaignsParams {
  limit?: number;
  offset?: number;
  status?: CampaignStatus;
}

export function useCampaigns({ limit = 20, offset = 0, status }: UseCampaignsParams = {}) {
  const { apiClient } = useActiveOrg();

  return useQuery<CampaignListResponse>({
    queryKey: ["campaigns", { limit, offset, status }],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/v1/campaigns", {
        params: { query: { limit, offset, ...(status ? { status } : {}) } },
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to fetch campaigns"
        );
      }
      return data.data;
    },
  });
}
