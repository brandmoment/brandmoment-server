import { Badge } from "@/components/ui/badge";
import type { CampaignStatus } from "@/types/campaign";

const STATUS_LABELS: Record<CampaignStatus, string> = {
  draft: "Draft",
  active: "Active",
  paused: "Paused",
  completed: "Completed",
};

const STATUS_VARIANTS: Record<
  CampaignStatus,
  "outline" | "success" | "warning" | "secondary"
> = {
  draft: "outline",
  active: "success",
  paused: "warning",
  completed: "secondary",
};

interface CampaignStatusBadgeProps {
  status: CampaignStatus;
}

export function CampaignStatusBadge({ status }: CampaignStatusBadgeProps) {
  return (
    <Badge variant={STATUS_VARIANTS[status]}>{STATUS_LABELS[status]}</Badge>
  );
}
