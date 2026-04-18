/**
 * THIS FILE IS AUTO-GENERATED.
 * Run `pnpm codegen` to regenerate from packages/proto/dashboard.yaml
 *
 * Stub version — replace with generated output once the OpenAPI spec exists.
 */

export interface paths {
  "/v1/organizations": {
    get: {
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["Organization"][];
            };
          };
        };
      };
    };
    post: {
      requestBody: {
        content: {
          "application/json": components["schemas"]["CreateOrganizationRequest"];
        };
      };
      responses: {
        201: {
          content: {
            "application/json": {
              data: components["schemas"]["Organization"];
            };
          };
        };
      };
    };
  };
  "/v1/organizations/{id}": {
    get: {
      parameters: {
        path: { id: string };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["Organization"];
            };
          };
        };
      };
    };
  };
  "/v1/me": {
    get: {
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["UserProfile"];
            };
          };
        };
      };
    };
  };
  "/v1/publisher-apps": {
    get: {
      parameters: {
        query?: { limit?: number; offset?: number };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: {
                items: components["schemas"]["PublisherApp"][];
                total: number;
                limit: number;
                offset: number;
              };
            };
          };
        };
      };
    };
    post: {
      requestBody: {
        content: {
          "application/json": components["schemas"]["CreatePublisherAppRequest"];
        };
      };
      responses: {
        201: {
          content: {
            "application/json": {
              data: components["schemas"]["PublisherApp"];
            };
          };
        };
      };
    };
  };
  "/v1/publisher-apps/{id}": {
    get: {
      parameters: {
        path: { id: string };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["PublisherApp"];
            };
          };
        };
      };
    };
    put: {
      parameters: {
        path: { id: string };
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["UpdatePublisherAppRequest"];
        };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["PublisherApp"];
            };
          };
        };
      };
    };
  };
  "/v1/publisher-apps/{id}/api-keys": {
    get: {
      parameters: {
        path: { id: string };
        query?: { include_revoked?: boolean };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: {
                items: components["schemas"]["APIKey"][];
              };
            };
          };
        };
      };
    };
    post: {
      parameters: {
        path: { id: string };
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["CreateAPIKeyRequest"];
        };
      };
      responses: {
        201: {
          content: {
            "application/json": {
              data: components["schemas"]["CreateAPIKeyResponse"];
            };
          };
        };
      };
    };
  };
  "/v1/publisher-apps/{id}/api-keys/{keyId}": {
    delete: {
      parameters: {
        path: { id: string; keyId: string };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: { id: string; revoked_at: string };
            };
          };
        };
      };
    };
  };
  "/v1/publisher-apps/{id}/rules": {
    get: {
      parameters: {
        path: { id: string };
        query?: { limit?: number; offset?: number };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: {
                items: components["schemas"]["PublisherRule"][];
                total: number;
                limit: number;
                offset: number;
              };
            };
          };
        };
      };
    };
    post: {
      parameters: {
        path: { id: string };
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["CreatePublisherRuleRequest"];
        };
      };
      responses: {
        201: {
          content: {
            "application/json": {
              data: components["schemas"]["PublisherRule"];
            };
          };
        };
      };
    };
  };
  "/v1/publisher-apps/{id}/rules/{ruleId}": {
    get: {
      parameters: {
        path: { id: string; ruleId: string };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["PublisherRule"];
            };
          };
        };
      };
    };
    put: {
      parameters: {
        path: { id: string; ruleId: string };
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["UpdatePublisherRuleRequest"];
        };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["PublisherRule"];
            };
          };
        };
      };
    };
    delete: {
      parameters: {
        path: { id: string; ruleId: string };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: { id: string };
            };
          };
        };
      };
    };
  };
  "/v1/campaigns": {
    get: {
      parameters: {
        query?: {
          limit?: number;
          offset?: number;
          status?: "draft" | "active" | "paused" | "completed";
        };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: {
                items: components["schemas"]["Campaign"][];
                total: number;
                limit: number;
                offset: number;
              };
            };
          };
        };
      };
    };
    post: {
      requestBody: {
        content: {
          "application/json": components["schemas"]["CreateCampaignRequest"];
        };
      };
      responses: {
        201: {
          content: {
            "application/json": {
              data: components["schemas"]["Campaign"];
            };
          };
        };
      };
    };
  };
  "/v1/campaigns/{id}": {
    get: {
      parameters: {
        path: { id: string };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["Campaign"];
            };
          };
        };
      };
    };
    put: {
      parameters: {
        path: { id: string };
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["UpdateCampaignRequest"];
        };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["Campaign"];
            };
          };
        };
      };
    };
  };
  "/v1/campaigns/{id}/status": {
    patch: {
      parameters: {
        path: { id: string };
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["UpdateCampaignStatusRequest"];
        };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: components["schemas"]["Campaign"];
            };
          };
        };
      };
    };
  };
  "/v1/campaigns/{id}/creatives": {
    get: {
      parameters: {
        path: { id: string };
      };
      responses: {
        200: {
          content: {
            "application/json": {
              data: {
                items: components["schemas"]["Creative"][];
                total: number;
              };
            };
          };
        };
      };
    };
    post: {
      parameters: {
        path: { id: string };
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["CreateCreativeRequest"];
        };
      };
      responses: {
        201: {
          content: {
            "application/json": {
              data: components["schemas"]["Creative"];
            };
          };
        };
      };
    };
  };
  "/v1/orgs/{id}/invites": {
    post: {
      parameters: {
        path: { id: string };
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["CreateInviteRequest"];
        };
      };
      responses: {
        201: {
          content: {
            "application/json": {
              data: components["schemas"]["OrgInvite"];
            };
          };
        };
      };
    };
  };
}

