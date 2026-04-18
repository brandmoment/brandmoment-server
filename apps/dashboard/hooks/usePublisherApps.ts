"use client";

import { useQuery } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { PublisherAppListResponse } from "@/types/publisher-app";

interface UsePublisherAppsParams {
  limit?: number;
  offset?: number;
}

export function usePublisherApps({ limit = 20, offset = 0 }: UsePublisherAppsParams = {}) {
  const { apiClient } = useActiveOrg();

  return useQuery<PublisherAppListResponse>({
    queryKey: ["publisher-apps", { limit, offset }],
    queryFn: async () => {
      const { data, error } = await apiClient.GET("/v1/publisher-apps", {
        params: { query: { limit, offset } },
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to fetch apps"
        );
      }
      return data.data;
    },
  });
}
