export type RuleType =
  | "blocklist"
  | "allowlist"
  | "frequency_cap"
  | "geo_filter"
  | "platform_filter";

export interface BlocklistAllowlistConfig {
  domains: string[];
  bundle_ids: string[];
}

export interface FrequencyCapConfig {
  max_impressions: number;
  window_seconds: number;
}

export interface GeoFilterConfig {
  mode: "include" | "exclude";
  country_codes: string[];
}

export interface PlatformFilterConfig {
  mode: "include" | "exclude";
  platforms: ("ios" | "android" | "web")[];
}

export type RuleConfig =
  | BlocklistAllowlistConfig
  | FrequencyCapConfig
  | GeoFilterConfig
  | PlatformFilterConfig;

export interface PublisherRule {
  id: string;
  org_id: string;
  app_id: string;
  type: RuleType;
  config: Record<string, unknown>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePublisherRuleRequest {
  type: RuleType;
  config: Record<string, unknown>;
}

export interface UpdatePublisherRuleRequest {
  config?: Record<string, unknown>;
  is_active?: boolean;
}

export interface PublisherRuleListResponse {
  items: PublisherRule[];
  total: number;
  limit: number;
  offset: number;
}
