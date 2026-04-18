"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Plus, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useCampaigns } from "@/hooks/useCampaigns";
import { CampaignStatusBadge } from "@/components/campaign/CampaignStatusBadge";
import { CreateCampaignDialog } from "@/components/campaign/CreateCampaignDialog";
import type { CampaignStatus } from "@/types/campaign";

const PAGE_SIZES = [20, 50, 100] as const;
type PageSize = (typeof PAGE_SIZES)[number];

const STATUS_OPTIONS: { value: CampaignStatus | "all"; label: string }[] = [
  { value: "all", label: "All statuses" },
  { value: "draft", label: "Draft" },
  { value: "active", label: "Active" },
  { value: "paused", label: "Paused" },
  { value: "completed", label: "Completed" },
];

function formatDate(iso: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function formatBudget(cents: number | null, currency: string) {
  if (cents === null) return "—";
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
    minimumFractionDigits: 0,
  }).format(cents / 100);
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

export function CampaignsList() {
  const router = useRouter();
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState<PageSize>(20);
  const [statusFilter, setStatusFilter] = useState<CampaignStatus | "all">("all");
  const [createOpen, setCreateOpen] = useState(false);

  const offset = page * pageSize;
  const { data, isLoading, isError } = useCampaigns({
    limit: pageSize,
    offset,
    status: statusFilter === "all" ? undefined : statusFilter,
  });

  const totalPages = data ? Math.ceil(data.total / pageSize) : 0;

  function handleRowClick(id: string) {
    router.push(`/campaigns/${id}`);
  }

  function handleStatusChange(value: string) {
    setStatusFilter(value as CampaignStatus | "all");
    setPage(0);
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Campaigns</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Create and manage your advertising campaigns
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Campaign
        </Button>
      </div>

      <Card>
        <CardHeader className="px-6 py-4">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">
              {data ? `${data.total} campaign${data.total !== 1 ? "s" : ""}` : "Campaigns"}
            </CardTitle>
            <Select value={statusFilter} onValueChange={handleStatusChange}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="Filter by status" />
              </SelectTrigger>
              <SelectContent>
                {STATUS_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="px-6 pb-6">
              <TableSkeleton />
            </div>
          ) : isError ? (
            <div className="px-6 pb-6 text-sm text-destructive">
              Failed to load campaigns. Please refresh.
            </div>
          ) : !data || data.items.length === 0 ? (
            <div className="px-6 pb-10 pt-6 text-center">
              <p className="text-muted-foreground text-sm">
                {statusFilter !== "all"
                  ? `No ${statusFilter} campaigns found.`
                  : "No campaigns yet."}
              </p>
              {statusFilter === "all" && (
                <Button
                  variant="outline"
                  className="mt-4"
                  onClick={() => setCreateOpen(true)}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Create your first campaign
                </Button>
              )}
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
                      Status
                    </th>
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      Budget
                    </th>
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      Start Date
                    </th>
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      End Date
                    </th>
                    <th className="px-6 py-3 text-left font-medium text-muted-foreground">
                      Created
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {data.items.map((campaign) => (
                    <tr
                      key={campaign.id}
                      className="cursor-pointer hover:bg-muted/30 transition-colors"
                      onClick={() => handleRowClick(campaign.id)}
                    >
                      <td className="px-6 py-3 font-medium">{campaign.name}</td>
                      <td className="px-6 py-3">
                        <CampaignStatusBadge status={campaign.status} />
                      </td>
                      <td className="px-6 py-3 text-muted-foreground">
                        {formatBudget(campaign.budget_cents, campaign.currency)}
                      </td>
                      <td className="px-6 py-3 text-muted-foreground">
                        {formatDate(campaign.start_date)}
                      </td>
                      <td className="px-6 py-3 text-muted-foreground">
                        {formatDate(campaign.end_date)}
                      </td>
                      <td className="px-6 py-3 text-muted-foreground">
                        {formatDate(campaign.created_at)}
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

      <CreateCampaignDialog open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  );
}
