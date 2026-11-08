package http

import (
	"encoding/json"
	"file-upload-service/internal/service"
	"net/http"

	"github.com/google/uuid"
)

type FileHandler struct {
	fileService service.FileService
}

func NewFileHandler(fs service.FileService) *FileHandler {
	return &FileHandler{fileService: fs}
}

// RegisterRouters registers all file related routes
func (h *FileHandler) RegisterRouters(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/files/upload", h.Upload)
	mux.HandleFunc("GET /api/v1/files/{id}", h.GetByID)
}

// Upload handles POST /api/v1/files/upload
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// 1. Limit body size before parsing (HTTP layer guard)
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024) // 100MB
	// 2. Parse multipart form (32MB in memory, rest to disk)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse form: "+err.Error())
		return
	}
	// 3. Extract file from multipart form
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "field 'file' is required")
		return
	}
	defer file.Close()
	// 4. Delegate to FileService
	result, err := h.fileService.Upload(r.Context(), service.UploadInput{
		Reader:     file,
		Filename:   header.Filename,
		Size:       header.Size,
		UploadedBy: r.Header.Get("X-User-ID"), // From future auth middleware
	})
	if err != nil {
		// Differentiate validation errors vs server errors
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, result.File)
}

// GetByID handles GET /api/v1/files/{id}
func (h *FileHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Go 1.22+ supports PathValue
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid file id")
		return
	}
	file, err := h.fileService.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "file not found")
		return
	}
	respondJSON(w, http.StatusOK, file)
}

// --- helpers ---
type errorResponse struct {
	Error string `json:"error"`
}

func respondJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
func respondError(w http.ResponseWriter, code int, msg string) {
	respondJSON(w, code, errorResponse{Error: msg})
}
