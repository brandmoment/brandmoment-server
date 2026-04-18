"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { PublisherApp, CreatePublisherAppRequest } from "@/types/publisher-app";

export function useCreatePublisherApp() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<PublisherApp, Error, CreatePublisherAppRequest>({
    mutationFn: async (body) => {
      const { data, error } = await apiClient.POST("/v1/publisher-apps", {
        body,
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to create app"
        );
      }
      return data.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["publisher-apps"] });
    },
  });
}
