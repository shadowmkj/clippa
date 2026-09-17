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

func TestBuildPageData_Empty(t *testing.T) {
	page := BuildPageData(nil, 0, func(name string) string { return "" })
	if page.TotalClips != 0 || page.CompletedCount != 0 || page.SelectedClip != nil {
		t.Errorf("expected empty page data, got %+v", page)
	}
}

func TestBuildPageData_SelectionAndStats(t *testing.T) {
	rows := []SheetRow{
		{RowIndex: 2, Name: "clip_1.mp4", Run: "1", ModelIn: "10", ModelOut: "12", ActualIn: "10", ActualOut: "12"}, // Done
		{RowIndex: 3, Name: "clip_2.mp4", Run: "1", ModelIn: "8", ModelOut: "9", ActualIn: "", ActualOut: ""},        // Pending
		{RowIndex: 4, Name: "clip_3.mp4", Run: "2", ModelIn: "15", ModelOut: "14", ActualIn: "", ActualOut: ""},     // Pending
	}

	mockResolver := func(name string) string { return "drive-" + name }

	// Default selection (selectedRow == 0) should pick first pending (clip_2.mp4, row 3)
	page := BuildPageData(rows, 0, mockResolver)
	if page.TotalClips != 3 || page.CompletedCount != 1 || page.PendingCount != 2 || page.ProgressPct != 33 {
		t.Errorf("stats mismatch: Total=%d, Done=%d, Pending=%d, Pct=%d", page.TotalClips, page.CompletedCount, page.PendingCount, page.ProgressPct)
	}
	if page.SelectedClip == nil || page.SelectedClip.RowIndex != 3 {
		t.Fatalf("expected selected clip row 3, got %+v", page.SelectedClip)
	}
	if !page.Clips[0].IsComplete || page.Clips[1].IsComplete {
		t.Errorf("clip complete status mismatch: clip1=%v, clip2=%v", page.Clips[0].IsComplete, page.Clips[1].IsComplete)
	}
	if !page.Clips[1].IsSelected || page.Clips[0].IsSelected {
		t.Errorf("clip selection mismatch: clip0=%v, clip1=%v", page.Clips[0].IsSelected, page.Clips[1].IsSelected)
	}

	// Explicit selection for row 2
	page2 := BuildPageData(rows, 2, mockResolver)
	if page2.SelectedClip == nil || page2.SelectedClip.RowIndex != 2 {
		t.Fatalf("expected selected clip row 2, got %+v", page2.SelectedClip)
	}
}
