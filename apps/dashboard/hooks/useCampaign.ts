"use client";

import { useQuery } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { Campaign } from "@/types/campaign";

export function useCampaign(id: string) {
  const { apiClient } = useActiveOrg();

  return useQuery<Campaign>({
    queryKey: ["campaign", id],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/v1/campaigns/{id}", {
        params: { path: { id } },
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to fetch campaign"
        );
      }
      return data.data;
    },
    enabled: Boolean(id),
  });
}
