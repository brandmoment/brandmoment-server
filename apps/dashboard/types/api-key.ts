export interface APIKey {
  id: string;
  org_id: string;
  app_id: string;
  name: string;
  key_prefix: string;
  is_revoked: boolean;
  created_at: string;
  revoked_at: string | null;
}

export interface CreateAPIKeyRequest {
  name: string;
}

/**
 * Response from POST /v1/publisher-apps/:id/api-keys.
 * The `key` field contains the full plaintext — returned ONCE only.
 */
export interface CreateAPIKeyResponse {
  id: string;
  org_id: string;
  app_id: string;
  name: string;
  key: string;
  key_prefix: string;
  is_revoked: boolean;
  created_at: string;
}

export interface APIKeyListResponse {
  items: APIKey[];
}
