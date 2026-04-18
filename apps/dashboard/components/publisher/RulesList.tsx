"use client";

import { useState } from "react";
import { Plus, Pencil, Trash2, ChevronLeft, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { usePublisherRules } from "@/hooks/usePublisherRules";
import { useDeleteRule } from "@/hooks/useDeleteRule";
import { useUpdateRule } from "@/hooks/useUpdateRule";
import { RuleEditorDialog } from "@/components/publisher/RuleEditorDialog";
import type { PublisherRule, RuleType } from "@/types/publisher-rule";

const RULE_TYPE_LABELS: Record<RuleType, string> = {
  blocklist: "Blocklist",
  allowlist: "Allowlist",
  frequency_cap: "Frequency Cap",
  geo_filter: "Geo Filter",
  platform_filter: "Platform Filter",
};

function getRuleConfigSummary(rule: PublisherRule): string {
  const cfg = rule.config;
  switch (rule.type) {
    case "blocklist":
    case "allowlist": {
      const domains = (cfg.domains as string[] | undefined) ?? [];
      const bundles = (cfg.bundle_ids as string[] | undefined) ?? [];
      const parts: string[] = [];
      if (domains.length > 0) parts.push(`${domains.length} domain(s)`);
      if (bundles.length > 0) parts.push(`${bundles.length} bundle ID(s)`);
      return parts.join(", ") || "No entries";
    }
    case "frequency_cap":
      return `Max ${cfg.max_impressions as number} per ${cfg.window_seconds as number}s`;
    case "geo_filter":
      return `${cfg.mode as string}: ${((cfg.country_codes as string[]) ?? []).join(", ") || "none"}`;
    case "platform_filter":
      return `${cfg.mode as string}: ${((cfg.platforms as string[]) ?? []).join(", ") || "none"}`;
    default:
      return "";
  }
}

interface DeleteConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  ruleType: RuleType | null;
  onConfirm: () => void;
  isPending: boolean;
}

function DeleteConfirmDialog({
  open,
  onOpenChange,
  ruleType,
  onConfirm,
  isPending,
}: DeleteConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete Rule</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete this{" "}
            {ruleType ? RULE_TYPE_LABELS[ruleType] : "rule"}? This action
            cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isPending}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? "Deleting..." : "Delete rule"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface RulesListProps {
  appId: string;
}

export function RulesList({ appId }: RulesListProps) {
  const [page, setPage] = useState(0);
  const PAGE_SIZE = 20;
  const offset = page * PAGE_SIZE;

  const [addOpen, setAddOpen] = useState(false);
  const [editRule, setEditRule] = useState<PublisherRule | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<PublisherRule | null>(null);

  const { data, isLoading, isError } = usePublisherRules({
    appId,
    limit: PAGE_SIZE,
    offset,
  });
  const { mutateAsync: deleteRule, isPending: isDeleting } = useDeleteRule();
  const { mutateAsync: updateRule } = useUpdateRule();

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;

  async function handleDeleteConfirm() {
    if (!deleteTarget) return;
    try {
      await deleteRule({ appId, ruleId: deleteTarget.id });
      toast.success("Rule deleted");
      setDeleteTarget(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete rule");
    }
  }

  async function handleToggleActive(rule: PublisherRule) {
    try {
      await updateRule({
        appId,
        ruleId: rule.id,
        body: { is_active: !rule.is_active },
      });
      toast.success(rule.is_active ? "Rule deactivated" : "Rule activated");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update rule");
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Rules control which ads are shown in your app.
        </p>
        <Button size="sm" onClick={() => setAddOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add rule
        </Button>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      ) : isError ? (
        <p className="text-sm text-destructive">Failed to load rules.</p>
      ) : !data || data.items.length === 0 ? (
        <div className="rounded-md border border-dashed p-8 text-center">
          <p className="text-sm text-muted-foreground">No rules configured.</p>
          <Button
            variant="outline"
            className="mt-4"
            size="sm"
            onClick={() => setAddOpen(true)}
          >
            <Plus className="mr-2 h-4 w-4" />
            Add your first rule
          </Button>
        </div>
      ) : (
        <>
          <div className="rounded-md border divide-y">
            {data.items.map((rule) => (
              <div key={rule.id} className="flex items-center gap-4 px-4 py-3">
                <div className="flex-1 min-w-0 space-y-0.5">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">{RULE_TYPE_LABELS[rule.type]}</Badge>
                    <Badge variant={rule.is_active ? "success" : "secondary"}>
                      {rule.is_active ? "Active" : "Inactive"}
                    </Badge>
                  </div>
                  <p className="text-xs text-muted-foreground truncate">
                    {getRuleConfigSummary(rule)}
                  </p>
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleToggleActive(rule)}
                    className="text-xs"
                  >
                    {rule.is_active ? "Deactivate" : "Activate"}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setEditRule(rule)}
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setDeleteTarget(rule)}
                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-end gap-3 text-sm">
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
          )}
        </>
      )}

      <RuleEditorDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        appId={appId}
      />

      <RuleEditorDialog
        open={Boolean(editRule)}
        onOpenChange={(open) => !open && setEditRule(null)}
        appId={appId}
        editRule={editRule}
      />

      <DeleteConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        ruleType={deleteTarget?.type ?? null}
        onConfirm={handleDeleteConfirm}
        isPending={isDeleting}
      />
    </div>
  );
}
