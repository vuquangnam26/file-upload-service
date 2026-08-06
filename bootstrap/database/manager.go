package database

import (
	"log"
	"sync"
)

var (
	instance Database
	once     sync.Once
	mu       sync.Mutex
)

func GetInstance(cfg DbConfig) Database {
	once.Do(func() {
		db, err := NewDatabase(cfg)
		if err != nil {
			log.Fatalf("failed to create database: %v", err)
		}
		if err := db.Connect(); err != nil {
			log.Fatalf("failed to connect database: %v", err)
		}
		if err := db.Ping(); err != nil {
			log.Fatalf("failed to ping database: %v", err)
		}
		log.Printf("database connected [driver=%s]", cfg.Driver)
		instance = db
	})
	return instance
}
func CloseInstance() error {
	if instance != nil {
		return instance.Close()
	}
	return nil
}
