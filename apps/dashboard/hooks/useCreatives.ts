"use client";

import { useQuery } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { CreativeListResponse } from "@/types/campaign";

export function useCreatives(campaignId: string) {
  const { apiClient } = useActiveOrg();

  return useQuery<CreativeListResponse>({
    queryKey: ["creatives", campaignId],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/v1/campaigns/{id}/creatives", {
        params: { path: { id: campaignId } },
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to fetch creatives"
        );
      }
      return data.data;
    },
    enabled: Boolean(campaignId),
  });
}
