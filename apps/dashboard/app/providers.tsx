"use client";

import { createContext, useState, useMemo, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";
import { createApiClient } from "@/lib/api-client";
import type { paths } from "@/lib/api-types.gen";
import createClient from "openapi-fetch";

interface OrgContextValue {
  activeOrgId: string | null;
  setActiveOrgId: (id: string) => void;
  apiClient: ReturnType<typeof createClient<paths>>;
}

export const OrgContext = createContext<OrgContextValue | null>(null);

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60 * 1000, // 1 minute
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

function getQueryClient() {
  if (typeof window === "undefined") {
    return makeQueryClient();
  }
  if (!browserQueryClient) {
    browserQueryClient = makeQueryClient();
  }
  return browserQueryClient;
}

interface ProvidersProps {
  children: ReactNode;
}

export function Providers({ children }: ProvidersProps) {
  const queryClient = getQueryClient();
  const [activeOrgId, setActiveOrgId] = useState<string | null>(null);

  const apiClient = useMemo(
    () => createApiClient(activeOrgId ?? undefined),
    [activeOrgId]
  );

  const orgContextValue = useMemo<OrgContextValue>(
    () => ({
      activeOrgId,
      setActiveOrgId,
      apiClient,
    }),
    [activeOrgId, apiClient]
  );

  return (
    <QueryClientProvider client={queryClient}>
      <OrgContext.Provider value={orgContextValue}>
        {children}
        <Toaster position="bottom-right" richColors />
      </OrgContext.Provider>
    </QueryClientProvider>
  );
}
