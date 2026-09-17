# Master-Detail Multi-User Review UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade Clippa into a master-detail multi-user review application with a collapsible sidebar clip list, non-sequential editing, one-click Google Drive sync, and HTMX Out-of-Band status updates.

**Architecture:** Go `net/http` server serving a 2-column responsive layout with embedded CSS. The sidebar renders all clips with filterable status badges (`✓ Done` vs `⏳ Pending`). Clicking any clip fetches its editor partial via `GET /clip?row={N}`. Submitting saves to Google Sheets columns E & F, displays a save confirmation, and triggers an HTMX Out-of-Band swap on the sidebar item.

**Tech Stack:** Go 1.22+, `google.golang.org/api/sheets/v4`, `google.golang.org/api/drive/v3`, HTMX 2.0.4 (CDN), Vanilla HTML5/CSS.

**Spec:** `docs/superpowers/specs/2026-09-17-master-detail-multi-user-review-ui-design.md`

## Global Constraints
- Strict zero-build: No npm, yarn, Vite, Webpack, or Tailwind.
- Pure Go build without CGO.
- All code blocks must be complete, functional, and compilable.
- Support full screen mode via sidebar collapse toggle.

---

### Task 1: Update Data Models & Page Data Parser

**Files:**
- Modify: `models.go`
- Modify: `parser.go`
- Modify: `parser_test.go`

**Interfaces:**
- Consumes: `SheetRow`
- Produces: `type ClipItem struct`, `type PageData struct`, `func BuildPageData(rows []SheetRow, selectedRow int, driveResolver func(name string) string) *PageData`

- [ ] **Step 1: Write failing tests for `BuildPageData`**

```go
package main

import (
	"testing"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestBuildPageData ./...`  
Expected: FAIL (undefined: `BuildPageData`)

- [ ] **Step 3: Update `models.go` and `parser.go`**

`models.go`:
```go
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

type ClipItem struct {
	RowIndex    int
	Name        string
	Run         string
	ModelIn     string
	ModelOut    string
	ActualIn    string
	ActualOut   string
	DriveFileID string
	IsComplete  bool
	IsSelected  bool
}

type PageData struct {
	Clips          []ClipItem
	SelectedClip   *ClipItem
	TotalClips     int
	CompletedCount int
	PendingCount   int
	ProgressPct    int
	SavedSuccess   bool
	ErrorMessage   string
}
```

`parser.go`:
```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestBuildPageData ./...`  
Expected: PASS

---

### Task 2: Implement Drive Sync & Append in Google Workspace Client

**Files:**
- Modify: `drive_cache.go`
- Modify: `sheets.go`
- Modify: `drive_cache_test.go`

**Interfaces:**
- Consumes: `DriveCache`, `GoogleWorkspaceClient`
- Produces:
  - `func (d *DriveCache) GetAllFileNames() []string`
  - `func (g *GoogleWorkspaceClient) SyncDriveClips(ctx context.Context) (int, error)`
  - `func (g *GoogleWorkspaceClient) FetchPageData(ctx context.Context, selectedRow int) (*PageData, error)`

- [ ] **Step 1: Write tests for `GetAllFileNames`**

```go
package main

import (
	"testing"
)

func TestDriveCache_GetAllFileNames(t *testing.T) {
	cache := &DriveCache{
		folderID: "test-folder",
		fileMap:  make(map[string]string),
	}

	cache.Set("clip_a.mp4", "id_a")
	cache.Set("clip_b.mp4", "id_b")

	names := cache.GetAllFileNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 filenames, got %d", len(names))
	}
}
```

- [ ] **Step 2: Update `drive_cache.go` and `sheets.go`**

`drive_cache.go`:
```go
func (d *DriveCache) GetAllFileNames() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var names []string
	for name := range d.fileMap {
		names = append(names, name)
	}
	return names
}
```

`sheets.go`:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"sort"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type GoogleWorkspaceClient struct {
	cfg        *Config
	sheetsSrv  *sheets.Service
	driveCache *DriveCache
}

