"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";

interface DeleteRuleVariables {
  appId: string;
  ruleId: string;
}

export function useDeleteRule() {
  const { apiClient } = useActiveOrg();
  const queryClient = useQueryClient();

  return useMutation<{ id: string }, Error, DeleteRuleVariables>({
    mutationFn: async ({ appId, ruleId }) => {
      const { data, error } = await apiClient.DELETE(
        "/v1/publisher-apps/{id}/rules/{ruleId}",
        {
          params: { path: { id: appId, ruleId } },
        }
      );
      if (error) {
        throw new Error(
          (error as { error?: { message?: string } }).error?.message ??
            "Failed to delete rule"
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
