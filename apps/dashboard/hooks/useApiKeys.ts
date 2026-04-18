"use client";

import { useQuery } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { APIKeyListResponse } from "@/types/api-key";

interface UseApiKeysParams {
  appId: string;
  includeRevoked?: boolean;
}

export function useApiKeys({ appId, includeRevoked = false }: UseApiKeysParams) {
  const { apiClient } = useActiveOrg();

  return useQuery<APIKeyListResponse>({
    queryKey: ["api-keys", appId, { includeRevoked }],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        "/v1/publisher-apps/{id}/api-keys",
        {
          params: {
            path: { id: appId },
            query: { include_revoked: includeRevoked },
          },
        }
      );
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to fetch API keys"
        );
      }
      return data.data;
    },
    enabled: Boolean(appId),
  });
}
