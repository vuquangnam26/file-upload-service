package main

import (
	"file-upload-service/bootstrap"
	"log"
	"net/http"
)

func main() {
	app := bootstrap.App()
	defer app.Close()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Println("Starting file-upload-service on :8088")
	if err := http.ListenAndServe(":8088", nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