func NewGoogleWorkspaceClient(ctx context.Context, cfg *Config) (*GoogleWorkspaceClient, error) {
	var opts []option.ClientOption

	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	opts = append(opts, option.WithScopes(
		sheets.SpreadsheetsScope,
		drive.DriveReadonlyScope,
	))

	sheetsSrv, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	driveSrv, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	driveCache := NewDriveCache(driveSrv, cfg.DriveFolderID)
	if err := driveCache.Refresh(ctx); err != nil {
		log.Printf("[Warning] Drive cache pre-fetch failed: %v", err)
	}

	return &GoogleWorkspaceClient{
		cfg:        cfg,
		sheetsSrv:  sheetsSrv,
		driveCache: driveCache,
	}, nil
}

func (g *GoogleWorkspaceClient) FetchPageData(ctx context.Context, selectedRow int) (*PageData, error) {
	readRange := fmt.Sprintf("%s!A:F", g.cfg.SheetName)
	resp, err := g.sheetsSrv.Spreadsheets.Values.Get(g.cfg.SpreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet range %s: %w", readRange, err)
	}

	rows := ParseSheetRows(resp.Values)
	pageData := BuildPageData(rows, selectedRow, g.driveCache.Resolve)
	return pageData, nil
}

func (g *GoogleWorkspaceClient) UpdateActualCounts(ctx context.Context, rowIndex int, actualIn, actualOut string) error {
	updateRange := fmt.Sprintf("%s!E%d:F%d", g.cfg.SheetName, rowIndex, rowIndex)
	vr := &sheets.ValueRange{
		Values: [][]interface{}{
			{actualIn, actualOut},
		},
	}

	_, err := g.sheetsSrv.Spreadsheets.Values.Update(g.cfg.SpreadsheetID, updateRange, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("failed to update sheet range %s: %w", updateRange, err)
	}

	return nil
}

