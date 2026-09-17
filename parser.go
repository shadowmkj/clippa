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

func BuildPageData(rows []SheetRow, selectedRow int, driveResolver func(name string) string) *PageData {
	total := len(rows)
	if total == 0 {
		return &PageData{
			Clips:        nil,
			SelectedClip: nil,
		}
	}

	var clips []ClipItem
	completed := 0
	var firstPending *ClipItem
	var exactSelected *ClipItem

	for _, row := range rows {
		isDone := row.IsComplete()
		if isDone {
			completed++
		}

		driveID := ""
		if driveResolver != nil {
			driveID = driveResolver(row.Name)
		}

		item := ClipItem{
			RowIndex:    row.RowIndex,
			Name:        row.Name,
			Run:         row.Run,
			ModelIn:     row.ModelIn,
			ModelOut:    row.ModelOut,
			ActualIn:    row.ActualIn,
			ActualOut:   row.ActualOut,
			DriveFileID: driveID,
			IsComplete:  isDone,
		}

		if selectedRow > 0 && row.RowIndex == selectedRow {
			item.IsSelected = true
			exactSelected = &item
		}

		if !isDone && firstPending == nil {
			firstPending = &item
		}

		clips = append(clips, item)
	}

	// If no specific row selected or row not found, select first pending, or first clip
	var activeClip *ClipItem
	if exactSelected != nil {
		activeClip = exactSelected
	} else if firstPending != nil {
		activeClip = firstPending
	} else if len(clips) > 0 {
		activeClip = &clips[0]
	}

	if activeClip != nil {
		for i := range clips {
			if clips[i].RowIndex == activeClip.RowIndex {
				clips[i].IsSelected = true
				activeClip = &clips[i]
				break
			}
		}
	}

	progressPct := 0
	if total > 0 {
		progressPct = (completed * 100) / total
	}

	return &PageData{
		Clips:          clips,
		SelectedClip:   activeClip,
		TotalClips:     total,
		CompletedCount: completed,
		PendingCount:   total - completed,
		ProgressPct:    progressPct,
	}
}
