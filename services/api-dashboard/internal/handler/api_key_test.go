package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// mockAPIKeyRepoForHandler implements repository.APIKeyRepository for handler tests.
type mockAPIKeyRepoForHandler struct {
	insertFn  func(ctx context.Context, key *model.APIKey) (*model.APIKey, error)
	getByIDFn func(ctx context.Context, orgID, appID, id uuid.UUID) (*model.APIKey, error)
	listFn    func(ctx context.Context, orgID, appID uuid.UUID, activeOnly bool) ([]model.APIKey, error)
	revokeFn  func(ctx context.Context, orgID, appID, id uuid.UUID, revokedAt time.Time) (*model.APIKey, error)
}

func (m *mockAPIKeyRepoForHandler) Insert(ctx context.Context, key *model.APIKey) (*model.APIKey, error) {
	return m.insertFn(ctx, key)
}

func (m *mockAPIKeyRepoForHandler) GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.APIKey, error) {
	return m.getByIDFn(ctx, orgID, appID, id)
}

func (m *mockAPIKeyRepoForHandler) ListByApp(ctx context.Context, orgID, appID uuid.UUID, activeOnly bool) ([]model.APIKey, error) {
	return m.listFn(ctx, orgID, appID, activeOnly)
}

func (m *mockAPIKeyRepoForHandler) Revoke(ctx context.Context, orgID, appID, id uuid.UUID, revokedAt time.Time) (*model.APIKey, error) {
	return m.revokeFn(ctx, orgID, appID, id, revokedAt)
}

// compile-time check: mockAPIKeyRepoForHandler satisfies repository.APIKeyRepository.
var _ repository.APIKeyRepository = (*mockAPIKeyRepoForHandler)(nil)

func newAPIKeyHandler(repo *mockAPIKeyRepoForHandler) *APIKeyHandler {
	svc := service.NewAPIKeyService(repo, noop.NewTracerProvider())
	return NewAPIKeyHandler(svc)
}

func stubAPIKey(orgID, appID uuid.UUID) *model.APIKey {
	return &model.APIKey{
		ID:        uuid.New(),
		OrgID:     orgID,
		AppID:     appID,
		Name:      "Production",
		KeyHash:   "fakehash",
		KeyPrefix: "bm_a1b2",
		IsRevoked: false,
		CreatedAt: time.Now(),
	}
}

// TestAPIKeyHandler_Create covers POST /v1/publisher-apps/{id}/api-keys.
// Critical: the create response MUST include a "key" field (plaintext) and MUST NOT expose key_hash.
func TestAPIKeyHandler_Create(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	tests := []struct {
		name        string
		urlAppID    string
		body        any
		insertFn    func(ctx context.Context, key *model.APIKey) (*model.APIKey, error)
		wantStatus  int
		wantErrCode string
		checkData   func(t *testing.T, data map[string]any)
	}{
		{
			name:     "valid request returns 201 with plaintext key",
			urlAppID: appID.String(),
			body:     map[string]string{"name": "Production"},
			insertFn: func(_ context.Context, key *model.APIKey) (*model.APIKey, error) {
				return key, nil
			},
			wantStatus: http.StatusCreated,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				// Plaintext key MUST be present in create response.
				key, ok := data["key"].(string)
				if !ok || key == "" {
					t.Error("create response must include non-empty 'key' field")
				}
				if !strings.HasPrefix(key, "bm_") {
					t.Errorf("key must start with 'bm_', got %q", key)
				}
				// key_hash MUST NOT appear in the response.
				if _, hasHash := data["key_hash"]; hasHash {
					t.Error("create response must NOT include 'key_hash' field")
				}
				// key_prefix MUST be present.
				prefix, ok := data["key_prefix"].(string)
				if !ok || prefix == "" {
					t.Error("create response must include non-empty 'key_prefix' field")
				}
			},
		},
		{
			name:        "invalid app UUID in URL returns 400",
			urlAppID:    "not-a-uuid",
			body:        map[string]string{"name": "Production"},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:        "invalid JSON body returns 400",
			urlAppID:    appID.String(),
			body:        "not-json{",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:     "empty name returns 400",
			urlAppID: appID.String(),
			body:     map[string]string{"name": ""},
			insertFn: func(_ context.Context, key *model.APIKey) (*model.APIKey, error) {
				return key, nil
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:     "repo insert error returns 500",
			urlAppID: appID.String(),
			body:     map[string]string{"name": "Production"},
			insertFn: func(_ context.Context, _ *model.APIKey) (*model.APIKey, error) {
				return nil, context.DeadlineExceeded
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes := marshalBody(t, tt.body)
			repo := &mockAPIKeyRepoForHandler{}
			if tt.insertFn != nil {
				repo.insertFn = tt.insertFn
			}
			h := newAPIKeyHandler(repo)

			req := httptest.NewRequest(http.MethodPost, "/v1/publisher-apps/"+tt.urlAppID+"/api-keys", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "editor", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlAppID)

			w := httptest.NewRecorder()
			h.Create(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
				return
			}

			if tt.checkData != nil {
				var resp map[string]any
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				data, ok := resp["data"].(map[string]any)
				if !ok {
					t.Fatalf("data field missing: %+v", resp)
				}
				tt.checkData(t, data)
			}
		})
	}
}

