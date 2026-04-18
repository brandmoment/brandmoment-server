"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { CreateAPIKeyRequest, CreateAPIKeyResponse } from "@/types/api-key";

interface CreateApiKeyVariables {
  appId: string;
  body: CreateAPIKeyRequest;
}

export function useCreateApiKey() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<CreateAPIKeyResponse, Error, CreateApiKeyVariables>({
    mutationFn: async ({ appId, body }) => {
      const { data, error } = await apiClient.POST(
        "/v1/publisher-apps/{id}/api-keys",
        {
          params: { path: { id: appId } },
          body,
        }
      );
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to create API key"
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
