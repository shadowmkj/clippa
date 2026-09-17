package main

import (
	"fmt"
	"strings"
)

func parseCell(row []interface{}, colIdx int) string {
	if colIdx < len(row) && row[colIdx] != nil {
		return strings.TrimSpace(fmt.Sprint(row[colIdx]))
	}
	return ""
}

func ParseSheetRows(rawRows [][]interface{}) []SheetRow {
	if len(rawRows) <= 1 {
		return nil
	}

	var dataRows []SheetRow
	for i, raw := range rawRows[1:] {
		rowIndex := i + 2 // Row 1 is header
		row := SheetRow{
			RowIndex:  rowIndex,
			Name:      parseCell(raw, 0),
			Run:       parseCell(raw, 1),
			ModelIn:   parseCell(raw, 2),
			ModelOut:  parseCell(raw, 3),
			ActualIn:  parseCell(raw, 4),
			ActualOut: parseCell(raw, 5),
		}
		if row.Name != "" {
			dataRows = append(dataRows, row)
		}
	}
	return dataRows
}

func BuildTaskQueue(rows []SheetRow, driveResolver func(name string) string) *ClipTask {
	total := len(rows)
	if total == 0 {
		return &ClipTask{
			IsCompleted: true,
		}
	}

	completed := 0
	var firstPending *SheetRow
	firstPendingIdx := -1

	for idx, row := range rows {
		if row.IsComplete() {
			completed++
		} else if firstPending == nil {
			rowCopy := row
			firstPending = &rowCopy
			firstPendingIdx = idx
		}
	}

	progressPct := 0
	if total > 0 {
		progressPct = (completed * 100) / total
	}

	if firstPending == nil {
		return &ClipTask{
			TotalClips:     total,
			CompletedCount: completed,
			ProgressPct:    100,
			IsCompleted:    true,
		}
	}

	driveID := ""
	if driveResolver != nil {
		driveID = driveResolver(firstPending.Name)
	}

	return &ClipTask{
		RowIndex:       firstPending.RowIndex,
		Name:           firstPending.Name,
		Run:            firstPending.Run,
		ModelIn:        firstPending.ModelIn,
		ModelOut:       firstPending.ModelOut,
		ActualIn:       firstPending.ActualIn,
		ActualOut:      firstPending.ActualOut,
		DriveFileID:    driveID,
		TotalClips:     total,
		CompletedCount: completed,
		QueuePosition:  firstPendingIdx + 1,
		ProgressPct:    progressPct,
		IsCompleted:    false,
	}
}
