package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// mockUserRepoForHandler satisfies repository.UserRepository for handler-level tests
// without a real database connection.
type mockUserRepoForHandler struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.User, error)
	upsertFn  func(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error)
}

func (m *mockUserRepoForHandler) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockUserRepoForHandler) Upsert(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error) {
	return m.upsertFn(ctx, id, email, name, createdAt)
}

func newUserHandler(repo *mockUserRepoForHandler) *UserHandler {
	svc := service.NewUserService(repo, noop.NewTracerProvider())
	return NewUserHandler(svc)
}

// TestUserHandler_GetMe covers the GET /v1/me endpoint.
func TestUserHandler_GetMe(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	fixedTime := time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)

	existingUser := &model.User{
		ID:        userID,
		Email:     "alice@example.com",
		Name:      "Alice",
		CreatedAt: fixedTime,
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		orgID       uuid.UUID
		role        string
		orgIDs      []uuid.UUID
		getByIDFn   func(ctx context.Context, id uuid.UUID) (*model.User, error)
		wantStatus  int
		wantErrCode string
		checkData   func(t *testing.T, data map[string]any)
	}{
		{
			name:   "returns user profile with org claims",
			userID: userID,
			orgID:  orgID,
			role:   "owner",
			orgIDs: []uuid.UUID{orgID},
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
				return existingUser, nil
			},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				if data["email"] != "alice@example.com" {
					t.Errorf("email = %v, want alice@example.com", data["email"])
				}
				if data["name"] != "Alice" {
					t.Errorf("name = %v, want Alice", data["name"])
				}
				orgs, ok := data["orgs"].([]any)
				if !ok || len(orgs) != 1 {
					t.Errorf("orgs = %v, want slice of length 1", data["orgs"])
				}
			},
		},
		{
			name:   "returns empty orgs when no org claims in context",
			userID: userID,
			orgID:  orgID,
			role:   "viewer",
			orgIDs: []uuid.UUID{},
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
				return existingUser, nil
			},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				orgs, ok := data["orgs"].([]any)
				if !ok {
					t.Errorf("orgs field missing or wrong type: %v", data["orgs"])
					return
				}
				if len(orgs) != 0 {
					t.Errorf("orgs len = %d, want 0", len(orgs))
				}
			},
		},
		{
			name:   "user not found returns 404",
			userID: uuid.New(),
			orgID:  orgID,
			role:   "editor",
			orgIDs: []uuid.UUID{orgID},
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
				return nil, model.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:   "service db error returns 500",
			userID: userID,
			orgID:  orgID,
			role:   "admin",
			orgIDs: []uuid.UUID{orgID},
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
				return nil, context.DeadlineExceeded
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
		{
			name:   "multiple org claims all returned",
			userID: userID,
			orgID:  orgID,
			role:   "editor",
			orgIDs: []uuid.UUID{orgID, uuid.New(), uuid.New()},
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
				return existingUser, nil
			},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				orgs, ok := data["orgs"].([]any)
				if !ok {
					t.Fatalf("orgs field missing: %v", data["orgs"])
				}
				if len(orgs) != 3 {
					t.Errorf("orgs len = %d, want 3", len(orgs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepoForHandler{getByIDFn: tt.getByIDFn}
			h := newUserHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
			ctx := middleware.InjectTestContext(req.Context(), tt.userID, tt.orgID, tt.role, tt.orgIDs)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			h.GetMe(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("GetMe() status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]any
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode body: %v", err)
			}

			if tt.wantErrCode != "" {
				errField, ok := resp["error"].(map[string]any)
				if !ok {
					t.Fatalf("GetMe() error field missing: %+v", resp)
				}
				gotCode, _ := errField["code"].(string)
				if gotCode != tt.wantErrCode {
					t.Errorf("GetMe() error.code = %q, want %q", gotCode, tt.wantErrCode)
				}
				return
			}

			data, ok := resp["data"].(map[string]any)
			if !ok {
				t.Fatalf("GetMe() data field missing: %+v", resp)
			}
			if tt.checkData != nil {
				tt.checkData(t, data)
			}
		})
	}
}
