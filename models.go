package main

import "strings"

type SheetRow struct {
	RowIndex  int    // 1-based sheet row index (Row 1 is header, data starts at Row 2)
	Name      string // Col A
	Run       string // Col B
	ModelIn   string // Col C
	ModelOut  string // Col D
	ActualIn  string // Col E
	ActualOut string // Col F
}

func (r SheetRow) IsComplete() bool {
	return strings.TrimSpace(r.ActualIn) != "" && strings.TrimSpace(r.ActualOut) != ""
}

type ClipTask struct {
	RowIndex       int
	Name           string
	Run            string
	ModelIn        string
	ModelOut       string
	ActualIn       string
	ActualOut      string
	DriveFileID    string
	TotalClips     int
	CompletedCount int
	QueuePosition  int
	ProgressPct    int
	IsCompleted    bool
	ErrorMessage   string
}
