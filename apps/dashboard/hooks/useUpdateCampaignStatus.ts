"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { Campaign, CampaignStatus } from "@/types/campaign";

interface UpdateCampaignStatusVariables {
  id: string;
  status: CampaignStatus;
}

export function useUpdateCampaignStatus() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<Campaign, Error, UpdateCampaignStatusVariables>({
    mutationFn: async ({ id, status }) => {
      const { data, error } = await apiClient.PATCH("/v1/campaigns/{id}/status", {
        params: { path: { id } },
        body: { status },
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to update campaign status"
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
