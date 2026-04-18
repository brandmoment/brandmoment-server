package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// mockOrgInviteRepoForHandler satisfies repository.OrgInviteRepository for handler-level tests.
type mockOrgInviteRepoForHandler struct {
	insertFn     func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error)
	getByTokenFn func(ctx context.Context, token string) (*model.OrgInvite, error)
}

func (m *mockOrgInviteRepoForHandler) Insert(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
	return m.insertFn(ctx, invite)
}

func (m *mockOrgInviteRepoForHandler) GetByToken(ctx context.Context, token string) (*model.OrgInvite, error) {
	return m.getByTokenFn(ctx, token)
}

func newOrgInviteHandlerWithRepo(repo *mockOrgInviteRepoForHandler) *OrgInviteHandler {
	svc := service.NewOrgInviteService(repo, noop.NewTracerProvider())
	return NewOrgInviteHandler(svc)
}

// TestOrgInviteHandler_Create covers POST /v1/orgs/{id}/invites.
func TestOrgInviteHandler_Create(t *testing.T) {
	orgID := uuid.New()
	callerID := uuid.New()

	tests := []struct {
		name        string
		urlOrgID    string
		body        any
		insertFn    func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error)
		wantStatus  int
		wantErrCode string
		checkData   func(t *testing.T, data map[string]any)
	}{
		{
			name:     "valid invite with editor role returns 201",
			urlOrgID: orgID.String(),
			body:     map[string]string{"email": "alice@example.com", "role": "editor"},
			insertFn: func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
				return invite, nil
			},
			wantStatus: http.StatusCreated,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				if data["token"] == "" {
					t.Error("token should be non-empty")
				}
				if data["email"] != "alice@example.com" {
					t.Errorf("email = %v, want alice@example.com", data["email"])
				}
				if data["role"] != "editor" {
					t.Errorf("role = %v, want editor", data["role"])
				}
			},
		},
		{
			name:     "valid invite with admin role returns 201",
			urlOrgID: orgID.String(),
			body:     map[string]string{"email": "bob@example.com", "role": "admin"},
			insertFn: func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
				return invite, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:     "valid invite with viewer role returns 201",
			urlOrgID: orgID.String(),
			body:     map[string]string{"email": "viewer@example.com", "role": "viewer"},
			insertFn: func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
				return invite, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:        "invalid org UUID in path returns 400",
			urlOrgID:    "not-a-uuid",
			body:        map[string]string{"email": "a@b.com", "role": "editor"},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:        "invalid JSON body returns 400",
			urlOrgID:    orgID.String(),
			body:        "not-json{",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:        "empty email returns 400",
			urlOrgID:    orgID.String(),
			body:        map[string]string{"email": "", "role": "editor"},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "owner role returns 400",
			urlOrgID:    orgID.String(),
			body:        map[string]string{"email": "owner@example.com", "role": "owner"},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "invalid role returns 400",
			urlOrgID:    orgID.String(),
			body:        map[string]string{"email": "super@example.com", "role": "superadmin"},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:     "db error returns 500",
			urlOrgID: orgID.String(),
			body:     map[string]string{"email": "fail@example.com", "role": "viewer"},
			insertFn: func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
				return nil, context.DeadlineExceeded
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			var err error
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, err = json.Marshal(v)
				if err != nil {
					t.Fatalf("marshal body: %v", err)
				}
			}

			repo := &mockOrgInviteRepoForHandler{}
			if tt.insertFn != nil {
				repo.insertFn = tt.insertFn
			}
			h := newOrgInviteHandlerWithRepo(repo)

			req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+tt.urlOrgID+"/invites", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := middleware.InjectTestContext(req.Context(), callerID, orgID, "owner", []uuid.UUID{orgID})
			req = req.WithContext(ctx)
			req = withChiID(req, tt.urlOrgID)

			w := httptest.NewRecorder()
			h.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Create() status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]any
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode body: %v", err)
			}

			if tt.wantErrCode != "" {
				errField, ok := resp["error"].(map[string]any)
				if !ok {
					t.Fatalf("Create() error field missing: %+v", resp)
				}
				gotCode, _ := errField["code"].(string)
				if gotCode != tt.wantErrCode {
					t.Errorf("Create() error.code = %q, want %q", gotCode, tt.wantErrCode)
				}
				return
			}

			data, ok := resp["data"].(map[string]any)
			if !ok {
				t.Fatalf("Create() data field missing: %+v", resp)
			}
			if tt.checkData != nil {
				tt.checkData(t, data)
			}
		})
	}
}
