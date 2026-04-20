# Baseline — gpt-4o-mini (no tuning)

Source eval: `data/eval.jsonl`

Run: 10 example(s)

---

## Example 1

### Input (user)

```go
func (r *apiKeyRepo) Insert(ctx context.Context, key *model.APIKey) (*model.APIKey, error) {
	row, err := r.q.InsertAPIKey(ctx, db.InsertAPIKeyParams{
		ID:        uuidToPgtype(key.ID),
		OrgID:     uuidToPgtype(key.OrgID),
		AppID:     uuidToPgtype(key.AppID),
		Name:      key.Name,
		KeyHash:   key.KeyHash,
		KeyPrefix: key.KeyPrefix,
		CreatedAt: pgtype.Timestamptz{Time: key.CreatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}
	return toAPIKey(row), nil
}

// Scenario: DBError
```

### Reference (expected)

```go
func TestAPIKeyRepo_Insert_DBError(t *testing.T) {
	dbErr := errSentinel("insert failed")
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: dbErr}
		},
	}
	repo := &apiKeyRepo{q: db.New(mock)}
	_, err := repo.Insert(context.Background(), &model.APIKey{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		AppID:     uuid.New(),
		Name:      "key",
		KeyHash:   "hash",
		KeyPrefix: "bm_",
		CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

### Baseline output

```go
```go
func TestApiKeyRepo_Insert_DBError(t *testing.T) {
	tests := []struct {
		name     string
		key      *model.APIKey
		mockFunc func(m *mock_dbtx.MockDBTX)
		wantErr  bool
	}{
		{
			name: "DB error on insert",
			key: &model.APIKey{
				ID:        "test-id",
				OrgID:     "test-org-id",
				AppID:     "test-app-id",
				Name:      "test-name",
				KeyHash:   "test-key-hash",
				KeyPrefix: "test-key-prefix",
				CreatedAt: time.Now(),
			},
			mockFunc: func(m *mock_dbtx.MockDBTX) {
				m.On("InsertAPIKey", mock.Anything, mock.Anything).Return(errSentinel)
			},
			wantErr: true,
		},
		{
			name: "successful insert",
			key: &model.APIKey{
				ID:        "test-id",
				OrgID:     "test-org-id",
				AppID:     "test-app-id",
				Name:      "test-name",
				KeyHash:   "test-key-hash",
				KeyPrefix: "test-key-prefix",
				CreatedAt: time.Now(),
			},
			mockFunc: func(m *mock_dbtx.MockDBTX) {
				m.On("InsertAPIKey", mock.Anything, mock.Anything).Return(mockRow{...}, nil) // replace with actual mock row
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := mockDBTX()
			tt.mockFunc(mockDB)

			repo := &apiKeyRepo{q: mockDB}
			got, err := repo.Insert(context.Background(), tt.key)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Fatalf("Insert() got = %v, want non-nil", got)
			}
		})
	}
}
```
```

### Evaluation

- Compiles: [✗] — `mock_dbtx.MockDBTX` не существует, UUID как строки
- Uses mock_dbtx helpers correctly: [✗] — testify/mock вместо `&mockDBTX{queryRowFn:...}`
- Naming matches TestType_Method[_Scenario]: [✓] — `TestApiKeyRepo_Insert_DBError` (camelCase вместо PascalCase но допустимо)
- org_id / scope params covered: [✗] — org_id строкой, не uuid.New()
- No extra blocks: [✗] — лишний success-кейс

**Score: Блок 1: 1/5 | Блок 2: 1/3 | Блок 3: 0/4 | Итого: 2/12**

---

## Example 2

### Input (user)

```go
func (r *campaignRepo) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status model.CampaignStatus, updatedAt time.Time) (*model.Campaign, error) {
	row, err := r.q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		Status:    string(status),
		UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
		OrgID:     uuidToPgtype(orgID),
		ID:        uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update campaign status: %w", err)
	}
	return toCampaign(row)
}

