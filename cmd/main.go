package main

import (
	"context"
	"encoding/json"
	"errors"
	"file-upload-service/bootstrap"
	"file-upload-service/internal/health"
	"file-upload-service/internal/queue"
	"file-upload-service/internal/repository"
	"file-upload-service/internal/service"
	"file-upload-service/internal/storage"
	handler "file-upload-service/internal/transport/http"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/uptrace/bun"
)

func main() {
	app := bootstrap.App()
	defer app.Close()
	bunDB := app.DB.GetDb().(*bun.DB)

	// --- khởi tạo checkers ---
	pgChecker := health.NewPostgreSQLHealthCheck(app.DB)
	minioChecker := health.NewMinIOHealthCheck(app.MinIO, app.Env.MinioBucket)
	kafkaChecker := health.NewKafkaHealthCheck(app.Kafka)

	checkers := []health.Checker{
		pgChecker,
		minioChecker,
		kafkaChecker,
	}

	// --- /healthz handler ---
	mux := http.NewServeMux()
	fileRepo := repository.NewPostgresFileRepository(bunDB)
	fileStorage := storage.NewMinIOStorage(app.MinIO, app.Env.MinioBucket)
	kafkaProducer := queue.NewKafkaProducer(app.Kafka)
	fileSvc := service.NewFileService(
		fileStorage,
		fileRepo,
		app.Env.MinioBucket,
		kafkaProducer,
		app.Env.KafkaTopicFileUploaded,
	)

	// Đăng ký routes
	fileHandler := handler.NewFileHandler(fileSvc)
	fileHandler.RegisterRouters(mux)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		report := health.Run(checkers)
		status := http.StatusOK
		if report.Status == health.StatusFail {
			status = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(report)
	})

	// 3. Khởi tạo HTTP Server
	server := &http.Server{
		Addr:         app.Env.ServerAddress,
		Handler:      mux,
		ReadTimeout:  15 * time.Minute, // Cần timeout lớn để hỗ trợ upload file to
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}
	// 4. Chạy Server trong goroutine
	go func() {
		log.Printf("Starting file-upload-service on %s", app.Env.ServerAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()
	// 5. Graceful Shutdown (Bắt tín hiệu Ctrl+C / SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited cleanly")
}
