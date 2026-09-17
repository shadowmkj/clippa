# Traffic Footage Review Tool (Clippa) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a zero-build Go + HTMX web application for reviewing video traffic footage in Google Drive and recording ground-truth counts into a Google Sheet.

**Architecture:** A lightweight Go HTTP server with standard library templates, an in-memory Google Drive File ID resolver, and a Google Sheets API v4 integration for row scanning and targeted cell updates. The frontend uses HTMX 2.0.4 and embedded CSS for seamless `#task-card` swapping without full page reloads.

**Tech Stack:** Go 1.22+, `google.golang.org/api/sheets/v4`, `google.golang.org/api/drive/v3`, HTMX 2.0.4 (CDN), Vanilla HTML5/CSS.

**Spec:** `docs/superpowers/specs/2026-09-17-traffic-footage-review-tool-design.md`

## Global Constraints
- Strict zero-build: No npm, yarn, Vite, Webpack, or Tailwind.
- No CGO: Standard pure Go build.
- Sheet column schema: Col A = `name`, Col B = `run`, Col C = `in`, Col D = `out`, Col E = `actual_in`, Col F = `actual_out`.
- 1-based row indexing: Data rows start at Sheet Row 2 (Header is Row 1).
- No placeholders: All code blocks in tasks and implementation must be complete and compilable.

---

### Task 1: Configuration Management & Go Module Setup

**Files:**
- Create: `go.mod`
- Create: `config.go`
- Test: `config_test.go`
- Create: `config.example.json`

**Interfaces:**
- Produces: `type Config struct`, `func LoadConfig(path string) (*Config, error)`

