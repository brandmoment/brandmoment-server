export type CampaignStatus = "draft" | "active" | "paused" | "completed";

export type CreativeType = "html5" | "image" | "video";

export interface CampaignTargeting {
  geo: string[];
  platforms: string[];
  age_range?: { min: number; max: number };
  interests: string[];
}

export interface Campaign {
  id: string;
  org_id: string;
  name: string;
  status: CampaignStatus;
  targeting: CampaignTargeting;
  budget_cents: number | null;
  currency: string;
  start_date: string | null;
  end_date: string | null;
  created_at: string;
  updated_at: string;
}

export interface Creative {
  id: string;
  org_id: string;
  campaign_id: string;
  name: string;
  type: CreativeType;
  file_url: string;
  file_size_bytes: number | null;
  preview_url: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateCampaignRequest {
  name: string;
  targeting?: Partial<CampaignTargeting>;
  budget_cents?: number | null;
  currency?: string;
  start_date?: string | null;
  end_date?: string | null;
}

export interface UpdateCampaignRequest {
  name?: string;
  targeting?: Partial<CampaignTargeting>;
  budget_cents?: number | null;
  currency?: string;
  start_date?: string | null;
  end_date?: string | null;
}

export interface UpdateCampaignStatusRequest {
  status: CampaignStatus;
}

export interface CreateCreativeRequest {
  name: string;
  type: CreativeType;
  file_url: string;
  file_size_bytes?: number | null;
  preview_url?: string | null;
}

export interface CampaignListResponse {
  items: Campaign[];
  total: number;
  limit: number;
  offset: number;
}

export interface CreativeListResponse {
  items: Creative[];
  total: number;
}

export const VALID_TRANSITIONS: Record<CampaignStatus, CampaignStatus[]> = {
  draft: ["active"],
  active: ["paused", "completed"],
  paused: ["active", "completed"],
  completed: [],
};