// Scenario: NotFound
```

### Reference (expected)

```go
func TestCampaignRepo_UpdateStatus_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &campaignRepo{q: db.New(mock)}
	_, err := repo.UpdateStatus(context.Background(), uuid.New(), uuid.New(), model.StatusActive, time.Now())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}
```

### Baseline output

```go
```go
func TestCampaignRepo_UpdateStatus_NotFound(t *testing.T) {
	tests := []struct {
		name     string
		orgID    uuid.UUID
		id       uuid.UUID
		status   model.CampaignStatus
		updatedAt time.Time
		mock     func(m *mock_dbtx)
	}{
		{
			name:     "not found",
			orgID:    uuid.New(),
			id:       uuid.New(),
			status:   model.CampaignStatusActive,
			updatedAt: time.Now(),
			mock: func(m *mock_dbtx) {
				m.ExpectUpdateCampaignStatus(gomock.Any(), gomock.Any()).Return(errRow)
			},
		},
		{
			name:     "database error",
			orgID:    uuid.New(),
			id:       uuid.New(),
			status:   model.CampaignStatusActive,
			updatedAt: time.Now(),
			mock: func(m *mock_dbtx) {
				m.ExpectUpdateCampaignStatus(gomock.Any(), gomock.Any()).Return(errSentinel)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMockDBTX(t)
			tt.mock(m)

			repo := &campaignRepo{q: m}
			campaign, err := repo.UpdateStatus(context.Background(), tt.orgID, tt.id, tt.status, tt.updatedAt)

			if tt.name == "not found" {
				if err != model.ErrNotFound {
					t.Fatalf("expected error %v, got %v", model.ErrNotFound, err)
				}
				if campaign != nil {
					t.Fatalf("expected campaign to be nil, got %v", campaign)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if campaign != nil {
					t.Fatalf("expected campaign to be nil, got %v", campaign)
				}
			}
		})
	}
}
```
```

### Evaluation

- Compiles: [✗] — `newMockDBTX(t)` не существует, gomock стиль
- Uses mock_dbtx helpers correctly: [✗] — gomock + `.ExpectUpdateCampaignStatus()` вместо `&mockDBTX{}`
- Naming matches TestType_Method[_Scenario]: [✓] — `TestCampaignRepo_UpdateStatus_NotFound`
- org_id / scope params covered: [✓] — передаёт orgID через uuid.New()
- No extra blocks: [✗] — лишний "database error" кейс + условная логика в ассертах

**Score: Блок 1: 2/5 | Блок 2: 1/3 | Блок 3: 1/4 | Итого: 4/12**

---

## Example 3

### Input (user)

```go
func (r *creativeRepo) Insert(ctx context.Context, c *model.Creative) (*model.Creative, error) {
	row, err := r.q.InsertCreative(ctx, db.InsertCreativeParams{
		ID:            uuidToPgtype(c.ID),
		OrgID:         uuidToPgtype(c.OrgID),
		CampaignID:    uuidToPgtype(c.CampaignID),
		Name:          c.Name,
		Type:          string(c.Type),
		FileUrl:       c.FileURL,
		FileSizeBytes: int64ToPgtypeInt8(c.FileSizeBytes),
		PreviewUrl:    stringToPgtypeText(c.PreviewURL),
		IsActive:      c.IsActive,
		CreatedAt:     pgtype.Timestamptz{Time: c.CreatedAt, Valid: true},
		UpdatedAt:     pgtype.Timestamptz{Time: c.UpdatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert creative: %w", err)
	}
	return toCreative(row), nil
}

// Scenario: DBError
```

### Reference (expected)

```go
func TestCreativeRepo_Insert_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("insert failed")}
		},
	}
	repo := &creativeRepo{q: db.New(mock)}
	now := time.Now()
	_, err := repo.Insert(context.Background(), &model.Creative{
		ID:         uuid.New(),
		OrgID:      uuid.New(),
		CampaignID: uuid.New(),
		Name:       "test creative",
		Type:       "image",
		FileURL:    "https://cdn.example.com/test.png",
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

### Baseline output

```go
```go
func TestCreativeRepo_Insert_DBError(t *testing.T) {
	tests := []struct {
		name      string
		creative  *model.Creative
		mock      func(m *mock_dbtx)
		expectErr bool
	}{
		{
			name: "DB error on insert",
			creative: &model.Creative{
				ID:            "some-id",
				OrgID:         "some-org-id",
				CampaignID:    "some-campaign-id",
				Name:          "Creative Name",
				Type:          model.CreativeTypeImage,
				FileURL:       "http://example.com/file.jpg",
				FileSizeBytes: 12345,
				PreviewURL:    "http://example.com/preview.jpg",
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			mock: func(m *mock_dbtx) {
				m.ExpectInsertCreative().WillReturnError(errSentinel)
			},
			expectErr: true,
		},
		{
			name: "successful insert",
			creative: &model.Creative{
				ID:            "some-id",
				OrgID:         "some-org-id",
				CampaignID:    "some-campaign-id",
				Name:          "Creative Name",
				Type:          model.CreativeTypeImage,
				FileURL:       "http://example.com/file.jpg",
				FileSizeBytes: 12345,
				PreviewURL:    "http://example.com/preview.jpg",
				IsActive:      true,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			mock: func(m *mock_dbtx) {
				m.ExpectInsertCreative().WillReturnRow(&db.InsertCreativeRow{
					ID:            "some-id",
					OrgID:         "some-org-id",
					CampaignID:    "some-campaign-id",
					Name:          "Creative Name",
					Type:          "image",
					FileUrl:       "http://example.com/file.jpg",
					FileSizeBytes: 12345,
					PreviewUrl:    "http://example.com/preview.jpg",
					IsActive:      true,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				})
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mockDBTX(t)
			tt.mock(m)

			repo := &creativeRepo{q: m}
			_, err := repo.Insert(context.Background(), tt.creative)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
```
```

### Evaluation

- Compiles: [✗] — `mock_dbtx` как тип + `.ExpectInsertCreative()` не существует; UUID как строки
- Uses mock_dbtx helpers correctly: [✗] — выдуманный mock-фреймворк
- Naming matches TestType_Method[_Scenario]: [✓] — `TestCreativeRepo_Insert_DBError`
- org_id / scope params covered: [✗] — org_id как строка "some-org-id"
- No extra blocks: [✗] — лишний success-кейс

**Score: Блок 1: 1/5 | Блок 2: 1/3 | Блок 3: 0/4 | Итого: 2/12**

---

## Example 4

### Input (user)

```go
func int64ToPgtypeInt8(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}
```

### Reference (expected)

```go
func TestInt64ToPgtypeInt8(t *testing.T) {
	tests := []struct {
		name      string
		input     *int64
		wantValid bool
		wantVal   int64
	}{
		{
			name:      "nil pointer yields invalid pgtype.Int8",
			input:     nil,
			wantValid: false,
		},
		{
			name:      "zero value",
			input:     int64Ptr(0),
			wantValid: true,
			wantVal:   0,
		},
		{
			name:      "positive value",
			input:     int64Ptr(100_000),
			wantValid: true,
			wantVal:   100_000,
		},
		{
			name:      "negative value",
			input:     int64Ptr(-1),
			wantValid: true,
			wantVal:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := int64ToPgtypeInt8(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", got.Valid, tt.wantValid)
			}
			if tt.wantValid && got.Int64 != tt.wantVal {
				t.Errorf("Int64 = %v, want %v", got.Int64, tt.wantVal)
			}
		})
	}
}
```

### Baseline output

```go
```go
func TestInt64ToPgtypeInt8(t *testing.T) {
	tests := []struct {
		name     string
		input    *int64
		expected pgtype.Int8
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: pgtype.Int8{},
		},
		{
			name:     "valid int64 input",
			input:    ptrToInt64(42),
			expected: pgtype.Int8{Int64: 42, Valid: true},
		},
		{
			name:     "negative int64 input",
			input:    ptrToInt64(-1),
			expected: pgtype.Int8{Int64: -1, Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := int64ToPgtypeInt8(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func ptrToInt64(i int64) *int64 {
	return &i
}
```
```

### Evaluation

- Compiles: [✓] — структурно верно, `ptrToInt64` определён инлайн
- Uses mock_dbtx helpers correctly: [n/a] — конвертер, DB не нужен
- Naming matches TestType_Method[_Scenario]: [✓] — `TestInt64ToPgtypeInt8`
- org_id / scope params covered: [n/a]
- No extra blocks: [✗] — определяет `ptrToInt64` хелпер (лишняя функция), но допустимо

**Score: Блок 1: 4/5 | Блок 2: 3/3 | Блок 3: 3/4 | Итого: 10/12**

---

## Example 5

### Input (user)

```go
func (r *organizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	row, err := r.q.GetOrganizationByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return toOrganization(row), nil
}

// Scenario: DBError
```

### Reference (expected)

```go
func TestOrganizationRepo_GetByID_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &organizationRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}
```

### Baseline output

```go
```go
func TestOrganizationRepo_GetByID_DBError(t *testing.T) {
	tests := []struct {
		name     string
		id       uuid.UUID
		mockFunc func()
		expected *model.Organization
		expectedErr error
	}{
		{
			name: "DB error",
			id:   uuid.New(),
			mockFunc: func() {
				mockDBTX.ExpectQuery("SELECT").WillReturnError(errSentinel)
			},
			expected:    nil,
			expectedErr: fmt.Errorf("get organization: %w", errSentinel),
		},
		{
			name: "Not found error",
			id:   uuid.New(),
			mockFunc: func() {
				mockDBTX.ExpectQuery("SELECT").WillReturnError(pgx.ErrNoRows)
			},
			expected:    nil,
			expectedErr: model.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			repo := &organizationRepo{q: mockDBTX}
			result, err := repo.GetByID(context.Background(), tt.id)

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
			if result != tt.expected {
				t.Errorf("expected result %v, got %v", tt.expected, result)
			}
		})
	}
}
```
```

### Evaluation

- Compiles: [✗] — `mockDBTX.ExpectQuery("SELECT")` sqlmock стиль; `errors.Is` на `fmt.Errorf` не работает
- Uses mock_dbtx helpers correctly: [✗] — sqlmock паттерн вместо `&mockDBTX{}`
- Naming matches TestType_Method[_Scenario]: [✓] — `TestOrganizationRepo_GetByID_DBError`
- org_id / scope params covered: [n/a] — organizations не имеет org_id
- No extra blocks: [✗] — лишний NotFound кейс (просили только DBError)

**Score: Блок 1: 2/5 | Блок 2: 1/3 | Блок 3: 0/4 | Итого: 3/12**

---

## Example 6

### Input (user)

```go
func (r *orgInviteRepo) GetByToken(ctx context.Context, token string) (*model.OrgInvite, error) {
	row, err := r.q.GetOrgInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get org invite by token: %w", err)
	}
	return toOrgInvite(row), nil
}

// Scenario: DBError
```

### Reference (expected)

```go
func TestOrgInviteRepo_GetByToken_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &orgInviteRepo{q: db.New(mock)}
	_, err := repo.GetByToken(context.Background(), "some-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}
```

### Baseline output

```go
```go
func TestOrgInviteRepo_GetByToken_DBError(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		mockSetup func()
		expected  *model.OrgInvite
		expectedErr error
	}{
		{
			name:  "DB error",
			token: "invalid-token",
			mockSetup: func() {
				mockDBTX.ExpectQuery("SELECT").WillReturnError(errSentinel)
			},
			expected:    nil,
			expectedErr: fmt.Errorf("get org invite by token: %w", errSentinel),
		},
		{
			name:  "no rows",
			token: "not-found-token",
			mockSetup: func() {
				mockDBTX.ExpectQuery("SELECT").WillReturnError(pgx.ErrNoRows)
			},
			expected:    nil,
			expectedErr: model.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			repo := &orgInviteRepo{q: mockDBTX}
			result, err := repo.GetByToken(context.Background(), tt.token)

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
			if result != tt.expected {
				t.Errorf("expected result %v, got %v", tt.expected, result)
			}
		})
	}
}
```
```

### Evaluation

- Compiles: [✗] — `mockDBTX.ExpectQuery("SELECT")` sqlmock стиль; `fmt.Errorf` в expectedErr
- Uses mock_dbtx helpers correctly: [✗] — sqlmock паттерн
- Naming matches TestType_Method[_Scenario]: [✓] — `TestOrgInviteRepo_GetByToken_DBError`
- org_id / scope params covered: [n/a] — метод не принимает orgID
- No extra blocks: [✗] — лишний "no rows" кейс

**Score: Блок 1: 2/5 | Блок 2: 1/3 | Блок 3: 0/4 | Итого: 3/12**

---

## Example 7

### Input (user)

```go
func (r *publisherAppRepo) GetByBundleID(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error) {
	row, err := r.q.GetPublisherAppByBundleID(ctx, db.GetPublisherAppByBundleIDParams{
		OrgID:    uuidToPgtype(orgID),
		BundleID: bundleID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get publisher app by bundle id: %w", err)
	}
	return toPublisherApp(row), nil
}

// Scenario: DBError
```

### Reference (expected)

```go
func TestPublisherAppRepo_GetByBundleID_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &publisherAppRepo{q: db.New(mock)}
	_, err := repo.GetByBundleID(context.Background(), uuid.New(), "com.example.app")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}
```

### Baseline output

```go
```go
func TestPublisherAppRepo_GetByBundleID_DBError(t *testing.T) {
	tests := []struct {
		name     string
		orgID    uuid.UUID
		bundleID string
		mock     func(m *mock_dbtx)
		expected *model.PublisherApp
		expectedErr error
	}{
		{
			name:     "DB error",
			orgID:    uuid.New(),
			bundleID: "test-bundle-id",
			mock: func(m *mock_dbtx) {
				m.On("GetPublisherAppByBundleID", mock.Anything, db.GetPublisherAppByBundleIDParams{
					OrgID:    uuidToPgtype(uuid.New()),
					BundleID: "test-bundle-id",
				}).Return(errSentinel)
			},
			expected:    nil,
			expectedErr: fmt.Errorf("get publisher app by bundle id: %w", errSentinel),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mock_dbtx{}
			tt.mock(m)
			repo := &publisherAppRepo{q: m}

			result, err := repo.GetByBundleID(context.Background(), tt.orgID, tt.bundleID)

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
			if result != tt.expected {
				t.Errorf("expected result %v, got %v", tt.expected, result)
			}
		})
	}
}
```
```

### Evaluation

- Compiles: [✗] — `mock_dbtx{}` + `.On()` testify стиль; не компилируется
- Uses mock_dbtx helpers correctly: [✗] — testify mock вместо `&mockDBTX{}`
- Naming matches TestType_Method[_Scenario]: [✓] — `TestPublisherAppRepo_GetByBundleID_DBError`
- org_id / scope params covered: [✓] — передаёт orgID
- No extra blocks: [✗] — `fmt.Errorf` в expectedErr не работает

**Score: Блок 1: 1/5 | Блок 2: 1/3 | Блок 3: 0/4 | Итого: 2/12**

---

## Example 8

### Input (user)

```go
func (r *publisherRuleRepo) ListByApp(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) ([]model.PublisherRule, int64, error) {
	rows, err := r.q.ListPublisherRulesByApp(ctx, db.ListPublisherRulesByAppParams{
		OrgID:     uuidToPgtype(orgID),
		AppID:     uuidToPgtype(appID),
		LimitVal:  limit,
		OffsetVal: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list publisher rules: %w", err)
	}

	total, err := r.q.CountPublisherRulesByApp(ctx, db.CountPublisherRulesByAppParams{
		OrgID: uuidToPgtype(orgID),
		AppID: uuidToPgtype(appID),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count publisher rules: %w", err)
	}

	rules := make([]model.PublisherRule, len(rows))
	for i, row := range rows {
		rules[i] = *toPublisherRule(row)
	}
	return rules, total, nil
}

// Scenario: DBError
```

### Reference (expected)

```go
func TestPublisherRuleRepo_ListByApp_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return nil, errSentinel("list failed")
		},
	}
	repo := &publisherRuleRepo{q: db.New(mock)}
	_, _, err := repo.ListByApp(context.Background(), uuid.New(), uuid.New(), 10, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

### Baseline output

```go
```go
func TestPublisherRuleRepo_ListByApp_DBError(t *testing.T) {
	repo := &publisherRuleRepo{q: mockDBTX()}
	orgID := uuid.New()
	appID := uuid.New()
	limit := int32(10)
	offset := int32(0)

	tests := []struct {
		name       string
		mockSetup  func()
		expected   []model.PublisherRule
		expectedTotal int64
		expectedErr error
	}{
		{
			name: "DB error on ListPublisherRulesByApp",
			mockSetup: func() {
				repo.q.(*mockDBTX).On("ListPublisherRulesByApp", mock.Anything, mock.Anything).Return(nil, errSentinel)
			},
			expected:   nil,
			expectedTotal: 0,
			expectedErr: fmt.Errorf("list publisher rules: %w", errSentinel),
		},
		{
			name: "DB error on CountPublisherRulesByApp",
			mockSetup: func() {
				repo.q.(*mockDBTX).On("ListPublisherRulesByApp", mock.Anything, mock.Anything).Return([]db.PublisherRuleRow{}, nil)
				repo.q.(*mockDBTX).On("CountPublisherRulesByApp", mock.Anything, mock.Anything).Return(int64(0), errSentinel)
			},
			expected:   nil,
			expectedTotal: 0,
			expectedErr: fmt.Errorf("count publisher rules: %w", errSentinel),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			rules, total, err := repo.ListByApp(context.Background(), orgID, appID, limit, offset)
			if !reflect.DeepEqual(rules, tt.expected) {
				t.Errorf("expected rules: %v, got: %v", tt.expected, rules)
			}
			if total != tt.expectedTotal {
				t.Errorf("expected total: %v, got: %v", tt.expectedTotal, total)
			}
			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("expected error: %v, got: %v", tt.expectedErr, err)
			}
		})
	}
}
```
```

### Evaluation

- Compiles: [✗] — `mockDBTX()` как вызов функции; `.On()` testify; `reflect.DeepEqual`
- Uses mock_dbtx helpers correctly: [✗] — testify mock + type assertion на `*mockDBTX`
- Naming matches TestType_Method[_Scenario]: [✓] — `TestPublisherRuleRepo_ListByApp_DBError`
- org_id / scope params covered: [✓] — передаёт orgID и appID
- No extra blocks: [✗] — `reflect.DeepEqual` для nil slice

**Score: Блок 1: 1/5 | Блок 2: 2/3 | Блок 3: 0/4 | Итого: 3/12**

---

## Example 9

### Input (user)

```go
func stringToPgtypeText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *v, Valid: true}
}
```

### Reference (expected)

```go
func TestStringToPgtypeText(t *testing.T) {
	tests := []struct {
		name      string
		input     *string
		wantValid bool
		wantStr   string
	}{
		{
			name:      "nil pointer yields invalid pgtype.Text",
			input:     nil,
			wantValid: false,
		},
		{
			name:      "empty string",
			input:     strPtr(""),
			wantValid: true,
			wantStr:   "",
		},
		{
			name:      "non-empty string",
			input:     strPtr("hello"),
			wantValid: true,
			wantStr:   "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringToPgtypeText(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", got.Valid, tt.wantValid)
			}
			if tt.wantValid && got.String != tt.wantStr {
				t.Errorf("String = %q, want %q", got.String, tt.wantStr)
			}
		})
	}
}
```

### Baseline output

```go
```go
func TestStringToPgtypeText(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected pgtype.Text
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: pgtype.Text{},
		},
		{
			name:     "valid string input",
			input:    ptr("test"),
			expected: pgtype.Text{String: "test", Valid: true},
		},
		{
			name:     "empty string input",
			input:    ptr(""),
			expected: pgtype.Text{String: "", Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringToPgtypeText(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func ptr(s string) *string {
	return &s
}
```
```

### Evaluation

- Compiles: [✓] — `ptr()` определён, структурно верно
- Uses mock_dbtx helpers correctly: [n/a] — конвертер
- Naming matches TestType_Method[_Scenario]: [✓] — `TestStringToPgtypeText`
- org_id / scope params covered: [n/a]
- No extra blocks: [✗] — определяет `ptr()` хелпер (у нас есть `strPtr`)

**Score: Блок 1: 4/5 | Блок 2: 3/3 | Блок 3: 3/4 | Итого: 10/12**

---

## Example 10

### Input (user)

```go
func timeToPgtypeDate(v *time.Time) pgtype.Date {
	if v == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *v, Valid: true}
}
```

### Reference (expected)

```go
func TestTimeToPgtypeDate(t *testing.T) {
	ref := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		input     *time.Time
		wantValid bool
	}{
		{
			name:      "nil pointer yields invalid pgtype.Date",
			input:     nil,
			wantValid: false,
		},
		{
			name:      "non-nil time",
			input:     &ref,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeToPgtypeDate(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", got.Valid, tt.wantValid)
			}
			if tt.wantValid && !got.Time.Equal(ref) {
				t.Errorf("Time = %v, want %v", got.Time, ref)
			}
		})
	}
}
```

### Baseline output

```go
```go
func TestTimeToPgtypeDate(t *testing.T) {
	tests := []struct {
		name     string
		input    *time.Time
		expected pgtype.Date
	}{
		{
			name:     "Nil time",
			input:    nil,
			expected: pgtype.Date{},
		},
		{
			name:     "Valid time",
			input:    func() *time.Time { t := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC); return &t }(),
			expected: pgtype.Date{Time: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeToPgtypeDate(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
```
```

### Evaluation

- Compiles: [✓] — инлайн-функция для указателя, структурно верно
- Uses mock_dbtx helpers correctly: [n/a] — конвертер
- Naming matches TestType_Method[_Scenario]: [✓] — `TestTimeToPgtypeDate`
- org_id / scope params covered: [n/a]
- No extra blocks: [✓] — чисто

**Score: Блок 1: 4/5 | Блок 2: 3/3 | Блок 3: 3/4 | Итого: 10/12**

---

## Summary

| # | Input | Блок 1 (0-5) | Блок 2 (0-3) | Блок 3 (0-4) | Итого (0-12) |
|---|-------|:---:|:---:|:---:|:---:|
| 1 | apiKeyRepo.Insert_DBError | 1 | 1 | 0 | 2 |
| 2 | campaignRepo.UpdateStatus_NotFound | 2 | 1 | 1 | 4 |
| 3 | creativeRepo.Insert_DBError | 1 | 1 | 0 | 2 |
| 4 | int64ToPgtypeInt8 | 4 | 3 | 3 | 10 |
| 5 | organizationRepo.GetByID_DBError | 2 | 1 | 0 | 3 |
| 6 | orgInviteRepo.GetByToken_DBError | 2 | 1 | 0 | 3 |
| 7 | publisherAppRepo.GetByBundleID_DBError | 1 | 1 | 0 | 2 |
| 8 | publisherRuleRepo.ListByApp_DBError | 1 | 2 | 0 | 3 |
| 9 | stringToPgtypeText | 4 | 3 | 3 | 10 |
| 10 | timeToPgtypeDate | 4 | 3 | 3 | 10 |
| **Среднее baseline** | | **2.2** | **1.7** | **1.0** | **4.9** |

### Выводы

- **Конвертеры** (4, 9, 10): модель справляется (~10/12) — простой паттерн, не нужны проектные хелперы
- **DB-error кейсы** (1-3, 5-8): полный провал (~2.7/12) — модель не знает наш `mockDBTX` и подставляет gomock/testify/sqlmock
- **Главная цель файнтюна**: научить модель использовать `&mockDBTX{queryRowFn:...}` + `db.New(mock)` + `errRow` + `errSentinel`

---