// TestAPIKeyHandler_ListByApp covers GET /v1/publisher-apps/{id}/api-keys.
// Critical: list response items MUST NOT include "key" (plaintext) or "key_hash" fields.
func TestAPIKeyHandler_ListByApp(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	tests := []struct {
		name        string
		urlAppID    string
		query       string
		listFn      func(ctx context.Context, orgID, appID uuid.UUID, activeOnly bool) ([]model.APIKey, error)
		wantStatus  int
		wantErrCode string
		checkData   func(t *testing.T, data map[string]any)
	}{
		{
			name:     "returns active keys by default",
			urlAppID: appID.String(),
			query:    "",
			listFn: func(_ context.Context, _, _ uuid.UUID, activeOnly bool) ([]model.APIKey, error) {
				if !activeOnly {
					return nil, context.DeadlineExceeded // should not be called with activeOnly=false
				}
				return []model.APIKey{*stubAPIKey(orgID, appID)}, nil
			},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				items, ok := data["items"].([]any)
				if !ok {
					t.Fatalf("items missing: %+v", data)
				}
				if len(items) != 1 {
					t.Errorf("items count = %d, want 1", len(items))
				}
				// Verify key (plaintext) and key_hash do NOT appear in list items.
				item, ok := items[0].(map[string]any)
				if !ok {
					t.Fatalf("item is not object: %+v", items[0])
				}
				if _, hasKey := item["key"]; hasKey {
					t.Error("list response items must NOT include 'key' (plaintext) field")
				}
				if _, hasHash := item["key_hash"]; hasHash {
					t.Error("list response items must NOT include 'key_hash' field")
				}
				// key_prefix MUST be present.
				if _, hasPrefix := item["key_prefix"]; !hasPrefix {
					t.Error("list response items must include 'key_prefix' field")
				}
			},
		},
		{
			name:     "include_revoked=true passes activeOnly=false to repo",
			urlAppID: appID.String(),
			query:    "?include_revoked=true",
			listFn: func(_ context.Context, _, _ uuid.UUID, activeOnly bool) ([]model.APIKey, error) {
				if activeOnly {
					return nil, context.DeadlineExceeded // should be called with activeOnly=false
				}
				revokedKey := stubAPIKey(orgID, appID)
				revokedKey.IsRevoked = true
				now := time.Now()
				revokedKey.RevokedAt = &now
				return []model.APIKey{*revokedKey}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid app UUID in URL returns 400",
			urlAppID:    "not-a-uuid",
			query:       "",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:     "repo error returns 500",
			urlAppID: appID.String(),
			query:    "",
			listFn: func(_ context.Context, _, _ uuid.UUID, _ bool) ([]model.APIKey, error) {
				return nil, context.DeadlineExceeded
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAPIKeyRepoForHandler{}
			if tt.listFn != nil {
				repo.listFn = tt.listFn
			}
			h := newAPIKeyHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/v1/publisher-apps/"+tt.urlAppID+"/api-keys"+tt.query, nil)
			req = injectAuthContext(req, orgID, "viewer", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlAppID)

			w := httptest.NewRecorder()
			h.List(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
				return
			}

			if tt.checkData != nil {
				var resp map[string]any
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				data, ok := resp["data"].(map[string]any)
				if !ok {
					t.Fatalf("data field missing: %+v", resp)
				}
				tt.checkData(t, data)
			}
		})
	}
}

