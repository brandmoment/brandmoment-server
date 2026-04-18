export type Platform = "ios" | "android" | "web";

export interface PublisherApp {
  id: string;
  org_id: string;
  name: string;
  platform: Platform;
  bundle_id: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePublisherAppRequest {
  name: string;
  platform: Platform;
  bundle_id: string;
}

export interface UpdatePublisherAppRequest {
  name?: string;
  is_active?: boolean;
}

export interface PublisherAppListResponse {
  items: PublisherApp[];
  total: number;
  limit: number;
  offset: number;
}
