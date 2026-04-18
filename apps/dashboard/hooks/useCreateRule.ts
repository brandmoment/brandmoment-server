"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { PublisherRule, CreatePublisherRuleRequest } from "@/types/publisher-rule";

interface CreateRuleVariables {
  appId: string;
  body: CreatePublisherRuleRequest;
}

export function useCreateRule() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<PublisherRule, Error, CreateRuleVariables>({
    mutationFn: async ({ appId, body }) => {
      const { data, error } = await apiClient.POST(
        "/v1/publisher-apps/{id}/rules",
        {
          params: { path: { id: appId } },
          body,
        }
      );
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to create rule"
        );
      }
      return data.data;
    },
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ["publisher-rules", variables.appId],
      });
    },
  });
}
