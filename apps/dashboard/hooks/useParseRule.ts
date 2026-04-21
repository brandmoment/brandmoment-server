"use client";

import { useMutation } from "@tanstack/react-query";
import { useActiveOrg } from "@/hooks/useActiveOrg";
import type {
  ParseRuleRequest,
  ParseRuleResponse,
} from "@/types/rule-parser";

const API_BASE_URL =
  process.env["NEXT_PUBLIC_API_URL"] ?? "http://localhost:8080";

export function useParseRule() {
  const { activeOrgId } = useActiveOrg();

  return useMutation<ParseRuleResponse, Error, ParseRuleRequest>({
    mutationFn: async (body) => {
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (activeOrgId) {
        headers["X-Org-ID"] = activeOrgId;
      }

      const response = await fetch(
        `${API_BASE_URL}/v1/publisher-rules/parse`,
        {
          method: "POST",
          headers,
          body: JSON.stringify(body),
        }
      );

      if (response.status === 501) {
        throw new Error(
          "LLM API key not configured. Set OPENAI_API_KEY or GEMINI_API_KEY on the server."
        );
      }

      const json = (await response.json()) as
        | { data: ParseRuleResponse }
        | { error: { code: string; message: string } };

      if (!response.ok) {
        const errJson = json as { error: { code: string; message: string } };
        throw new Error(
          errJson.error?.message ?? `Request failed: ${response.status}`
        );
      }

      return (json as { data: ParseRuleResponse }).data;
    },
  });
}
