package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestTemplates_RenderMasterDetail(t *testing.T) {
	tmpl, err := template.ParseFiles(
		"templates/index.html",
		"templates/sidebar.html",
		"templates/clip_item.html",
		"templates/task_card.html",
	)
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	page := PageData{
		TotalClips:     2,
		CompletedCount: 1,
		PendingCount:   1,
		ProgressPct:    50,
		Clips: []ClipItem{
			{RowIndex: 2, Name: "clip_done.mp4", Run: "1", IsComplete: true, IsSelected: false},
			{RowIndex: 3, Name: "clip_pending.mp4", Run: "1", IsComplete: false, IsSelected: true, DriveFileID: "drive-xyz"},
		},
		SelectedClip: &ClipItem{
			RowIndex:    3,
			Name:        "clip_pending.mp4",
			Run:         "1",
			DriveFileID: "drive-xyz",
			IsComplete:  false,
		},
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "index.html", page); err != nil {
		t.Fatalf("failed to render index: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "clip_done.mp4") || !strings.Contains(html, "clip_pending.mp4") {
		t.Errorf("expected both clips in sidebar HTML")
	}
	if !strings.Contains(html, "toggle-sidebar-btn") {
		t.Errorf("expected sidebar toggle button in HTML")
	}
}
