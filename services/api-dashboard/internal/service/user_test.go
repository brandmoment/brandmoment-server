package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

type mockUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.User, error)
	upsertFn  func(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error)
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockUserRepo) Upsert(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error) {
	return m.upsertFn(ctx, id, email, name, createdAt)
}

func TestUserService_GetMe(t *testing.T) {
	userID := uuid.New()
	existingUser := &model.User{
		ID:        userID,
		Email:     "alice@example.com",
		Name:      "Alice",
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name    string
		userID  uuid.UUID
		repoFn  func(ctx context.Context, id uuid.UUID) (*model.User, error)
		wantErr bool
	}{
		{
			name:   "user exists",
			userID: userID,
			repoFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
				return existingUser, nil
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: uuid.New(),
			repoFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
		{
			name:   "db error",
			userID: userID,
			repoFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
				return nil, errors.New("connection error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepo{getByIDFn: tt.repoFn}
			svc := NewUserService(repo, noop.NewTracerProvider())

			got, err := svc.GetMe(context.Background(), tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("GetMe() returned nil user")
			}
			if !tt.wantErr && got.ID != tt.userID {
				t.Errorf("GetMe() id = %v, want %v", got.ID, tt.userID)
			}
		})
	}
}

func TestUserService_UpsertUser(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		email   string
		uname   string
		repoFn  func(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error)
		wantErr bool
	}{
		{
			name:  "new user",
			id:    userID,
			email: "bob@example.com",
			uname: "Bob",
			repoFn: func(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error) {
				return &model.User{ID: id, Email: email, Name: name, CreatedAt: createdAt}, nil
			},
			wantErr: false,
		},
		{
			name:  "existing user same email",
			id:    userID,
			email: "bob@example.com",
			uname: "Robert",
			repoFn: func(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error) {
				return &model.User{ID: id, Email: email, Name: name, CreatedAt: createdAt}, nil
			},
			wantErr: false,
		},
		{
			name:    "empty email",
			id:      userID,
			email:   "",
			uname:   "Bob",
			repoFn:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepo{upsertFn: tt.repoFn}
			svc := NewUserService(repo, noop.NewTracerProvider())

			got, err := svc.UpsertUser(context.Background(), tt.id, tt.email, tt.uname)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpsertUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && errors.Is(err, model.ErrInvalidInput) && tt.email == "" {
				// expected path
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("UpsertUser() returned nil user")
					return
				}
				if got.Email != tt.email {
					t.Errorf("UpsertUser() email = %v, want %v", got.Email, tt.email)
				}
				if got.Name != tt.uname {
					t.Errorf("UpsertUser() name = %v, want %v", got.Name, tt.uname)
				}
			}
		})
	}
}
