package bootstrap

import (
	"file-upload-service/bootstrap/database"
	"log"
)

type Application struct {
	Env *Env
	DB  database.Database
}

func App() *Application {
	env := NewEnv()
	dbCfg := database.DbConfig{
		Driver:   env.DBDriver,
		Host:     env.DBHost,
		Port:     env.DBPort,
		User:     env.DBUser,
		Password: env.DBPass,
		DbName:   env.DBName,
	}
	db := database.GetInstance(dbCfg)

	return &Application{
		Env: env,
		DB:  db,
	}
}
func (app *Application) Close() {
	if err := database.CloseInstance(); err != nil {
		log.Printf("failed to close database: %v", err)
	}
}
