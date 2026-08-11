package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type postgresDB struct {
	cfg DbConfig
	db  *bun.DB
}

func init() {
	Register("postgres", func(cfg DbConfig) Database {
		return &postgresDB{cfg: cfg}
	})
}

func (p *postgresDB) Connect() error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.cfg.User, p.cfg.Password, p.cfg.Host, p.cfg.Port, p.cfg.DbName,
	)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	// Cấu hình connection pool
	sqldb.SetMaxOpenConns(25)
	sqldb.SetMaxIdleConns(10)
	sqldb.SetConnMaxLifetime(5 * time.Minute)

	p.db = bun.NewDB(sqldb, pgdialect.New())
	return nil
}

func (p *postgresDB) GetDb() any {
	return p.db
}

func (p *postgresDB) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *postgresDB) Ping() error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}
	// Thực sự ping DB với timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.db.PingContext(ctx)
}
