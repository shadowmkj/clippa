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

type mockMasterDetailService struct {
	pageData    *PageData
	updatedRow  int
	updatedIn   string
	updatedOut  string
	syncInvoked bool
}

func (m *mockMasterDetailService) FetchPageData(ctx context.Context, selectedRow int) (*PageData, error) {
	return m.pageData, nil
}

func (m *mockMasterDetailService) UpdateActualCounts(ctx context.Context, rowIndex int, actualIn, actualOut string) error {
	m.updatedRow = rowIndex
	m.updatedIn = actualIn
	m.updatedOut = actualOut
	return nil
}

func (m *mockMasterDetailService) SyncDriveClips(ctx context.Context) (int, error) {
	m.syncInvoked = true
	return 2, nil
}

func TestServer_Routes(t *testing.T) {
	tmpl, err := template.ParseFiles(
		"templates/index.html",
		"templates/sidebar.html",
		"templates/clip_item.html",
		"templates/task_card.html",
	)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	svc := &mockMasterDetailService{
		pageData: &PageData{
			TotalClips: 1,
			Clips: []ClipItem{
				{RowIndex: 2, Name: "test_clip.mp4", Run: "1", IsSelected: true},
			},
			SelectedClip: &ClipItem{
				RowIndex: 2,
				Name:     "test_clip.mp4",
				Run:      "1",
			},
		},
	}

	mux := NewServer(svc, tmpl)

	// GET /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "test_clip.mp4") {
		t.Errorf("GET / failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	// GET /clip?row=2
	req = httptest.NewRequest(http.MethodGet, "/clip?row=2", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Errorf("GET /clip should return partial, got %d", rec.Code)
	}

	// POST /submit
	form := url.Values{"row_index": {"2"}, "actual_in": {"12"}, "actual_out": {"14"}}
	req = httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Saved successfully") {
		t.Errorf("POST /submit failed: %d, body: %s", rec.Code, rec.Body.String())
	}

	// POST /sync-drive
	req = httptest.NewRequest(http.MethodPost, "/sync-drive", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !svc.syncInvoked {
		t.Errorf("POST /sync-drive failed: %d, invoked=%v", rec.Code, svc.syncInvoked)
	}
}
