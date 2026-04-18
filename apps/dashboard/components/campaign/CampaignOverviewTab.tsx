"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Pencil, X, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CampaignStatusBadge } from "@/components/campaign/CampaignStatusBadge";
import { TargetingEditor } from "@/components/campaign/TargetingEditor";
import { useUpdateCampaignStatus } from "@/hooks/useUpdateCampaignStatus";
import { useUpdateCampaign } from "@/hooks/useUpdateCampaign";
import { VALID_TRANSITIONS } from "@/types/campaign";
import type { Campaign, CampaignStatus, CampaignTargeting } from "@/types/campaign";

const STATUS_LABELS: Record<CampaignStatus, string> = {
  draft: "Draft",
  active: "Activate",
  paused: "Pause",
  completed: "Complete",
};

const editSchema = z
  .object({
    name: z.string().min(1).max(200),
    budget_cents: z.string().optional(),
    currency: z.string().min(1).max(3),
    start_date: z.string().optional(),
    end_date: z.string().optional(),
  })
  .refine(
    (d) => {
      if (d.start_date && d.end_date) {
        return new Date(d.end_date) > new Date(d.start_date);
      }
      return true;
    },
    { message: "End date must be after start date", path: ["end_date"] }
  );

type EditFormValues = z.infer<typeof editSchema>;

interface CampaignOverviewTabProps {
  campaign: Campaign;
}

function formatDate(iso: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function formatBudget(cents: number | null, currency: string) {
  if (cents === null) return "Not set";
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
    minimumFractionDigits: 0,
  }).format(cents / 100);
}

