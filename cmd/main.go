package main

import (
	"encoding/json"
	"file-upload-service/bootstrap"
	"file-upload-service/internal/health"
	"log"
	"net/http"
)

func main() {
	app := bootstrap.App()
	defer app.Close()

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
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		report := health.Run(checkers)

		status := http.StatusOK
		if report.Status == health.StatusFail {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(report)
	})

	log.Println("Starting file-upload-service on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
