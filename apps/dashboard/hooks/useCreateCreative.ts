"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { Creative, CreateCreativeRequest } from "@/types/campaign";

interface CreateCreativeVariables {
  campaignId: string;
  body: CreateCreativeRequest;
}

export function useCreateCreative() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<Creative, Error, CreateCreativeVariables>({
    mutationFn: async ({ campaignId, body }) => {
      const { data, error } = await apiClient.POST("/v1/campaigns/{id}/creatives", {
        params: { path: { id: campaignId } },
        body,
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to create creative"
        );
      }
      return data.data;
    },
    onSuccess: (_data, { campaignId }) => {
      void queryClient.invalidateQueries({ queryKey: ["creatives", campaignId] });
    },
  });
}
