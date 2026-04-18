"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { Campaign, UpdateCampaignRequest } from "@/types/campaign";

interface UpdateCampaignVariables {
  id: string;
  body: UpdateCampaignRequest;
}

export function useUpdateCampaign() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<Campaign, Error, UpdateCampaignVariables>({
    mutationFn: async ({ id, body }) => {
      const { data, error } = await apiClient.PUT("/v1/campaigns/{id}", {
        params: { path: { id } },
        body,
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to update campaign"
        );
      }
      return data.data;
    },
    onSuccess: (_data, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ["campaign", id] });
      void queryClient.invalidateQueries({ queryKey: ["campaigns"] });
    },
  });
}
