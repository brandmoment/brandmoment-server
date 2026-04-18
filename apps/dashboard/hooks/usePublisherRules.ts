"use client";

import { useQuery } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { PublisherRuleListResponse } from "@/types/publisher-rule";

interface UsePublisherRulesParams {
  appId: string;
  limit?: number;
  offset?: number;
}

export function usePublisherRules({
  appId,
  limit = 20,
  offset = 0,
}: UsePublisherRulesParams) {
  const { apiClient } = useActiveOrg();

  return useQuery<PublisherRuleListResponse>({
    queryKey: ["publisher-rules", appId, { limit, offset }],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        "/v1/publisher-apps/{id}/rules",
        {
          params: {
            path: { id: appId },
            query: { limit, offset },
          },
        }
      );
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to fetch rules"
        );
      }
      return data.data;
    },
    enabled: Boolean(appId),
  });
}
