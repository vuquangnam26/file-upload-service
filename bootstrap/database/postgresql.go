package database

import (
	"database/sql"
	"fmt"

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
		p.cfg.DbName, p.cfg.Password, p.cfg.Host, p.cfg.Port, p.cfg.DbName,
	)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
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
		return fmt.Errorf("database not connected")
	}

	return nil
}
