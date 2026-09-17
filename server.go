package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type TaskService interface {
	FetchPageData(ctx context.Context, selectedRow int) (*PageData, error)
	UpdateActualCounts(ctx context.Context, rowIndex int, actualIn, actualOut string) error
	SyncDriveClips(ctx context.Context) (int, error)
}

type Server struct {
	svc  TaskService
	tmpl *template.Template
}

func NewServer(svc TaskService, tmpl *template.Template) *http.ServeMux {
	s := &Server{
		svc:  svc,
		tmpl: tmpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /clip", s.handleClip)
	mux.HandleFunc("POST /submit", s.handleSubmit)
	mux.HandleFunc("POST /sync-drive", s.handleSyncDrive)
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	pageData, err := s.svc.FetchPageData(r.Context(), 0)
	if err != nil {
		log.Printf("[Error] FetchPageData failed: %v", err)
		pageData = &PageData{ErrorMessage: err.Error()}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", pageData); err != nil {
		log.Printf("[Error] Failed to render index template: %v", err)
	}
}

func (s *Server) handleClip(w http.ResponseWriter, r *http.Request) {
	rowStr := r.URL.Query().Get("row")
	row, _ := strconv.Atoi(rowStr)

	pageData, err := s.svc.FetchPageData(r.Context(), row)
	if err != nil {
		log.Printf("[Error] FetchPageData for row %d failed: %v", row, err)
		pageData = &PageData{ErrorMessage: err.Error()}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "task_card.html", pageData); err != nil {
		log.Printf("[Error] Failed to render task_card template: %v", err)
	}
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	rowIndexStr := r.FormValue("row_index")
	actualIn := strings.TrimSpace(r.FormValue("actual_in"))
	actualOut := strings.TrimSpace(r.FormValue("actual_out"))

	rowIndex, err := strconv.Atoi(rowIndexStr)
	if err != nil || rowIndex < 2 {
		http.Error(w, "Invalid row index", http.StatusBadRequest)
		return
	}

	if err := s.svc.UpdateActualCounts(r.Context(), rowIndex, actualIn, actualOut); err != nil {
		log.Printf("[Error] UpdateActualCounts failed for row %d: %v", rowIndex, err)
		pageData, _ := s.svc.FetchPageData(r.Context(), rowIndex)
		if pageData == nil {
			pageData = &PageData{}
		}
		pageData.ErrorMessage = "Failed to update Google Sheet: " + err.Error()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = s.tmpl.ExecuteTemplate(w, "task_card.html", pageData)
		return
	}

	pageData, err := s.svc.FetchPageData(r.Context(), rowIndex)
	if err != nil {
		log.Printf("[Error] FetchPageData after submit failed: %v", err)
		pageData = &PageData{
			ErrorMessage: "Saved to sheet, but failed to refresh data: " + err.Error(),
		}
	} else {
		pageData.SavedSuccess = true
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "task_card.html", pageData); err != nil {
		log.Printf("[Error] Failed to render task_card template: %v", err)
	}
}

func (s *Server) handleSyncDrive(w http.ResponseWriter, r *http.Request) {
	count, err := s.svc.SyncDriveClips(r.Context())
	if err != nil {
		log.Printf("[Error] SyncDriveClips failed: %v", err)
	} else {
		log.Printf("[Sync] Appended %d clips from Drive to Sheet", count)
	}

	pageData, err := s.svc.FetchPageData(r.Context(), 0)
	if err != nil {
		log.Printf("[Error] FetchPageData after sync failed: %v", err)
		pageData = &PageData{ErrorMessage: err.Error()}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", pageData); err != nil {
		log.Printf("[Error] Failed to render index template: %v", err)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
