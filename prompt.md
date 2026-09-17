# ROLE & CONTEXT
You are an expert Go and frontend software engineer acting as an autonomous coding agent.
Your objective is to build a hyper-lightweight, zero-build web interface for reviewing traffic footage from Google Drive and logging ground-truth counts into an existing Google Sheet.

Stack Requirements:
- Backend: Go (Standard Library `net/http`, `html/template`)
- Frontend: Vanilla HTML/CSS + HTMX (loaded via unpkg CDN, NO Node.js/npm, NO JS frameworks)
- Storage / Integration: Google Sheets API v4 (`google.golang.org/api/sheets/v4`) using a Service Account (`credentials.json`)
- Video Playback: Google Drive embedded iframe preview player

---

# DATA SPECIFICATION
The target Google Sheet contains the following exact column format:
- Col A (1): `name` (e.g., "clip_1.mp4")
- Col B (2): `run` (e.g., "1")
- Col C (3): `in` (automated model count)
- Col D (4): `out` (automated model count)
- Col E (5): `actual_in` (ground truth to be filled)
- Col F (6): `actual_out` (ground truth to be filled)

A static or config-driven map translates video filenames (`name`) to Google Drive File IDs for iframe embedding.

---

# OBJECTIVE & UX FLOW
1. User visits `/`:
   - Server reads the sheet rows via Google Sheets API.
   - Server identifies the first row where `actual_in` or `actual_out` is blank/empty.
   - Server renders the base page (`index.html`) including a video card component (`task_card.html`).
2. Video Card Component:
   - Displays clip metadata: filename and run number.
   - Shows automated model counts: `In` and `Out` for visual reference.
   - Embeds the Google Drive preview: `https://drive.google.com/file/d/{DRIVE_FILE_ID}/preview`.
   - Renders a form with two numeric inputs: `actual_in` and `actual_out`.
   - Hidden inputs preserve `row_index`, `name`, and `run`.
3. User submits the form:
   - Form uses HTMX (`hx-post="/submit"`, `hx-target="#task-card"`, `hx-swap="outerHTML"`).
   - Server parses inputs and updates columns E and F for that specific row in Google Sheets via `Spreadsheets.Values.Update` (`USER_ENTERED`).
   - Server queries the next unannotated row.
   - Server returns ONLY the updated `task_card.html` partial (swapping the card in place without a full page refresh).
4. Completion:
   - If no pending clips remain, render a simple "All clips completed" state inside the card.

---

# NON-FUNCTIONAL REQUIREMENTS & CONSTRAINTS
- Strict Zero-Build: Do NOT introduce npm, yarn, Vite, Webpack, or Tailwind build tooling. All styling must be minimal embedded CSS in `templates/index.html`.
- No CGO: Avoid dependencies requiring CGO.
- Reliability: Handle Google API rate limits or network hiccups gracefully by logging errors cleanly without crashing the server.
- No Placeholders: Write complete, functional Go code and HTML templates. Do not emit truncated comments like `// ... rest of implementation`.

---

# PROJECT STRUCTURE TO GENERATE
- `main.go`: Server routing, HTTP handlers, configuration, and app entry point.
- `sheets.go`: Google Sheets client wrapper (`FetchAllClips`, `UpdateActuals`).
- `templates/index.html`: Base document layout loading HTMX from CDN.
- `templates/task_card.html`: The dynamic card partial swapped by HTMX.
- `go.mod`: Explicit Go module definition.

---

# VERIFICATION CHECKLIST
1. Run `go mod tidy` and verify no missing dependencies.
2. Ensure the code compiles cleanly via `go build -o /dev/null .`.
3. Verify that the HTMX form submission targets `#task-card` and uses `outerHTML` swap.
4. Verify that Google Sheet indices correctly handle the 1-based index and header row offset.
