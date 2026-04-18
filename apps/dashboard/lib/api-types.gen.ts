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
    ErrorResponse: {
      error: {
        code: string;
        message: string;
      };
    };
  };
}

export type webhooks = Record<string, never>;
