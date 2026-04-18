"use client";

import { useQuery } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { PublisherApp } from "@/types/publisher-app";

export function usePublisherApp(id: string) {
  const { apiClient } = useActiveOrg();

  return useQuery<PublisherApp>({
    queryKey: ["publisher-apps", id],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/v1/publisher-apps/{id}", {
        params: { path: { id } },
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to fetch app"
        );
      }
      return data.data;
    },
    enabled: Boolean(id),
  });
}
