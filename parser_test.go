package main

import (
	"testing"
)

func TestParseSheetRows_ShortRowsAndPadded(t *testing.T) {
	raw := [][]interface{}{
		{"name", "run", "in", "out", "actual_in", "actual_out"},
		{"clip_1.mp4", "1", "10", "12", "10", "12"}, // Completed
		{"clip_2.mp4", "1", "8", "9"},                // Trailing empty omitted by API
		{"clip_3.mp4", "2", "15", "14", "15", ""},    // Partial actual_out empty
	}

	rows := ParseSheetRows(raw)
	if len(rows) != 3 {
		t.Fatalf("expected 3 data rows, got %d", len(rows))
	}

	if rows[0].RowIndex != 2 || rows[0].Name != "clip_1.mp4" || rows[0].IsComplete() != true {
		t.Errorf("row 0 mismatch: %+v", rows[0])
	}
	if rows[1].RowIndex != 3 || rows[1].Name != "clip_2.mp4" || rows[1].IsComplete() != false {
		t.Errorf("row 1 mismatch: %+v", rows[1])
	}
	if rows[2].RowIndex != 4 || rows[2].Name != "clip_3.mp4" || rows[2].IsComplete() != false {
		t.Errorf("row 2 mismatch: %+v", rows[2])
	}
}

func TestBuildTaskQueue_FindsFirstIncomplete(t *testing.T) {
	raw := [][]interface{}{
		{"name", "run", "in", "out", "actual_in", "actual_out"},
		{"clip_1.mp4", "1", "10", "12", "10", "12"},
		{"clip_2.mp4", "1", "8", "9", "", ""},
		{"clip_3.mp4", "2", "15", "14", "", ""},
	}

	rows := ParseSheetRows(raw)
	mockResolver := func(name string) string {
		return "drive-id-" + name
	}

	task := BuildTaskQueue(rows, mockResolver)
	if task.IsCompleted {
		t.Fatalf("expected task to not be completed")
	}
	if task.RowIndex != 3 {
		t.Errorf("expected RowIndex 3 (clip_2.mp4), got %d", task.RowIndex)
	}
	if task.Name != "clip_2.mp4" {
		t.Errorf("expected Name clip_2.mp4, got %s", task.Name)
	}
	if task.DriveFileID != "drive-id-clip_2.mp4" {
		t.Errorf("expected DriveFileID drive-id-clip_2.mp4, got %s", task.DriveFileID)
	}
	if task.TotalClips != 3 || task.CompletedCount != 1 || task.QueuePosition != 2 {
		t.Errorf("stats mismatch: Total=%d, Completed=%d, Pos=%d", task.TotalClips, task.CompletedCount, task.QueuePosition)
	}
	if task.ProgressPct != 33 {
		t.Errorf("expected progress 33%%, got %d%%", task.ProgressPct)
	}
}

func TestBuildTaskQueue_AllCompleted(t *testing.T) {
	raw := [][]interface{}{
		{"name", "run", "in", "out", "actual_in", "actual_out"},
		{"clip_1.mp4", "1", "10", "12", "10", "12"},
	}

	rows := ParseSheetRows(raw)
	task := BuildTaskQueue(rows, func(name string) string { return "" })
	if !task.IsCompleted {
		t.Fatalf("expected task to be marked completed")
	}
	if task.CompletedCount != 1 || task.TotalClips != 1 || task.ProgressPct != 100 {
		t.Errorf("stats mismatch for all completed: %+v", task)
	}
}
