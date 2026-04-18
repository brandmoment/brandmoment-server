"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";

interface RevokeApiKeyVariables {
  appId: string;
  keyId: string;
}

interface RevokeApiKeyResult {
  id: string;
  revoked_at: string;
}

export function useRevokeApiKey() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<RevokeApiKeyResult, Error, RevokeApiKeyVariables>({
    mutationFn: async ({ appId, keyId }) => {
      const { data, error } = await apiClient.DELETE(
        "/v1/publisher-apps/{id}/api-keys/{keyId}",
        {
          params: { path: { id: appId, keyId } },
        }
      );
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to revoke API key"
        );
      }
      return data.data;
    },
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ["api-keys", variables.appId],
      });
    },
  });
}
