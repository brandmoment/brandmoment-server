"use client";

import type { createAuthClient as CreateAuthClient } from "better-auth/react";

type AuthClient = ReturnType<typeof CreateAuthClient>;

let _client: AuthClient | undefined;

export function getAuthClient(): AuthClient {
  if (!_client) {
    // Dynamic require to avoid module-level localStorage access during SSR
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { createAuthClient } = require("better-auth/react");
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { organizationClient } = require("better-auth/client/plugins");
    _client = createAuthClient({
      baseURL: process.env["NEXT_PUBLIC_APP_URL"] ?? "http://localhost:3000",
      plugins: [organizationClient()],
    }) as AuthClient;
  }
  return _client;
}

// Re-export convenience accessors that lazy-init on first call
export const authClient = new Proxy({} as AuthClient, {
  get(_, prop: string | symbol) {
    return Reflect.get(getAuthClient(), prop);
  },
});

export function useSession() {
  return getAuthClient().useSession();
}
