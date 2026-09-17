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
	FetchNextTask(ctx context.Context) (*ClipTask, error)
	UpdateActualCounts(ctx context.Context, rowIndex int, actualIn, actualOut string) error
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
	mux.HandleFunc("POST /submit", s.handleSubmit)
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	task, err := s.svc.FetchNextTask(r.Context())
	if err != nil {
		log.Printf("[Error] FetchNextTask failed: %v", err)
		task = &ClipTask{
			ErrorMessage: err.Error(),
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", task); err != nil {
		log.Printf("[Error] Failed to render index template: %v", err)
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
		task, _ := s.svc.FetchNextTask(r.Context())
		if task == nil {
			task = &ClipTask{RowIndex: rowIndex}
		}
		task.ErrorMessage = "Failed to update Google Sheet: " + err.Error()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = s.tmpl.ExecuteTemplate(w, "task_card.html", task)
		return
	}

	nextTask, err := s.svc.FetchNextTask(r.Context())
	if err != nil {
		log.Printf("[Error] FetchNextTask after submit failed: %v", err)
		nextTask = &ClipTask{
			ErrorMessage: "Updated sheet, but failed to fetch next task: " + err.Error(),
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "task_card.html", nextTask); err != nil {
		log.Printf("[Error] Failed to render task_card template: %v", err)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