- [ ] **Step 1: Write the failing test for configuration loading**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_DefaultsAndFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	jsonContent := `{
		"spreadsheet_id": "test-sheet-id",
		"sheet_name": "TrafficData",
		"drive_folder_id": "test-folder-id",
		"credentials_file": "test-creds.json",
		"port": "9090"
	}`

	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.SpreadsheetID != "test-sheet-id" {
		t.Errorf("expected spreadsheet_id 'test-sheet-id', got '%s'", cfg.SpreadsheetID)
	}
	if cfg.SheetName != "TrafficData" {
		t.Errorf("expected sheet_name 'TrafficData', got '%s'", cfg.SheetName)
	}
	if cfg.DriveFolderID != "test-folder-id" {
		t.Errorf("expected drive_folder_id 'test-folder-id', got '%s'", cfg.DriveFolderID)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port '9090', got '%s'", cfg.Port)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"spreadsheet_id": "file-id"}`), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	t.Setenv("CLIPPA_SPREADSHEET_ID", "env-sheet-id")
	t.Setenv("PORT", "7070")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.SpreadsheetID != "env-sheet-id" {
		t.Errorf("expected env override 'env-sheet-id', got '%s'", cfg.SpreadsheetID)
	}
	if cfg.Port != "7070" {
		t.Errorf("expected port override '7070', got '%s'", cfg.Port)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./...`  
Expected: FAIL (undefined: `LoadConfig`)

- [ ] **Step 3: Implement `go.mod`, `config.go`, and `config.example.json`**

`go.mod`:
```go
module clippa

go 1.22.0

require (
	google.golang.org/api v0.170.0
)
```

`config.go`:
```go
package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	SpreadsheetID   string `json:"spreadsheet_id"`
	SheetName       string `json:"sheet_name"`
	DriveFolderID   string `json:"drive_folder_id"`
	CredentialsFile string `json:"credentials_file"`
	Port            string `json:"port"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		SheetName:       "Sheet1",
		CredentialsFile: "credentials.json",
		Port:            "8080",
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	if envVal := os.Getenv("CLIPPA_SPREADSHEET_ID"); envVal != "" {
		cfg.SpreadsheetID = envVal
	}
	if envVal := os.Getenv("CLIPPA_SHEET_NAME"); envVal != "" {
		cfg.SheetName = envVal
	}
	if envVal := os.Getenv("CLIPPA_DRIVE_FOLDER_ID"); envVal != "" {
		cfg.DriveFolderID = envVal
	}
	if envVal := os.Getenv("CLIPPA_CREDENTIALS_FILE"); envVal != "" {
		cfg.CredentialsFile = envVal
	}
	if envVal := os.Getenv("PORT"); envVal != "" {
		cfg.Port = envVal
	}

	return cfg, nil
}
```

`config.example.json`:
```json
{
  "spreadsheet_id": "YOUR_GOOGLE_SPREADSHEET_ID",
  "sheet_name": "Sheet1",
  "drive_folder_id": "YOUR_GOOGLE_DRIVE_FOLDER_ID",
  "credentials_file": "credentials.json",
  "port": "8080"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./...`  
Expected: PASS

- [ ] **Step 5: Commit changes**

```bash
git add go.mod config.go config_test.go config.example.json
git commit -m "feat: add configuration loader and module setup"
```

---

### Task 2: Data Models & Sheet Queue Parser

**Files:**
- Create: `models.go`
- Create: `parser.go`
- Test: `parser_test.go`

**Interfaces:**
- Consumes: None
- Produces:
  - `type ClipTask struct`
  - `type SheetRow struct`
  - `func ParseSheetRows(rawRows [][]interface{}) []SheetRow`
  - `func BuildTaskQueue(rows []SheetRow, driveResolver func(name string) string) *ClipTask`

- [ ] **Step 1: Write the failing tests for sheet row parsing and queue determination**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./...`  
Expected: FAIL (undefined: `ParseSheetRows`, `BuildTaskQueue`)

- [ ] **Step 3: Implement `models.go` and `parser.go`**

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./...`  
Expected: PASS

- [ ] **Step 5: Commit changes**

```bash
git add models.go parser.go parser_test.go
git commit -m "feat: add sheet data models and queue parser"
```

---

### Task 3: Google Drive Cache & Sheets API Service Wrapper

**Files:**
- Create: `drive_cache.go`
- Create: `sheets.go`
- Test: `drive_cache_test.go`

**Interfaces:**
- Consumes: `Config`, `models.go`, `parser.go`
- Produces:
  - `type DriveCache struct`, `func NewDriveCache(srv *drive.Service, folderID string) *DriveCache`
  - `type GoogleWorkspaceClient struct`, `func NewGoogleWorkspaceClient(ctx context.Context, cfg *Config) (*GoogleWorkspaceClient, error)`
  - `func (c *GoogleWorkspaceClient) FetchNextTask(ctx context.Context) (*ClipTask, error)`
  - `func (c *GoogleWorkspaceClient) UpdateActualCounts(ctx context.Context, rowIndex int, actualIn, actualOut string) error`

- [ ] **Step 1: Write test for DriveCache in-memory operations**

```go
package main

import (
	"testing"
)

func TestDriveCache_ManualPopulateAndGet(t *testing.T) {
	cache := &DriveCache{
		folderID: "test-folder",
		fileMap:  make(map[string]string),
	}

	cache.Set("clip_1.mp4", "drive_id_1")
	cache.Set("clip_2.mp4", "drive_id_2")

	if id := cache.Get("clip_1.mp4"); id != "drive_id_1" {
		t.Errorf("expected drive_id_1, got '%s'", id)
	}
	if id := cache.Get("clip_2.mp4"); id != "drive_id_2" {
		t.Errorf("expected drive_id_2, got '%s'", id)
	}
	if id := cache.Get("nonexistent.mp4"); id != "" {
		t.Errorf("expected empty string for missing file, got '%s'", id)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./...`  
Expected: FAIL (undefined: `DriveCache`)

- [ ] **Step 3: Implement `drive_cache.go` and `sheets.go`**

`drive_cache.go`:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"google.golang.org/api/drive/v3"
)

type DriveCache struct {
	srv      *drive.Service
	folderID string
	mu       sync.RWMutex
	fileMap  map[string]string
}

func NewDriveCache(srv *drive.Service, folderID string) *DriveCache {
	return &DriveCache{
		srv:      srv,
		folderID: folderID,
		fileMap:  make(map[string]string),
	}
}

func (d *DriveCache) Set(name, fileID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fileMap[name] = fileID
}

func (d *DriveCache) Get(name string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.fileMap[name]
}

func (d *DriveCache) Refresh(ctx context.Context) error {
	if d.srv == nil || d.folderID == "" {
		return nil
	}

	query := fmt.Sprintf("'%s' in parents and trashed = false", d.folderID)
	var pageToken string
	newMap := make(map[string]string)

	for {
		call := d.srv.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name)").
			PageSize(100).
			Context(ctx)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		res, err := call.Do()
		if err != nil {
			return fmt.Errorf("failed to list drive files: %w", err)
		}

		for _, file := range res.Files {
			newMap[file.Name] = file.Id
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	d.mu.Lock()
	d.fileMap = newMap
	d.mu.Unlock()

	log.Printf("[Drive] Cached %d files from folder %s", len(newMap), d.folderID)
	return nil
}

func (d *DriveCache) Resolve(name string) string {
	id := d.Get(name)
	if id != "" {
		return id
	}

	if d.srv == nil || d.folderID == "" {
		return ""
	}

	// Fallback single query on cache miss
	query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", name, d.folderID)
	res, err := d.srv.Files.List().Q(query).Fields("files(id, name)").PageSize(1).Do()
	if err == nil && len(res.Files) > 0 {
		d.Set(name, res.Files[0].Id)
		return res.Files[0].Id
	}

	return ""
}
```

`sheets.go`:
```go
package main

import (
	"context"
	"fmt"
	"log"

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

func (g *GoogleWorkspaceClient) FetchNextTask(ctx context.Context) (*ClipTask, error) {
	readRange := fmt.Sprintf("%s!A:F", g.cfg.SheetName)
	resp, err := g.sheetsSrv.Spreadsheets.Values.Get(g.cfg.SpreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet range %s: %w", readRange, err)
	}

	rows := ParseSheetRows(resp.Values)
	task := BuildTaskQueue(rows, g.driveCache.Resolve)
	return task, nil
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./...`  
Expected: PASS

- [ ] **Step 5: Commit changes**

```bash
git add drive_cache.go drive_cache_test.go sheets.go
git commit -m "feat: add Google Drive cache and Google Sheets API wrapper"
```

---

### Task 4: UI HTML Templates & Embedded CSS

**Files:**
- Create: `templates/index.html`
- Create: `templates/task_card.html`
- Test: `templates_test.go`

**Interfaces:**
- Consumes: `ClipTask` model
- Produces: Parsable `html/template` templates with HTMX integration

- [ ] **Step 1: Write template rendering unit tests**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./...`  
Expected: FAIL (missing template files)

- [ ] **Step 3: Implement `templates/index.html` and `templates/task_card.html`**

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
      min-height: 100vh;
      display: flex;
      flex-direction: column;
    }
    header {
      background-color: var(--card-bg);
      border-bottom: 1px solid var(--card-border);
      padding: 1rem 2rem;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      font-size: 1.25rem;
      font-weight: 700;
      color: var(--text);
    }
    .brand span { color: var(--primary); }
    .container {
      max-width: 960px;
      margin: 2rem auto;
      padding: 0 1rem;
      width: 100%;
      flex: 1;
    }
    .card {
      background-color: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 12px;
      padding: 1.5rem;
      box-shadow: 0 10px 25px -5px rgba(0,0,0,0.3);
    }
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1.25rem;
    }
    .clip-title {
      font-size: 1.15rem;
      font-weight: 600;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
    .badge {
      display: inline-block;
      padding: 0.25rem 0.6rem;
      font-size: 0.75rem;
      font-weight: 600;
      border-radius: 9999px;
      background-color: var(--card-border);
      color: var(--text-muted);
    }
    .badge-primary { background-color: rgba(59, 130, 246, 0.2); color: #60a5fa; }
    .badge-success { background-color: rgba(16, 185, 129, 0.2); color: #34d399; }
    .progress-bar-container {
      width: 100%;
      height: 6px;
      background-color: var(--input-bg);
      border-radius: 9999px;
      margin-bottom: 1.5rem;
      overflow: hidden;
    }
    .progress-bar {
      height: 100%;
      background-color: var(--primary);
      transition: width 0.3s ease;
    }
    .video-wrapper {
      position: relative;
      padding-bottom: 56.25%; /* 16:9 */
      height: 0;
      border-radius: 8px;
      overflow: hidden;
      background-color: #000;
      border: 1px solid var(--card-border);
      margin-bottom: 1.5rem;
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
      margin-bottom: 1.5rem;
    }
    .metric-box {
      display: flex;
      flex-direction: column;
    }
    .metric-label { font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase; }
    .metric-value { font-size: 1.25rem; font-weight: 700; color: #38bdf8; }
    .form-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 1rem;
      margin-bottom: 1.5rem;
    }
    .form-group {
      display: flex;
      flex-direction: column;
      gap: 0.4rem;
    }
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
    .btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      width: 100%;
      padding: 0.85rem 1.5rem;
      font-size: 1rem;
      font-weight: 600;
      border-radius: 6px;
      border: none;
      cursor: pointer;
      background-color: var(--primary);
      color: white;
      transition: background-color 0.2s;
    }
    .btn:hover { background-color: var(--primary-hover); }
    .htmx-indicator { display: none; }
    .htmx-request .htmx-indicator { display: inline-block; }
    .alert {
      padding: 0.75rem 1rem;
      border-radius: 6px;
      margin-bottom: 1rem;
      font-size: 0.9rem;
    }
    .alert-danger { background-color: rgba(239, 68, 68, 0.2); border: 1px solid var(--danger); color: #fca5a5; }
    .alert-warning { background-color: rgba(245, 158, 11, 0.2); border: 1px solid var(--warning); color: #fde68a; }
    .completed-state {
      text-align: center;
      padding: 3rem 1rem;
    }
    .completed-icon { font-size: 3rem; margin-bottom: 1rem; }
    .completed-title { font-size: 1.75rem; font-weight: 700; margin-bottom: 0.5rem; }
  </style>
</head>
<body>
  <header>
    <div class="brand">
      <span>🚗</span> Clippa
    </div>
    <div>
      <span class="badge badge-primary">Google Sheets & Drive Connected</span>
    </div>
  </header>

  <main class="container">
    {{template "task_card.html" .}}
  </main>
</body>
</html>
```

`templates/task_card.html`:
```html
<div id="task-card" class="card">
  {{if .IsCompleted}}
    <div class="completed-state">
      <div class="completed-icon">🎉</div>
      <h2 class="completed-title">All Clips Completed!</h2>
      <p style="color: var(--text-muted); margin-bottom: 1.5rem;">
        All {{.TotalClips}} traffic video clips have been reviewed and verified in Google Sheets.
      </p>
      <div class="badge badge-success" style="font-size: 0.9rem; padding: 0.5rem 1rem;">
        ✓ {{.CompletedCount}} of {{.TotalClips}} logged (100%)
      </div>
    </div>
  {{else}}
    <div class="card-header">
      <div>
        <div class="clip-title">
          📹 {{.Name}}
          <span class="badge">Run {{.Run}}</span>
        </div>
        <div style="font-size: 0.8rem; color: var(--text-muted); margin-top: 0.25rem;">
          Row {{.RowIndex}} in Google Sheet
        </div>
      </div>
      <div>
        <span class="badge badge-primary">Clip {{.QueuePosition}} of {{.TotalClips}} ({{.ProgressPct}}%)</span>
      </div>
    </div>

    <div class="progress-bar-container">
      <div class="progress-bar" style="width: {{.ProgressPct}}%;"></div>
    </div>

    {{if .ErrorMessage}}
      <div class="alert alert-danger">
        <strong>Error:</strong> {{.ErrorMessage}}
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
        ⚠️ <strong>Video Preview Not Available:</strong> File <code>{{.Name}}</code> was not found in the designated Google Drive folder.
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

      <button type="submit" class="btn">
        <span>Save & Next Clip ➔</span>
        <span class="htmx-indicator">⏳</span>
      </button>
    </form>
  {{end}}
</div>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./...`  
Expected: PASS

- [ ] **Step 5: Commit changes**

```bash
git add templates/index.html templates/task_card.html templates_test.go
git commit -m "feat: add HTML templates with HTMX integration and modern styling"
```

---

### Task 5: HTTP Server Routing & HTMX Handlers

**Files:**
- Create: `server.go`
- Create: `main.go`
- Test: `server_test.go`

**Interfaces:**
- Consumes: `Config`, `GoogleWorkspaceClient`, `models.go`, `templates`
- Produces:
  - `type TaskService interface { FetchNextTask(ctx) (*ClipTask, error); UpdateActualCounts(ctx, row, in, out) error }`
  - `func NewServer(taskService TaskService, tmpl *template.Template) *http.ServeMux`
  - `main()`

- [ ] **Step 1: Write HTTP handler tests using a mock TaskService**

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./...`  
Expected: FAIL (undefined: `NewServer`)

- [ ] **Step 3: Implement `server.go` and `main.go`**

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
	FetchNextTask(ctx context.Context) (*ClipTask, error)
	UpdateActualCounts(ctx context.Context, rowIndex int, actualIn, actualOut string) error
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
	mux.HandleFunc("POST /submit", s.handleSubmit)
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	task, err := s.svc.FetchNextTask(r.Context())
	if err != nil {
		log.Printf("[Error] FetchNextTask failed: %v", err)
		task = &ClipTask{
			ErrorMessage: err.Error(),
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", task); err != nil {
		log.Printf("[Error] Failed to render index template: %v", err)
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
		task, _ := s.svc.FetchNextTask(r.Context())
		if task == nil {
			task = &ClipTask{RowIndex: rowIndex}
		}
		task.ErrorMessage = "Failed to update Google Sheet: " + err.Error()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = s.tmpl.ExecuteTemplate(w, "task_card.html", task)
		return
	}

	nextTask, err := s.svc.FetchNextTask(r.Context())
	if err != nil {
		log.Printf("[Error] FetchNextTask after submit failed: %v", err)
		nextTask = &ClipTask{
			ErrorMessage: "Updated sheet, but failed to fetch next task: " + err.Error(),
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "task_card.html", nextTask); err != nil {
		log.Printf("[Error] Failed to render task_card template: %v", err)
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

	tmpl, err := template.ParseFiles("templates/index.html", "templates/task_card.html")
	if err != nil {
		log.Fatalf("Fatal: failed to parse templates: %v", err)
	}

	mux := NewServer(client, tmpl)

	addr := ":" + cfg.Port
	log.Printf("🚗 Clippa server listening at http://localhost%s", addr)
	log.Printf("Spreadsheet ID: %s (Sheet: %s)", cfg.SpreadsheetID, cfg.SheetName)
	log.Printf("Drive Folder ID: %s", cfg.DriveFolderID)

	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server stopped unexpectedly: %v", err)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./...`  
Expected: PASS

- [ ] **Step 5: Commit changes**

```bash
git add server.go main.go server_test.go
git commit -m "feat: implement HTTP routing and HTMX submission handlers"
```

---

### Task 6: Documentation, Dependency Cleanup & Build Verification

**Files:**
- Create: `README.md`
- Verify: `go.mod`, `go.sum`

- [ ] **Step 1: Run `go mod tidy` and test suite**

```bash
go mod tidy
go test -v ./...
```

- [ ] **Step 2: Verify binary compilation**

```bash
go build -o /dev/null .
```

- [ ] **Step 3: Create `README.md` with setup instructions**

`README.md`:
```markdown
# Clippa — Traffic Footage Review & Ground Truth Logging

A lightweight, zero-build Go web application for reviewing video traffic footage from Google Drive and recording ground truth counts (`actual_in`, `actual_out`) directly into Google Sheets.

## Features
- **Zero Build Tooling**: Vanilla HTML5/CSS and HTMX (loaded via CDN).
- **Embedded Drive Preview**: Automatically resolves video filenames from Google Drive and embeds responsive 16:9 previews.
- **Atomic Google Sheets Sync**: Updates columns E (`actual_in`) and F (`actual_out`) using `USER_ENTERED` formatting.
- **Fast HTMX Card Swap**: Seamless partial DOM swap (`#task-card`) on form submission without full page refreshes.
- **Keyboard-Optimized**: Numeric inputs auto-focus upon transitioning to each new clip.

## Setup & Configuration

1. **Google Cloud Service Account**:
   - Place your service account JSON file as `credentials.json` in the project root.
   - Share your Google Sheet and Google Drive Folder with the service account client email (`...-compute@developer.gserviceaccount.com` or `@...iam.gserviceaccount.com`) as **Editor / Viewer**.

2. **Configuration (`config.json`)**:
   Create a `config.json` file (or copy `config.example.json`):
   ```json
   {
     "spreadsheet_id": "YOUR_SPREADSHEET_ID",
     "sheet_name": "Sheet1",
     "drive_folder_id": "YOUR_GOOGLE_DRIVE_FOLDER_ID",
     "credentials_file": "credentials.json",
     "port": "8080"
   }
   ```

3. **Run**:
   ```bash
   go run .
   ```
   Visit `http://localhost:8080`.
```

- [ ] **Step 4: Commit changes**

```bash
git add README.md go.mod go.sum
git commit -m "docs: add README with setup and execution instructions"
```
