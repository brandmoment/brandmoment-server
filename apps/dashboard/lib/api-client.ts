"use client";

import createClient from "openapi-fetch";
import type { paths } from "./api-types.gen";

const API_BASE_URL =
  process.env["NEXT_PUBLIC_API_URL"] ?? "http://localhost:8080";

/**
 * Create a typed API client with an optional X-Org-ID header.
 * The activeOrgId is injected from OrgContext when available.
 */
export function createApiClient(activeOrgId?: string) {
  return createClient<paths>({
    baseUrl: API_BASE_URL,
    headers: {
      "Content-Type": "application/json",
      ...(activeOrgId ? { "X-Org-ID": activeOrgId } : {}),
    },
  });
}

/**
 * Default client — no org context. Use createApiClient(orgId) in components.
 */
export const apiClient = createApiClient();
