# Design Specification: Master-Detail Multi-User Traffic Footage Review UI

**Date:** 2026-09-17  
**Project:** Clippa  
**Status:** Approved  

---

## 1. Overview & Objective

Upgrade Clippa from a single-card sequential queue to a **Master-Detail Multi-User Review Interface**.

Key user outcomes:
1. **Clip List & Status Tracking**: A collapsable sidebar displaying all clips from Google Sheets with status indicators (`✓ Done` vs `⏳ Pending`), model predicted counts, and actual logged counts.
2. **Non-Sequential Random Access**: Multiple annotators can click and review any clip independently without being locked into a strict sequential order.
3. **One-Click Google Drive Sync**: A `"📥 Sync from Drive"` button that scans the Google Drive folder, finds missing video files not yet in the Google Sheet, and appends them automatically.
4. **Full-Screen Video Mode**: A sidebar collapse/expand toggle button allowing users to hide the sidebar for maximum video playback width.
5. **Real-Time Partial Updates**: Form submission updates Google Sheets columns E & F, marks the clip as `Done`, shows a `"✓ Saved successfully"` confirmation on the card, and updates the sidebar badge via HTMX Out-of-Band (`hx-swap-oob`) swap.

---

## 2. System Architecture & Endpoints

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Master-Detail Browser UI                        │
│   [☰ Toggle Sidebar]   [✓ 5/9 Done]   [📥 Sync Drive]   [🔄 Refresh]   │
├───────────────────────────────┬────────────────────────────────────────┤
│ SIDEBAR (#clip-sidebar)       │ DETAIL EDITOR (#task-card)             │
│ - Filter: All / Pending / Done│ - Responsive 16:9 Drive Video Preview │
│ - Search / Clip Items         │ - Model Predicted In / Out             │
│ - Click -> hx-get="/clip?row" │ - Ground Truth Form (actual_in / out)  │
│ - OOB Target on Submit        │ - "Save Ground Truth" Button           │
└───────────────────────────────┴────────────────────────────────────────┘
```

### HTTP Routes
- `GET /`: Full page layout (`templates/index.html`), loads all clips into the sidebar and automatically selects the first pending clip (or row 2 if all complete).
- `GET /clip?row={N}`: Returns **only** the `#task-card` HTML partial for row `N`.
- `POST /submit`:
  - Receives `row_index`, `actual_in`, `actual_out`.
  - Updates Google Sheets range `Sheet!E{row}:F{row}` (`USER_ENTERED`).
  - Returns the updated `#task-card` partial with `SavedSuccess: true`.
  - Includes an Out-of-Band swap `<div id="clip-item-{row}" hx-swap-oob="outerHTML">...</div>` updating that item's status in the sidebar.
- `POST /sync-drive`:
  - Refreshes the Google Drive file cache (`DriveCache.Refresh`).
  - Reads existing Sheet rows (`A:F`).
  - Identifies video files in the Drive folder that do not exist in the Sheet.
  - Appends new rows to the Google Sheet (Col A = name, Col B = "1", Col C-F = "").
  - Returns the updated full page or sidebar partial.

---

## 3. Data Models

```go
type ClipItem struct {
    RowIndex    int    // 1-based sheet row index (Row 2, 3...)
    Name        string // Col A
    Run         string // Col B
    ModelIn     string // Col C
    ModelOut    string // Col D
    ActualIn    string // Col E
    ActualOut   string // Col F
    DriveFileID string // Google Drive File ID
    IsComplete  bool   // true if actual_in != "" && actual_out != ""
    IsSelected  bool   // true if currently active in editor
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
    ActiveFilter   string // "all", "pending", "done"
}
```

---

## 4. UI & Layout Specifications

1. **Collapsible 2-Column CSS Grid**:
   - Split layout with `.sidebar` (width: 360px, scrollable) and `.main-content` (flex: 1).
   - `.sidebar.collapsed`: `width: 0; padding: 0; overflow: hidden; opacity: 0;`
   - Smooth CSS transitions (`transition: all 0.25s ease-in-out`).
   - Toggle button `[ ☰ Clips ]` in the top header.

2. **Sidebar Features**:
   - Stats summary: Total, Completed (`Done`), and `Pending`.
   - Filter tabs (`All`, `Pending`, `Done`) powered by CSS/JS dataset attributes (`data-status="done"`).
   - Sync from Drive button with HTMX indicator.
   - Clip items displaying:
     - Name and Run badge.
     - Status badge: `✓ Done` (green) or `⏳ Pending` (amber).
     - Automated model predictions and logged actual counts.
     - Active selection border / highlight.

3. **Detail Editor Card**:
   - Responsive 16:9 Google Drive embedded player (`https://drive.google.com/file/d/{ID}/preview`).
   - Model prediction metrics grid.
   - Ground truth form with autofocus on input fields.
   - Save button with HTMX loading indicator.
   - Success toast / banner upon save.

---

## 5. Verification Plan

1. **Unit Tests**:
   - `parser_test.go`: Test `ParseSheetRows`, `BuildPageData`, and row indexing.
   - `sheets_test.go`: Test `AppendMissingClips` logic.
   - `server_test.go`: Test `GET /`, `GET /clip?row=N`, `POST /submit`, and `POST /sync-drive`.
   - `templates_test.go`: Test full page and partial template rendering.
2. **Compilation**:
   - `go test -v ./...`
   - `go build -o /dev/null .`
