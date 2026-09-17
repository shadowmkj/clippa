package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestTemplates_RenderTaskCard_Pending(t *testing.T) {
	tmpl, err := template.ParseFiles("templates/index.html", "templates/task_card.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	task := ClipTask{
		RowIndex:       2,
		Name:           "sample_traffic.mp4",
		Run:            "1",
		ModelIn:        "14",
		ModelOut:       "8",
		DriveFileID:    "1AbC_xyz123",
		TotalClips:     10,
		CompletedCount: 3,
		QueuePosition:  4,
		ProgressPct:    30,
		IsCompleted:    false,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "task_card.html", task); err != nil {
		t.Fatalf("failed to execute task_card template: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "sample_traffic.mp4") {
		t.Errorf("expected sample_traffic.mp4 in card html")
	}
	if !strings.Contains(html, "https://drive.google.com/file/d/1AbC_xyz123/preview") {
		t.Errorf("expected google drive preview iframe url")
	}
	if !strings.Contains(html, `hx-post="/submit"`) {
		t.Errorf("expected hx-post attribute in form")
	}
	if !strings.Contains(html, `hx-target="#task-card"`) {
		t.Errorf("expected hx-target=#task-card in form")
	}
}

func TestTemplates_RenderTaskCard_Completed(t *testing.T) {
	tmpl, err := template.ParseFiles("templates/index.html", "templates/task_card.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	task := ClipTask{
		TotalClips:     10,
		CompletedCount: 10,
		ProgressPct:    100,
		IsCompleted:    true,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "task_card.html", task); err != nil {
		t.Fatalf("failed to execute completed template: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "All Clips Completed") {
		t.Errorf("expected completion message in card html")
	}
}
