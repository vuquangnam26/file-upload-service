package database

import "fmt"

type Database interface {
	Connect() error
	GetDb() any
	Close() error
	Ping() error
}

type DbConfig struct {
	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	DbName   string
}

var registry = map[string]func(DbConfig) Database{}

func Register(driver string, fn func(DbConfig) Database) {
	registry[driver] = fn
}

func NewDatabase(cfg DbConfig) (Database, error) {
	fn, ok := registry[cfg.Driver]
	if !ok {
		return nil, fmt.Errorf("unregistered driver: %s", cfg.Driver)
	}
	return fn(cfg), nil
}
