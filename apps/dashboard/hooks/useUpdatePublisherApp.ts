"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { PublisherApp, UpdatePublisherAppRequest } from "@/types/publisher-app";

interface UpdatePublisherAppVariables {
  id: string;
  body: UpdatePublisherAppRequest;
}

export function useUpdatePublisherApp() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<PublisherApp, Error, UpdatePublisherAppVariables>({
    mutationFn: async ({ id, body }) => {
      const { data, error } = await apiClient.PUT("/v1/publisher-apps/{id}", {
        params: { path: { id } },
        body,
      });
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to update app"
        );
      }
      return data.data;
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(["publisher-apps", updated.id], updated);
      void queryClient.invalidateQueries({ queryKey: ["publisher-apps"] });
    },
  });
}
