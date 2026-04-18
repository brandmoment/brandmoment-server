"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type { PublisherRule, UpdatePublisherRuleRequest } from "@/types/publisher-rule";

interface UpdateRuleVariables {
  appId: string;
  ruleId: string;
  body: UpdatePublisherRuleRequest;
}

export function useUpdateRule() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<PublisherRule, Error, UpdateRuleVariables>({
    mutationFn: async ({ appId, ruleId, body }) => {
      const { data, error } = await apiClient.PUT(
        "/v1/publisher-apps/{id}/rules/{ruleId}",
        {
          params: { path: { id: appId, ruleId } },
          body,
        }
      );
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to update rule"
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
