package main

import (
	"context"
	"encoding/json"
	"errors"
	"file-upload-service/bootstrap"
	"file-upload-service/internal/health"
	"file-upload-service/internal/queue"
	"file-upload-service/internal/repository"
	"file-upload-service/internal/scanner"
	"file-upload-service/internal/service"
	"file-upload-service/internal/storage"
	"file-upload-service/internal/thumbnail"
	handler "file-upload-service/internal/transport/http"
	"file-upload-service/internal/worker"
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

	// --- Initialize health checkers ---
	pgChecker := health.NewPostgreSQLHealthCheck(app.DB)
	minioChecker := health.NewMinIOHealthCheck(app.MinIO, app.Env.MinioBucket)
	kafkaChecker := health.NewKafkaHealthCheck(app.Kafka)

	checkers := []health.Checker{
		pgChecker,
		minioChecker,
		kafkaChecker,
	}

	// --- Initialize Handlers & Services ---
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
	// --- Initialize ClamAV Scanner ---
	clamavScanner := scanner.NewClamAVScanner(app.Env.ClamAVHost, app.Env.ClamAVPort, 30*time.Second)
	// --- Initialize VirusScanWorker ---
	virusWorker := worker.NewVirusScanWorker(
		fileStorage,
		fileRepo,
		clamavScanner,
		kafkaProducer,
		app.Env.KafkaTopicFileScanned,
	)
	// --- Initialize Kafka Consumer for Virus Scan ---
	virusConsumer, err := queue.NewKafkaConsumer(
		app.Env.KafkaBrokers,
		app.Env.KafkaGroupVirusScan,
		[]string{app.Env.KafkaTopicFileUploaded},
		virusWorker.ProcessEvent,
	)
	if err != nil {
		log.Fatalf("failed to create virus consumer: %v", err)
	}
	defer virusConsumer.Close()
	// Run consumers in background goroutine with graceful shutdown context
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()
	go virusConsumer.Start(consumerCtx)

	// --- Initialize Thumbnail Worker ---
	thumbGenerator := thumbnail.NewImagingGenerator(300, 300)
	thumbWorker := worker.NewThumbnailWorker(
		fileStorage, fileRepo, thumbGenerator, kafkaProducer,
		app.Env.KafkaTopicThumbnailDone,
	)
	thumbConsumer, err := queue.NewKafkaConsumer(
		app.Env.KafkaBrokers,
		app.Env.KafkaGroupThumbnail,
		[]string{app.Env.KafkaTopicFileUploaded},
		thumbWorker.ProcessEvent,
	)
	if err != nil {
		log.Fatalf("failed to create thumbnail consumer: %v", err)
	}
	defer thumbConsumer.Close()
	go thumbConsumer.Start(consumerCtx)
	// Register routes
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

	// 3. Initialize HTTP Server
	server := &http.Server{
		Addr:         app.Env.ServerAddress,
		Handler:      mux,
		ReadTimeout:  15 * time.Minute, // High timeout to support large file uploads
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}
	// 4. Run Server in goroutine
	go func() {
		log.Printf("Starting file-upload-service on %s", app.Env.ServerAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()
	// 5. Graceful Shutdown (Listen for SIGINT / SIGTERM)
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
