"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Plus, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { usePublisherApps } from "@/hooks/usePublisherApps";
import { CreateAppDialog } from "@/components/publisher/CreateAppDialog";
import type { PublisherApp } from "@/types/publisher-app";

const PAGE_SIZES = [20, 50, 100] as const;
type PageSize = (typeof PAGE_SIZES)[number];

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function PlatformBadge({ platform }: { platform: PublisherApp["platform"] }) {
  const labels: Record<PublisherApp["platform"], string> = {
    ios: "iOS",
    android: "Android",
    web: "Web",
  };
  return <Badge variant="outline">{labels[platform]}</Badge>;
}

function TableSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 5 }).map((_, i) => (
        <Skeleton key={i} className="h-12 w-full" />
      ))}
    </div>
  );
}

export function AppsList() {
  const router = useRouter();
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState<PageSize>(20);
  const [createOpen, setCreateOpen] = useState(false);

  const offset = page * pageSize;
  const { data, isLoading, isError } = usePublisherApps({ limit: pageSize, offset });

  const totalPages = data ? Math.ceil(data.total / pageSize) : 0;

  function handleRowClick(id: string) {
    router.push(`/apps/${id}`);
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Publisher Apps</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Manage your apps and their SDK integrations
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New App
        </Button>
      </div>

      <Card>
        <CardHeader className="px-6 py-4">
          <CardTitle className="text-base">
            {data ? `${data.total} app${data.total !== 1 ? "s" : ""}` : "Apps"}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="px-6 pb-6">
              <TableSkeleton />
            </div>
          ) : isError ? (
            <div className="px-6 pb-6 text-sm text-destructive">
              Failed to load apps. Please refresh.
            </div>
          ) : !data || data.items.length === 0 ? (
            <div className="px-6 pb-10 pt-6 text-center">
              <p className="text-muted-foreground text-sm">No apps yet.</p>
              <Button
                variant="outline"
                className="mt-4"
                onClick={() => setCreateOpen(true)}
              >
                <Plus className="mr-2 h-4 w-4" />
                Create your first app
              </Button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40">
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      Name
                    </th>
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      Platform
                    </th>
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      Bundle ID
                    </th>
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      Created
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {data.items.map((app) => (
                    <tr
                      key={app.id}
                      className="cursor-pointer hover:bg-muted/30 transition-colors"
                      onClick={() => handleRowClick(app.id)}
                    >
                      <td className="px-6 py-3 font-medium">{app.name}</td>
                      <td className="px-6 py-3">
                        <PlatformBadge platform={app.platform} />
                      </td>
                      <td className="px-6 py-3 font-mono text-xs text-muted-foreground">
                        {app.bundle_id}
                      </td>
                      <td className="px-6 py-3">
                        <Badge variant={app.is_active ? "success" : "outline"}>
                          {app.is_active ? "Active" : "Inactive"}
                        </Badge>
                      </td>
                      <td className="px-6 py-3 text-muted-foreground">
                        {formatDate(app.created_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {data && data.total > 0 && (
            <div className="flex items-center justify-between border-t px-6 py-3">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span>Rows per page:</span>
                <select
                  className="rounded border bg-background px-2 py-1 text-sm"
                  value={pageSize}
                  onChange={(e) => {
                    setPageSize(Number(e.target.value) as PageSize);
                    setPage(0);
                  }}
                >
                  {PAGE_SIZES.map((s) => (
                    <option key={s} value={s}>
                      {s}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex items-center gap-3 text-sm">
                <span className="text-muted-foreground">
                  Page {page + 1} of {totalPages}
                </span>
                <div className="flex gap-1">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => Math.max(0, p - 1))}
                    disabled={page === 0}
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                    disabled={page >= totalPages - 1}
                  >
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <CreateAppDialog open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  );
}
