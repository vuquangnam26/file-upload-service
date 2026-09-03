package bootstrap

import (
	"context"
	"file-upload-service/bootstrap/database"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Application struct {
	Env   *Env
	DB    database.Database
	MinIO *minio.Client
	Kafka *kgo.Client
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
	// 1. Khởi tạo MinIO Client
	minioClient, err := minio.New(env.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(env.MinioUser, env.MinioPassword, ""),
		Secure: env.MinioUseSSL,
	})
	if err != nil {
		log.Fatalf("failed to initialize MinIO client: %v", err)
	}
	// Tự động kiểm tra và tạo bucket nếu chưa tồn tại
	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	exists, err := minioClient.BucketExists(initCtx, env.MinioBucket)
	if err == nil && !exists {
		err = minioClient.MakeBucket(initCtx, env.MinioBucket, minio.MakeBucketOptions{})
		if err != nil {
			log.Printf("warning: failed to auto-create bucket %s: %v", env.MinioBucket, err)
		} else {
			log.Printf("MinIO bucket %q created successfully", env.MinioBucket)
		}
	}
	// 2. Khởi tạo Kafka Client (franz-go)
	kafkaClient, err := kgo.NewClient(
		kgo.SeedBrokers(env.KafkaBrokers...),
	)
	if err != nil {
		log.Fatalf("failed to initialize Kafka client: %v", err)
	}

	return &Application{
		Env:   env,
		DB:    db,
		MinIO: minioClient,
		Kafka: kafkaClient,
	}
}
func (app *Application) Close() {
	if err := database.CloseInstance(); err != nil {
		log.Printf("failed to close database: %v", err)
	}
	if app.Kafka != nil {
		app.Kafka.Close()
	}
}
