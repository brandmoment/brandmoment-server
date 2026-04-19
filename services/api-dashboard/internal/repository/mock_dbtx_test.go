package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// mockDBTX implements db.DBTX for unit testing repositories without a real DB.
type mockDBTX struct {
	execFn     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	queryFn    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func (m *mockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *mockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, args...)
	}
	return &emptyRows{}, nil
}

func (m *mockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &errRow{err: errors.New("no queryRowFn set")}
}

// errRow is a pgx.Row that always returns the given error on Scan.
type errRow struct {
	err error
}

func (r *errRow) Scan(dest ...interface{}) error {
	return r.err
}

// emptyRows is a pgx.Rows with no data.
type emptyRows struct{}

func (e *emptyRows) Close()                        {}
func (e *emptyRows) Err() error                    { return nil }
func (e *emptyRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (e *emptyRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (e *emptyRows) Next() bool                          { return false }
func (e *emptyRows) Scan(dest ...interface{}) error      { return nil }
func (e *emptyRows) Values() ([]interface{}, error)      { return nil, nil }
func (e *emptyRows) RawValues() [][]byte                 { return nil }
func (e *emptyRows) Conn() *pgx.Conn                     { return nil }
