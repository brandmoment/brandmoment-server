"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { Campaign, CreateCampaignRequest } from "@/types/campaign";

export function useCreateCampaign() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<Campaign, Error, CreateCampaignRequest>({
    mutationFn: async (body) => {
      const { data, error } = await apiClient.POST("/v1/campaigns", { body });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to create campaign"
        );
      }
      return data.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["campaigns"] });
    },
  });
}
