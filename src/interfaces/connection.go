package interfaces

import (
	"context"
	"database/sql"
)

type DatabaseConnection interface {
	Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}
