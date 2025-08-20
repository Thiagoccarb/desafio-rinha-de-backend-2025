package infrastructure

import (
	"context"
	"database/sql"
	"payment-processor/config"
	"payment-processor/interfaces"
	"time"
)

type PostgresConnection struct {
	Conn *sql.DB
}

func (p *PostgresConnection) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result, err := p.Conn.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}
func NewPostgresConnection() interfaces.DatabaseConnection {
	config := config.LoadConfig()
	connString := config.Database.ConnectionString()
	db, _ := sql.Open("postgres", connString)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)
	return &PostgresConnection{Conn: db}
}

func (p *PostgresConnection) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := p.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}
