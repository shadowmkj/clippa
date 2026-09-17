package main

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type mockTaskService struct {
	task        *ClipTask
	updateError error
	updatedRow  int
	updatedIn   string
	updatedOut  string
}

func (m *mockTaskService) FetchNextTask(ctx context.Context) (*ClipTask, error) {
	return m.task, nil
}

func (m *mockTaskService) UpdateActualCounts(ctx context.Context, rowIndex int, actualIn, actualOut string) error {
	m.updatedRow = rowIndex
	m.updatedIn = actualIn
	m.updatedOut = actualOut
	return m.updateError
}

func TestServer_Index(t *testing.T) {
	tmpl, err := template.ParseFiles("templates/index.html", "templates/task_card.html")
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	mockSvc := &mockTaskService{
		task: &ClipTask{
			RowIndex:    2,
			Name:        "traffic_video.mp4",
			DriveFileID: "drive-123",
			TotalClips:  5,
		},
	}

	mux := NewServer(mockSvc, tmpl)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "traffic_video.mp4") {
		t.Errorf("expected traffic_video.mp4 in body")
	}
}

func TestServer_Submit_HTMX(t *testing.T) {
	tmpl, err := template.ParseFiles("templates/index.html", "templates/task_card.html")
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	mockSvc := &mockTaskService{
		task: &ClipTask{
			RowIndex:    3,
			Name:        "next_clip.mp4",
			DriveFileID: "drive-456",
			TotalClips:  5,
		},
	}

	mux := NewServer(mockSvc, tmpl)

	formData := url.Values{
		"row_index":  {"2"},
		"actual_in":  {"11"},
		"actual_out": {"14"},
	}

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
	if mockSvc.updatedRow != 2 || mockSvc.updatedIn != "11" || mockSvc.updatedOut != "14" {
		t.Errorf("expected update on row 2 (11, 14), got row %d (%s, %s)", mockSvc.updatedRow, mockSvc.updatedIn, mockSvc.updatedOut)
	}
	// Should return ONLY the partial, not the full <html> document
	if strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Errorf("HTMX response should be a partial, but contained <!DOCTYPE html>")
	}
	if !strings.Contains(rec.Body.String(), "next_clip.mp4") {
		t.Errorf("expected next_clip.mp4 in response partial")
	}
}