export interface components {
  schemas: {
    Organization: {
      id: string;
      type: "admin" | "publisher" | "brand";
      name: string;
      slug: string;
      created_at: string;
      updated_at: string;
    };
    CreateOrganizationRequest: {
      type: "publisher" | "brand";
      name: string;
      slug: string;
    };
    UserProfile: {
      id: string;
      email: string;
      name: string;
      created_at: string;
      orgs: Array<{
        org_id: string;
        role: "owner" | "admin" | "editor" | "viewer";
      }>;
    };
    CreateInviteRequest: {
      email: string;
      role: "admin" | "editor" | "viewer";
    };
    OrgInvite: {
      invite_id: string;
      token: string;
      email: string;
      role: string;
      org_id: string;
      expires_at: string;
    };
    PublisherApp: {
      id: string;
      org_id: string;
      name: string;
      platform: "ios" | "android" | "web";
      bundle_id: string;
      is_active: boolean;
      created_at: string;
      updated_at: string;
    };
    CreatePublisherAppRequest: {
      name: string;
      platform: "ios" | "android" | "web";
      bundle_id: string;
    };
    UpdatePublisherAppRequest: {
      name?: string;
      is_active?: boolean;
    };
    APIKey: {
      id: string;
      org_id: string;
      app_id: string;
      name: string;
      key_prefix: string;
      is_revoked: boolean;
      created_at: string;
      revoked_at: string | null;
    };
    CreateAPIKeyRequest: {
      name: string;
    };
    CreateAPIKeyResponse: {
      id: string;
      org_id: string;
      app_id: string;
      name: string;
      key: string;
      key_prefix: string;
      is_revoked: boolean;
      created_at: string;
    };
    PublisherRule: {
      id: string;
      org_id: string;
      app_id: string;
      type: "blocklist" | "allowlist" | "frequency_cap" | "geo_filter" | "platform_filter";
      config: Record<string, unknown>;
      is_active: boolean;
      created_at: string;
      updated_at: string;
    };
    CreatePublisherRuleRequest: {
      type: "blocklist" | "allowlist" | "frequency_cap" | "geo_filter" | "platform_filter";
      config: Record<string, unknown>;
    };
    UpdatePublisherRuleRequest: {
      config?: Record<string, unknown>;
      is_active?: boolean;
    };
    Campaign: {
      id: string;
      org_id: string;
      name: string;
      status: "draft" | "active" | "paused" | "completed";
      targeting: {
        geo: string[];
        platforms: string[];
        age_range?: { min: number; max: number };
        interests: string[];
      };
      budget_cents: number | null;
      currency: string;
      start_date: string | null;
      end_date: string | null;
      created_at: string;
      updated_at: string;
    };
    CreateCampaignRequest: {
      name: string;
      targeting?: {
        geo?: string[];
        platforms?: string[];
        age_range?: { min: number; max: number };
        interests?: string[];
      };
      budget_cents?: number | null;
      currency?: string;
      start_date?: string | null;
      end_date?: string | null;
    };
    UpdateCampaignRequest: {
      name?: string;
      targeting?: {
        geo?: string[];
        platforms?: string[];
        age_range?: { min: number; max: number };
        interests?: string[];
      };
      budget_cents?: number | null;
      currency?: string;
      start_date?: string | null;
      end_date?: string | null;
    };
    UpdateCampaignStatusRequest: {
      status: "draft" | "active" | "paused" | "completed";
    };
    Creative: {
      id: string;
      org_id: string;
      campaign_id: string;
      name: string;
      type: "html5" | "image" | "video";
      file_url: string;
      file_size_bytes: number | null;
      preview_url: string | null;
      is_active: boolean;
      created_at: string;
      updated_at: string;
    };
    CreateCreativeRequest: {
      name: string;
      type: "html5" | "image" | "video";
      file_url: string;
      file_size_bytes?: number | null;
      preview_url?: string | null;
    };
    ErrorResponse: {
      error: {
        code: string;
        message: string;
      };
    };
  };
}

export type webhooks = Record<string, never>;