export function CampaignOverviewTab({ campaign }: CampaignOverviewTabProps) {
  const [editing, setEditing] = useState(false);
  const [targeting, setTargeting] = useState<Partial<CampaignTargeting>>(
    campaign.targeting
  );

  const { mutateAsync: updateStatus, isPending: statusPending } =
    useUpdateCampaignStatus();
  const { mutateAsync: updateCampaign, isPending: updatePending } =
    useUpdateCampaign();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<EditFormValues>({
    resolver: zodResolver(editSchema),
    defaultValues: {
      name: campaign.name,
      budget_cents:
        campaign.budget_cents !== null
          ? (campaign.budget_cents / 100).toString()
          : "",
      currency: campaign.currency,
      start_date: campaign.start_date ?? "",
      end_date: campaign.end_date ?? "",
    },
  });

  const validNextStatuses = VALID_TRANSITIONS[campaign.status];

  async function handleStatusTransition(target: CampaignStatus) {
    try {
      await updateStatus({ id: campaign.id, status: target });
      toast.success(`Campaign status updated to ${target}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update status");
    }
  }

  async function onSave(values: EditFormValues) {
    const budgetCents =
      values.budget_cents
        ? Math.round(parseFloat(values.budget_cents) * 100)
        : null;
    try {
      await updateCampaign({
        id: campaign.id,
        body: {
          name: values.name,
          targeting,
          budget_cents: budgetCents,
          currency: values.currency,
          start_date: values.start_date || null,
          end_date: values.end_date || null,
        },
      });
      toast.success("Campaign updated");
      setEditing(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update campaign");
    }
  }

  function handleCancel() {
    reset();
    setTargeting(campaign.targeting);
    setEditing(false);
  }

  return (
    <div className="space-y-6">
      {/* Status Transitions */}
      {validNextStatuses.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground uppercase tracking-wide">
              Status Transitions
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <CampaignStatusBadge status={campaign.status} />
              <span className="text-muted-foreground text-sm">→</span>
              {validNextStatuses.map((next) => (
                <Button
                  key={next}
                  variant="outline"
                  size="sm"
                  disabled={statusPending}
                  onClick={() => handleStatusTransition(next)}
                >
                  {STATUS_LABELS[next]}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Campaign Details */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium text-muted-foreground uppercase tracking-wide">
              Campaign Details
            </CardTitle>
            {!editing ? (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setEditing(true)}
              >
                <Pencil className="h-4 w-4 mr-1" />
                Edit
              </Button>
            ) : (
              <div className="flex gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleCancel}
                  disabled={updatePending}
                >
                  <X className="h-4 w-4 mr-1" />
                  Cancel
                </Button>
                <Button
                  size="sm"
                  onClick={handleSubmit(onSave)}
                  disabled={updatePending}
                >
                  <Check className="h-4 w-4 mr-1" />
                  {updatePending ? "Saving..." : "Save"}
                </Button>
              </div>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {editing ? (
            <>
              <div className="space-y-2">
                <Label>Name</Label>
                <Input {...register("name")} />
                {errors.name && (
                  <p className="text-xs text-destructive">{errors.name.message}</p>
                )}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Budget</Label>
                  <Input
                    type="number"
                    step="0.01"
                    min="0"
                    placeholder="500.00"
                    {...register("budget_cents")}
                  />
                  {errors.budget_cents && (
                    <p className="text-xs text-destructive">
                      {errors.budget_cents.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label>Currency</Label>
                  <Input maxLength={3} {...register("currency")} />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Start date</Label>
                  <Input type="date" {...register("start_date")} />
                </div>
                <div className="space-y-2">
                  <Label>End date</Label>
                  <Input type="date" {...register("end_date")} />
                  {errors.end_date && (
                    <p className="text-xs text-destructive">
                      {errors.end_date.message}
                    </p>
                  )}
                </div>
              </div>
            </>
          ) : (
            <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
              <div>
                <dt className="text-muted-foreground">Name</dt>
                <dd className="font-medium mt-0.5">{campaign.name}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Status</dt>
                <dd className="mt-0.5">
                  <CampaignStatusBadge status={campaign.status} />
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Budget</dt>
                <dd className="font-medium mt-0.5">
                  {formatBudget(campaign.budget_cents, campaign.currency)}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Currency</dt>
                <dd className="font-medium mt-0.5">{campaign.currency}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Start date</dt>
                <dd className="font-medium mt-0.5">{formatDate(campaign.start_date)}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">End date</dt>
                <dd className="font-medium mt-0.5">{formatDate(campaign.end_date)}</dd>
              </div>
            </dl>
          )}
        </CardContent>
      </Card>

      {/* Targeting */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium text-muted-foreground uppercase tracking-wide">
            Targeting
          </CardTitle>
        </CardHeader>
        <CardContent>
          {editing ? (
            <TargetingEditor
              value={targeting}
              onChange={setTargeting}
              disabled={updatePending}
            />
          ) : (
            <div className="space-y-3 text-sm">
              <div>
                <p className="text-muted-foreground mb-1">Geo</p>
                <div className="flex flex-wrap gap-1">
                  {campaign.targeting.geo.length > 0 ? (
                    campaign.targeting.geo.map((g) => (
                      <Badge key={g} variant="outline">
                        {g}
                      </Badge>
                    ))
                  ) : (
                    <span className="text-muted-foreground">All regions</span>
                  )}
                </div>
              </div>
              <div>
                <p className="text-muted-foreground mb-1">Platforms</p>
                <div className="flex flex-wrap gap-1">
                  {campaign.targeting.platforms.length > 0 ? (
                    campaign.targeting.platforms.map((p) => (
                      <Badge key={p} variant="outline">
                        {p}
                      </Badge>
                    ))
                  ) : (
                    <span className="text-muted-foreground">All platforms</span>
                  )}
                </div>
              </div>
              {campaign.targeting.age_range && (
                <div>
                  <p className="text-muted-foreground mb-1">Age range</p>
                  <span>
                    {campaign.targeting.age_range.min}–{campaign.targeting.age_range.max}
                  </span>
                </div>
              )}
              <div>
                <p className="text-muted-foreground mb-1">Interests</p>
                <div className="flex flex-wrap gap-1">
                  {campaign.targeting.interests.length > 0 ? (
                    campaign.targeting.interests.map((i) => (
                      <Badge key={i} variant="outline">
                        {i}
                      </Badge>
                    ))
                  ) : (
                    <span className="text-muted-foreground">All interests</span>
                  )}
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