// TestAPIKeyHandler_Revoke covers DELETE /v1/publisher-apps/{id}/api-keys/{keyId}.
func TestAPIKeyHandler_Revoke(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	keyID := uuid.New()

	now := time.Now()

	tests := []struct {
		name        string
		urlAppID    string
		urlKeyID    string
		repo        *mockAPIKeyRepoForHandler
		wantStatus  int
		wantErrCode string
		checkData   func(t *testing.T, data map[string]any)
	}{
		{
			name:     "valid revoke returns 200 with id and revoked_at",
			urlAppID: appID.String(),
			urlKeyID: keyID.String(),
			repo: &mockAPIKeyRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.APIKey, error) {
					return stubAPIKey(orgID, appID), nil
				},
				revokeFn: func(_ context.Context, _, _, id uuid.UUID, revokedAt time.Time) (*model.APIKey, error) {
					k := stubAPIKey(orgID, appID)
					k.ID = id
					k.IsRevoked = true
					k.RevokedAt = &now
					return k, nil
				},
			},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				if _, ok := data["id"]; !ok {
					t.Error("revoke response must include 'id' field")
				}
				if _, ok := data["revoked_at"]; !ok {
					t.Error("revoke response must include 'revoked_at' field")
				}
				// key_hash must never appear.
				if _, hasHash := data["key_hash"]; hasHash {
					t.Error("revoke response must NOT include 'key_hash' field")
				}
			},
		},
		{
			name:        "invalid app UUID in URL returns 400",
			urlAppID:    "not-a-uuid",
			urlKeyID:    keyID.String(),
			repo:        &mockAPIKeyRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:        "invalid key UUID in URL returns 400",
			urlAppID:    appID.String(),
			urlKeyID:    "not-a-uuid",
			repo:        &mockAPIKeyRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:     "key not found returns 404",
			urlAppID: appID.String(),
			urlKeyID: keyID.String(),
			repo: &mockAPIKeyRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.APIKey, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:     "already revoked key returns 400",
			urlAppID: appID.String(),
			urlKeyID: keyID.String(),
			repo: &mockAPIKeyRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.APIKey, error) {
					k := stubAPIKey(orgID, appID)
					k.IsRevoked = true
					k.RevokedAt = &now
					return k, nil
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:     "repo revoke error returns 500",
			urlAppID: appID.String(),
			urlKeyID: keyID.String(),
			repo: &mockAPIKeyRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.APIKey, error) {
					return stubAPIKey(orgID, appID), nil
				},
				revokeFn: func(_ context.Context, _, _, _ uuid.UUID, _ time.Time) (*model.APIKey, error) {
					return nil, context.DeadlineExceeded
				},
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newAPIKeyHandler(tt.repo)

			req := httptest.NewRequest(http.MethodDelete, "/v1/publisher-apps/"+tt.urlAppID+"/api-keys/"+tt.urlKeyID, nil)
			req = injectAuthContext(req, orgID, "admin", []uuid.UUID{orgID})
			req = withChiAppAndKeyID(req, tt.urlAppID, tt.urlKeyID)

			w := httptest.NewRecorder()
			h.Revoke(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
				return
			}

			if tt.checkData != nil {
				var resp map[string]any
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				data, ok := resp["data"].(map[string]any)
				if !ok {
					t.Fatalf("data field missing: %+v", resp)
				}
				tt.checkData(t, data)
			}
		})
	}
}