func (g *GoogleWorkspaceClient) SyncDriveClips(ctx context.Context) (int, error) {
	if err := g.driveCache.Refresh(ctx); err != nil {
		return 0, fmt.Errorf("failed to refresh drive files: %w", err)
	}

	readRange := fmt.Sprintf("%s!A:F", g.cfg.SheetName)
	resp, err := g.sheetsSrv.Spreadsheets.Values.Get(g.cfg.SpreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to read sheet: %w", err)
	}

	existingRows := ParseSheetRows(resp.Values)
	existingNames := make(map[string]bool)
	for _, r := range existingRows {
		existingNames[r.Name] = true
	}

	driveFiles := g.driveCache.GetAllFileNames()
	sort.Strings(driveFiles)

	var newRows [][]interface{}
	// If sheet is completely empty, ensure header row
	if len(resp.Values) == 0 {
		newRows = append(newRows, []interface{}{"name", "run", "in", "out", "actual_in", "actual_out"})
	}

	for _, name := range driveFiles {
		if !existingNames[name] {
			newRows = append(newRows, []interface{}{name, "1", "", "", "", ""})
		}
	}

	if len(newRows) == 0 {
		return 0, nil
	}

	appendRange := fmt.Sprintf("%s!A:F", g.cfg.SheetName)
	vr := &sheets.ValueRange{Values: newRows}
	_, err = g.sheetsSrv.Spreadsheets.Values.Append(g.cfg.SpreadsheetID, appendRange, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()

	if err != nil {
		return 0, fmt.Errorf("failed to append rows to sheet: %w", err)
	}

	log.Printf("[Sync] Appended %d missing clips to Google Sheet", len(newRows))
	return len(newRows), nil
}
```

- [ ] **Step 3: Run tests to verify**

Run: `go test -v ./...`  
Expected: PASS

---

### Task 3: Refactor UI Templates & Collapsible 2-Column CSS

**Files:**
- Modify: `templates/index.html`
- Create: `templates/sidebar.html`
- Create: `templates/clip_item.html`
- Modify: `templates/task_card.html`
- Modify: `templates_test.go`

**Interfaces:**
- Consumes: `PageData`, `ClipItem`
- Produces: Renderable master-detail templates with HTMX OOB updates and sidebar collapsing

- [ ] **Step 1: Write template tests for Master-Detail rendering**

```go
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
```

- [ ] **Step 2: Implement templates**

`templates/index.html`:
```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Clippa — Traffic Ground Truth Annotator</title>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <style>
    :root {
      --bg: #0f172a;
      --card-bg: #1e293b;
      --card-border: #334155;
      --sidebar-bg: #111827;
      --item-hover: #1e293b;
      --item-selected: #1e3a8a;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --primary: #3b82f6;
      --primary-hover: #2563eb;
      --success: #10b981;
      --warning: #f59e0b;
      --danger: #ef4444;
      --input-bg: #0f172a;
      --input-border: #475569;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background-color: var(--bg);
      color: var(--text);
      height: 100vh;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
    header {
      background-color: var(--card-bg);
      border-bottom: 1px solid var(--card-border);
      padding: 0.75rem 1.5rem;
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-shrink: 0;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      font-size: 1.2rem;
      font-weight: 700;
    }
    .brand span { color: var(--primary); }
    .header-actions {
      display: flex;
      align-items: center;
      gap: 0.75rem;
    }
    .btn-sm {
      padding: 0.4rem 0.8rem;
      font-size: 0.85rem;
      font-weight: 600;
      border-radius: 6px;
      border: 1px solid var(--card-border);
      background-color: var(--card-bg);
      color: var(--text);
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      gap: 0.4rem;
      transition: all 0.2s;
    }
    .btn-sm:hover {
      background-color: var(--card-border);
      border-color: var(--primary);
    }
    .btn-primary-sm {
      background-color: var(--primary);
      border-color: var(--primary);
      color: white;
    }
    .btn-primary-sm:hover {
      background-color: var(--primary-hover);
    }
    .app-layout {
      display: flex;
      flex: 1;
      height: calc(100vh - 57px);
      overflow: hidden;
    }
    .sidebar {
      width: 360px;
      background-color: var(--sidebar-bg);
      border-right: 1px solid var(--card-border);
      display: flex;
      flex-direction: column;
      transition: width 0.25s ease, opacity 0.25s ease;
      flex-shrink: 0;
      overflow: hidden;
    }
    .sidebar.collapsed {
      width: 0;
      border-right: none;
      opacity: 0;
    }
    .sidebar-header {
      padding: 1rem;
      border-bottom: 1px solid var(--card-border);
    }
    .stats-row {
      display: flex;
      justify-content: space-between;
      font-size: 0.85rem;
      color: var(--text-muted);
      margin-bottom: 0.75rem;
    }
    .progress-bar-container {
      width: 100%;
      height: 6px;
      background-color: var(--input-bg);
      border-radius: 9999px;
      overflow: hidden;
      margin-bottom: 0.75rem;
    }
    .progress-bar {
      height: 100%;
      background-color: var(--success);
      transition: width 0.3s ease;
    }
    .filter-group {
      display: flex;
      gap: 0.4rem;
    }
    .filter-btn {
      flex: 1;
      padding: 0.35rem;
      font-size: 0.75rem;
      font-weight: 600;
      border-radius: 4px;
      border: 1px solid var(--card-border);
      background-color: transparent;
      color: var(--text-muted);
      cursor: pointer;
      text-align: center;
    }
    .filter-btn.active {
      background-color: var(--primary);
      color: white;
      border-color: var(--primary);
    }
    .clip-list {
      flex: 1;
      overflow-y: auto;
      padding: 0.5rem;
      display: flex;
      flex-direction: column;
      gap: 0.4rem;
    }
    .clip-item {
      padding: 0.75rem;
      border-radius: 8px;
      background-color: var(--card-bg);
      border: 1px solid var(--card-border);
      cursor: pointer;
      display: flex;
      flex-direction: column;
      gap: 0.35rem;
      transition: all 0.15s;
    }
    .clip-item:hover {
      background-color: var(--item-hover);
      border-color: var(--primary);
    }
    .clip-item.selected {
      background-color: var(--item-selected);
      border-color: #60a5fa;
      box-shadow: 0 0 0 1px #60a5fa;
    }
    .clip-item-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    .clip-name {
      font-size: 0.9rem;
      font-weight: 600;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      max-width: 220px;
    }
    .clip-meta {
      display: flex;
      justify-content: space-between;
      font-size: 0.75rem;
      color: var(--text-muted);
    }
    .badge {
      display: inline-block;
      padding: 0.2rem 0.5rem;
      font-size: 0.7rem;
      font-weight: 600;
      border-radius: 9999px;
      background-color: var(--card-border);
      color: var(--text-muted);
    }
    .badge-done { background-color: rgba(16, 185, 129, 0.2); color: #34d399; }
    .badge-pending { background-color: rgba(245, 158, 11, 0.2); color: #fbbf24; }
    .main-content {
      flex: 1;
      overflow-y: auto;
      padding: 1.5rem;
      display: flex;
      flex-direction: column;
      align-items: center;
    }
    .editor-container {
      width: 100%;
      max-width: 1000px;
    }
    .card {
      background-color: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 12px;
      padding: 1.5rem;
      box-shadow: 0 10px 25px -5px rgba(0,0,0,0.3);
    }
    .video-wrapper {
      position: relative;
      padding-bottom: 56.25%; /* 16:9 */
      height: 0;
      border-radius: 8px;
      overflow: hidden;
      background-color: #000;
      border: 1px solid var(--card-border);
      margin-bottom: 1.25rem;
    }
    .video-wrapper iframe {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      border: none;
    }
    .model-metrics {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 1rem;
      background-color: rgba(15, 23, 42, 0.6);
      border: 1px dashed var(--card-border);
      border-radius: 8px;
      padding: 0.75rem 1rem;
      margin-bottom: 1.25rem;
    }
    .metric-box { display: flex; flex-direction: column; }
    .metric-label { font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase; }
    .metric-value { font-size: 1.2rem; font-weight: 700; color: #38bdf8; }
    .form-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 1rem;
      margin-bottom: 1.25rem;
    }
    .form-group { display: flex; flex-direction: column; gap: 0.4rem; }
    label { font-size: 0.85rem; font-weight: 600; color: var(--text-muted); }
    input[type="number"], input[type="text"] {
      background-color: var(--input-bg);
      border: 1px solid var(--input-border);
      color: var(--text);
      border-radius: 6px;
      padding: 0.75rem;
      font-size: 1.1rem;
      outline: none;
      transition: border-color 0.2s;
    }
    input:focus { border-color: var(--primary); }
    .btn-submit {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      width: 100%;
      padding: 0.85rem;
      font-size: 1rem;
      font-weight: 600;
      border-radius: 6px;
      border: none;
      cursor: pointer;
      background-color: var(--primary);
      color: white;
      transition: background-color 0.2s;
    }
    .btn-submit:hover { background-color: var(--primary-hover); }
    .htmx-indicator { display: none; }
    .htmx-request .htmx-indicator { display: inline-block; }
    .alert {
      padding: 0.75rem 1rem;
      border-radius: 6px;
      margin-bottom: 1rem;
      font-size: 0.9rem;
    }
    .alert-success { background-color: rgba(16, 185, 129, 0.2); border: 1px solid var(--success); color: #34d399; }
    .alert-danger { background-color: rgba(239, 68, 68, 0.2); border: 1px solid var(--danger); color: #fca5a5; }
    .alert-warning { background-color: rgba(245, 158, 11, 0.2); border: 1px solid var(--warning); color: #fde68a; }
  </style>
</head>
<body>
  <header>
    <div style="display: flex; align-items: center; gap: 1rem;">
      <button id="toggle-sidebar-btn" class="btn-sm" onclick="toggleSidebar()" title="Toggle Sidebar (Full Screen)">
        <span>☰</span> Clips
      </button>
      <div class="brand">
        <span>🚗</span> Clippa
      </div>
    </div>

    <div class="header-actions">
      <button class="btn-sm btn-primary-sm" hx-post="/sync-drive" hx-target="body" hx-indicator="#sync-spinner">
        <span>📥</span> Sync Drive <span id="sync-spinner" class="htmx-indicator">⏳</span>
      </button>
      <button class="btn-sm" onclick="window.location.reload()">
        <span>🔄</span> Refresh
      </button>
    </div>
  </header>

  <div class="app-layout">
    <aside id="sidebar-panel" class="sidebar">
      {{template "sidebar.html" .}}
    </aside>

    <main class="main-content">
      <div class="editor-container">
        {{template "task_card.html" .}}
      </div>
    </main>
  </div>

  <script>
    function toggleSidebar() {
      const sidebar = document.getElementById('sidebar-panel');
      sidebar.classList.toggle('collapsed');
    }

    function filterClips(type) {
      document.querySelectorAll('.filter-btn').forEach(btn => btn.classList.remove('active'));
      const activeBtn = document.getElementById('filter-' + type);
      if (activeBtn) activeBtn.classList.add('active');

      const items = document.querySelectorAll('.clip-item');
      items.forEach(item => {
        const isDone = item.getAttribute('data-status') === 'done';
        if (type === 'all') {
          item.style.display = 'flex';
        } else if (type === 'done') {
          item.style.display = isDone ? 'flex' : 'none';
        } else if (type === 'pending') {
          item.style.display = !isDone ? 'flex' : 'none';
        }
      });
    }
  </script>
</body>
</html>
```

`templates/sidebar.html`:
```html
<div class="sidebar-header">
  <div class="stats-row">
    <span>Progress</span>
    <strong>{{.CompletedCount}} / {{.TotalClips}} Done ({{.ProgressPct}}%)</strong>
  </div>
  <div class="progress-bar-container">
    <div class="progress-bar" style="width: {{.ProgressPct}}%;"></div>
  </div>
  <div class="filter-group">
    <button id="filter-all" class="filter-btn active" onclick="filterClips('all')">All ({{.TotalClips}})</button>
    <button id="filter-pending" class="filter-btn" onclick="filterClips('pending')">Pending ({{.PendingCount}})</button>
    <button id="filter-done" class="filter-btn" onclick="filterClips('done')">Done ({{.CompletedCount}})</button>
  </div>
</div>

<div id="clip-list" class="clip-list">
  {{if eq (len .Clips) 0}}
    <div style="padding: 2rem 1rem; text-align: center; color: var(--text-muted); font-size: 0.9rem;">
      No clips found in sheet.<br><br>
      <button class="btn-sm btn-primary-sm" hx-post="/sync-drive" hx-target="body">
        📥 Sync 9 Clips from Drive
      </button>
    </div>
  {{else}}
    {{range .Clips}}
      {{template "clip_item.html" .}}
    {{end}}
  {{end}}
</div>
```

`templates/clip_item.html`:
```html
<div
  id="clip-item-{{.RowIndex}}"
  class="clip-item {{if .IsSelected}}selected{{end}}"
  data-status="{{if .IsComplete}}done{{else}}pending{{end}}"
  hx-get="/clip?row={{.RowIndex}}"
  hx-target="#task-card"
  hx-swap="outerHTML">
  <div class="clip-item-header">
    <span class="clip-name">📹 {{.Name}}</span>
    {{if .IsComplete}}
      <span class="badge badge-done">✓ Done</span>
    {{else}}
      <span class="badge badge-pending">⏳ Pending</span>
    {{end}}
  </div>
  <div class="clip-meta">
    <span>Run {{.Run}} (Row {{.RowIndex}})</span>
    {{if .IsComplete}}
      <span style="color: #34d399;">Act: {{.ActualIn}} / {{.ActualOut}}</span>
    {{else if or .ModelIn .ModelOut}}
      <span style="color: #38bdf8;">Pred: {{.ModelIn}} / {{.ModelOut}}</span>
    {{end}}
  </div>
</div>
```

`templates/task_card.html`:
```html
<div id="task-card" class="card">
  {{if .SelectedClip}}
    {{with .SelectedClip}}
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem;">
        <div>
          <h2 style="font-size: 1.25rem; font-weight: 700; display: flex; align-items: center; gap: 0.5rem;">
            📹 {{.Name}}
            <span class="badge">Run {{.Run}}</span>
          </h2>
          <div style="font-size: 0.8rem; color: var(--text-muted); margin-top: 0.25rem;">
            Google Sheet Row {{.RowIndex}}
          </div>
        </div>
        <div>
          {{if .IsComplete}}
            <span class="badge badge-done" style="font-size: 0.8rem; padding: 0.35rem 0.75rem;">✓ Completed</span>
          {{else}}
            <span class="badge badge-pending" style="font-size: 0.8rem; padding: 0.35rem 0.75rem;">⏳ Pending Review</span>
          {{end}}
        </div>
      </div>

      {{if $.SavedSuccess}}
        <div class="alert alert-success">
          ✓ <strong>Saved successfully!</strong> Row {{.RowIndex}} updated in Google Sheets.
        </div>
      {{end}}

      {{if $.ErrorMessage}}
        <div class="alert alert-danger">
          <strong>Error:</strong> {{$.ErrorMessage}}
        </div>
      {{end}}

      {{if .DriveFileID}}
        <div class="video-wrapper">
          <iframe
            src="https://drive.google.com/file/d/{{.DriveFileID}}/preview"
            allow="autoplay; fullscreen"
            allowfullscreen>
          </iframe>
        </div>
      {{else}}
        <div class="alert alert-warning">
          ⚠️ <strong>Video Preview Not Available:</strong> File <code>{{.Name}}</code> was not found in the Google Drive folder.
        </div>
      {{end}}

      <div class="model-metrics">
        <div class="metric-box">
          <span class="metric-label">Model Predicted In</span>
          <span class="metric-value">{{if .ModelIn}}{{.ModelIn}}{{else}}—{{end}}</span>
        </div>
        <div class="metric-box">
          <span class="metric-label">Model Predicted Out</span>
          <span class="metric-value">{{if .ModelOut}}{{.ModelOut}}{{else}}—{{end}}</span>
        </div>
      </div>

      <form hx-post="/submit" hx-target="#task-card" hx-swap="outerHTML">
        <input type="hidden" name="row_index" value="{{.RowIndex}}">
        <input type="hidden" name="name" value="{{.Name}}">
        <input type="hidden" name="run" value="{{.Run}}">

        <div class="form-grid">
          <div class="form-group">
            <label for="actual_in">Actual In (Ground Truth) *</label>
            <input
              type="number"
              id="actual_in"
              name="actual_in"
              value="{{.ActualIn}}"
              required
              min="0"
              autofocus
              placeholder="e.g. 10">
          </div>
          <div class="form-group">
            <label for="actual_out">Actual Out (Ground Truth) *</label>
            <input
              type="number"
              id="actual_out"
              name="actual_out"
              value="{{.ActualOut}}"
              required
              min="0"
              placeholder="e.g. 12">
          </div>
        </div>

        <button type="submit" class="btn-submit">
          <span>💾 Save Ground Truth</span>
          <span class="htmx-indicator">⏳</span>
        </button>
      </form>

      <!-- Out-of-band swap to update this item in the sidebar immediately -->
      <div id="clip-item-{{.RowIndex}}" hx-swap-oob="outerHTML">
        {{template "clip_item.html" .}}
      </div>
    {{end}}
  {{else}}
    <div style="text-align: center; padding: 4rem 1rem; color: var(--text-muted);">
      <div style="font-size: 3rem; margin-bottom: 1rem;">📹</div>
      <h3 style="font-size: 1.25rem; font-weight: 600; margin-bottom: 0.5rem; color: var(--text);">No Clip Selected</h3>
      <p>Select a clip from the sidebar on the left or sync clips from Google Drive.</p>
    </div>
  {{end}}
</div>
```

- [ ] **Step 3: Run template tests to verify**

Run: `go test -v -run TestTemplates ./...`  
Expected: PASS

---

### Task 4: Update HTTP Server Routes & Handlers

**Files:**
- Modify: `server.go`
- Modify: `main.go`
- Modify: `server_test.go`

**Interfaces:**
- Consumes: `TaskService` interface
- Produces: Updated `Server` supporting `GET /`, `GET /clip?row=N`, `POST /submit`, `POST /sync-drive`.

- [ ] **Step 1: Write server tests for routes**

```go
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
```

- [ ] **Step 2: Update `server.go` and `main.go`**

`server.go`:
```go
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
	FetchPageData(ctx context.Context, selectedRow int) (*PageData, error)
	UpdateActualCounts(ctx context.Context, rowIndex int, actualIn, actualOut string) error
	SyncDriveClips(ctx context.Context) (int, error)
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
	mux.HandleFunc("GET /clip", s.handleClip)
	mux.HandleFunc("POST /submit", s.handleSubmit)
	mux.HandleFunc("POST /sync-drive", s.handleSyncDrive)
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	pageData, err := s.svc.FetchPageData(r.Context(), 0)
	if err != nil {
		log.Printf("[Error] FetchPageData failed: %v", err)
		pageData = &PageData{ErrorMessage: err.Error()}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", pageData); err != nil {
		log.Printf("[Error] Failed to render index template: %v", err)
	}
}

func (s *Server) handleClip(w http.ResponseWriter, r *http.Request) {
	rowStr := r.URL.Query().Get("row")
	row, _ := strconv.Atoi(rowStr)

	pageData, err := s.svc.FetchPageData(r.Context(), row)
	if err != nil {
		log.Printf("[Error] FetchPageData for row %d failed: %v", row, err)
		pageData = &PageData{ErrorMessage: err.Error()}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "task_card.html", pageData); err != nil {
		log.Printf("[Error] Failed to render task_card template: %v", err)
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
		pageData, _ := s.svc.FetchPageData(r.Context(), rowIndex)
		if pageData == nil {
			pageData = &PageData{}
		}
		pageData.ErrorMessage = "Failed to update Google Sheet: " + err.Error()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = s.tmpl.ExecuteTemplate(w, "task_card.html", pageData)
		return
	}

	pageData, err := s.svc.FetchPageData(r.Context(), rowIndex)
	if err != nil {
		log.Printf("[Error] FetchPageData after submit failed: %v", err)
		pageData = &PageData{
			ErrorMessage: "Saved to sheet, but failed to refresh data: " + err.Error(),
		}
	} else {
		pageData.SavedSuccess = true
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "task_card.html", pageData); err != nil {
		log.Printf("[Error] Failed to render task_card template: %v", err)
	}
}

func (s *Server) handleSyncDrive(w http.ResponseWriter, r *http.Request) {
	count, err := s.svc.SyncDriveClips(r.Context())
	if err != nil {
		log.Printf("[Error] SyncDriveClips failed: %v", err)
	} else {
		log.Printf("[Sync] Appended %d clips from Drive to Sheet", count)
	}

	pageData, err := s.svc.FetchPageData(r.Context(), 0)
	if err != nil {
		log.Printf("[Error] FetchPageData after sync failed: %v", err)
		pageData = &PageData{ErrorMessage: err.Error()}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", pageData); err != nil {
		log.Printf("[Error] Failed to render index template: %v", err)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
```

`main.go`:
```go
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Fatal: failed to load configuration: %v", err)
	}

	ctx := context.Background()
	client, err := NewGoogleWorkspaceClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize Google Workspace client: %v", err)
	}

	tmpl, err := template.ParseFiles(
		"templates/index.html",
		"templates/sidebar.html",
		"templates/clip_item.html",
		"templates/task_card.html",
	)
	if err != nil {
		log.Fatalf("Fatal: failed to parse templates: %v", err)
	}

	mux := NewServer(client, tmpl)

	addr := ":" + cfg.Port
	log.Printf("🚗 Clippa Master-Detail Review Server running at http://localhost%s", addr)
	log.Printf("Spreadsheet ID: %s (Sheet: %s)", cfg.SpreadsheetID, cfg.SheetName)
	log.Printf("Drive Folder ID: %s", cfg.DriveFolderID)

	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server stopped unexpectedly: %v", err)
	}
}
```

- [ ] **Step 3: Run all tests**

Run: `go test -v ./...`  
Expected: PASS

---

### Task 5: End-to-End Verification & Build Check

**Files:**
- Verify: `go mod tidy`
- Verify: `go test -v ./...`
- Verify: `go build -o /dev/null .`
- Update: `README.md`

- [ ] **Step 1: Run full verification commands**

```bash
go mod tidy
go test -v ./...
go build -o /dev/null .
```
Expected: All tests pass and build succeeds cleanly.
