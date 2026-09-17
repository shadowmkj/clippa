# Design Specification: Traffic Footage Review & Ground Truth Logging Web Interface

**Date:** 2026-09-17  
**Project:** Clippa (Traffic Footage Review Tool)  
**Status:** Approved  

---

## 1. Overview & Objective

Clippa is a lightweight, zero-build Go web application designed for reviewing video traffic footage stored in Google Drive and logging verified ground-truth counts (`actual_in`, `actual_out`) into a Google Sheet.

The application uses Go's standard library (`net/http`, `html/template`), vanilla embedded CSS, and HTMX to provide single-page application feel with fast partial DOM updates, zero npm/build dependencies, and robust integration with Google Sheets API v4 and Google Drive API v3.

---

## 2. Architecture & Data Flow

```
┌────────────────────────────────────────────────────────┐
│                   Browser Client                       │
│    - Vanilla HTML/CSS (Embedded Dark Mode)             │
│    - HTMX 2.0.4 (CDN)                                  │
│    - Embedded Google Drive Iframe Video Player         │
└───────────────────────────┬────────────────────────────┘
                            │
            HTTP GET /      │ HTTP POST /submit
            (Initial Page)  │ (HTMX outerHTML swap #task-card)
                            ▼
┌────────────────────────────────────────────────────────┐
│                   Go Web Server                        │
│    - net/http Router & Request Handlers                │
│    - html/template Engine                              │
│    - Google Workspace Client Wrapper                   │
│    - In-Memory Drive ID Cache                          │
└───────────────┬────────────────────────┬───────────────┘
                │                        │
  Drive API v3  │                        │ Sheets API v4
  (Read-only)   │                        │ (Read / Write)
                ▼                        ▼
┌────────────────────────┐      ┌────────────────────────┐
│  Google Drive Folder   │      │  Google Spreadsheet   │
│  (Traffic Video Files) │      │  (Data Rows A:F)       │
└────────────────────────┘      └────────────────────────┘
```

---

## 3. Data Specification & Google Sheet Schema

### Sheet Column Mapping
The target Google Spreadsheet contains 6 columns:
- **Col A (1)**: `name` — Filename of the video (e.g., `clip_1.mp4`)
- **Col B (2)**: `run` — Run / experiment identifier (e.g., `1`, `test_run_a`)
- **Col C (3)**: `in` — Model predicted inbound vehicle count
- **Col D (4)**: `out` — Model predicted outbound vehicle count
- **Col E (5)**: `actual_in` — Ground truth inbound count (to be annotated)
- **Col F (6)**: `actual_out` — Ground truth outbound count (to be annotated)

### Row Indexing Rules
- **Header Row**: Row 1 contains headers (`name`, `run`, `in`, `out`, `actual_in`, `actual_out`).
- **Data Rows**: Begin at Row 2. For any 0-indexed slice index `i` of data rows, the 1-based Google Sheets row number is `i + 2`.
- **Pending/Incomplete Row Condition**: A row is incomplete if `actual_in == ""` OR `actual_out == ""` (after trimming whitespace).
- **Targeted Update Range**: `fmt.Sprintf("%s!E%d:F%d", sheetName, rowIndex, rowIndex)` using `ValueInputOption: "USER_ENTERED"`.

---

## 4. Components & Implementation Modules

### 4.1 Configuration (`config.go`)
Configuration is loaded from `config.json` with fallback/overrides from environment variables:
- `SpreadsheetID` / `CLIPPA_SPREADSHEET_ID`: Target Google Sheet ID.
- `SheetName` / `CLIPPA_SHEET_NAME`: Sheet tab name (default: `Sheet1`).
- `DriveFolderID` / `CLIPPA_DRIVE_FOLDER_ID`: Google Drive folder containing video files.
- `CredentialsFile` / `CLIPPA_CREDENTIALS_FILE`: Service account JSON path (default: `credentials.json`).
- `Port` / `PORT`: HTTP server port (default: `8080`).

### 4.2 Google Workspace Client Wrapper (`sheets.go`)
- **Initialization**: Single service account client configured with OAuth scopes:
  - `https://www.googleapis.com/auth/spreadsheets`
  - `https://www.googleapis.com/auth/drive.readonly`
- **Drive Cache (`DriveCache`)**:
  - Pre-fetches `name -> fileId` mapping from `DriveFolderID` using `Files.List(q: "'{folderId}' in parents and trashed = false")`.
  - Thread-safe lookup with mutex.
  - On cache-miss, issues an on-demand single file query and caches the result.
- **`FetchTaskQueue(ctx)`**:
  - Calls `Spreadsheets.Values.Get(spreadsheetId, "{sheetName}!A:F")`.
  - Normalizes row lengths (handling missing trailing empty cells).
  - Finds the first incomplete row, counts total data rows and completed rows, and builds `ClipTask`.
- **`UpdateActuals(ctx, rowIndex, actualIn, actualOut)`**:
  - Updates columns E and F for `rowIndex` with `ValueInputOption="USER_ENTERED"`.

### 4.3 Data Structures
```go
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

### 4.4 HTTP Handlers (`main.go`)
- `GET /`: Renders `templates/index.html` (which includes `task_card.html`).
- `POST /submit`:
  - Parses form fields: `row_index`, `actual_in`, `actual_out`, `name`, `run`.
  - Validates numeric inputs.
  - Updates Google Sheets.
  - Fetches the next incomplete task.
  - Renders and returns **only** `task_card.html` (with HTTP 200).
- `GET /healthz`: Health check endpoint.

### 4.5 Frontend Templates
- `templates/index.html`:
  - Standalone HTML5 document with embedded modern CSS (dark theme, glassmorphic card styling, responsive grid, clear typography).
  - Imports HTMX from `https://unpkg.com/htmx.org@2.0.4`.
  - Header with app title, sheet link, and connection status.
  - Main container `#task-card` containing initial `task_card.html`.
- `templates/task_card.html`:
  - Dynamic HTMX partial with `id="task-card"`.
  - Embedded iframe video preview: `https://drive.google.com/file/d/{DRIVE_FILE_ID}/preview`.
  - Model prediction comparison badge.
  - Ground truth entry form (`hx-post="/submit"`, `hx-target="#task-card"`, `hx-swap="outerHTML"`).
  - Loading spinner indicator on submit button.
  - Completed state card when all clips are reviewed.

---

## 5. Error Handling & Edge Cases

1. **Missing Google Drive Video**: If a file in the sheet is not found in the Drive folder, a clear notification banner is shown with a manual Drive File ID override input or skip option so labeling is not blocked.
2. **Rate Limit / Network Hiccups**: Google API errors are caught, logged cleanly with timestamps, and displayed within the card partial without crashing the server or losing entered form values.
3. **Sparse Sheets / Short Rows**: When reading rows where trailing columns are empty, the reader safely pads the row slice to at least 6 elements.
4. **All Rows Completed**: The app gracefully renders the all-completed celebration screen and disables the input form.

---

## 6. Verification & Testing Plan

1. **Compilation & Dependency Check**:
   - `go mod tidy`
   - `go build -o /dev/null .`
2. **Unit / Integration Logic Tests**:
   - `sheets_test.go`: Test row indexing conversion (0-index to Sheet 1-index), incomplete row detection, row padding, and progress percentage computation.
   - `main_test.go`: Test HTTP route handling and HTMX template partial rendering with dummy/mock data.
3. **Manual Validation Checklist**:
   - Verify HTMX outerHTML swap on `#task-card`.
   - Verify correct columns E and F are updated in the sheet.
