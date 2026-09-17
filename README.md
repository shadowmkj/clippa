# Clippa — Traffic Footage Review & Ground Truth Logging

A lightweight, zero-build Go web application for reviewing video traffic footage from Google Drive and recording ground truth counts (`actual_in`, `actual_out`) directly into Google Sheets.

---

## 🌟 Key Features

- **Master-Detail Review Interface**: Full-screen collapsible sidebar with filterable clip list (`All`, `Pending`, `Done`).
- **Non-Sequential Random Access**: Multiple annotators can click and review any clip independently.
- **One-Click Google Drive Sync**: Automatically scans your Google Drive folder and imports any missing video filenames to Google Sheets.
- **Mobile-Friendly**: Slide-in responsive drawer with dimmed backdrop and touch-optimized controls for mobile and tablet labeling.
- **Zero-Build Architecture**: Vanilla HTML5/CSS and HTMX 2.0.4 loaded via CDN. No Node.js, npm, Webpack, or Tailwind required.
- **Embedded Google Drive Player**: Responsive 16:9 iframe video previews (`https://drive.google.com/file/d/{FILE_ID}/preview`).
- **Atomic Google Sheets Sync**: Updates columns `E` (`actual_in`) and `F` (`actual_out`) using `USER_ENTERED` formatting.
- **Seamless HTMX Transitions**: Form submissions update the sheet, show a save confirmation, and update the sidebar badge via HTMX Out-of-Band (`hx-swap-oob`) swap.

---

## 📊 Google Sheet Schema

The target Google Spreadsheet must follow this column structure:

| Col A (1) | Col B (2) | Col C (3) | Col D (4) | Col E (5) | Col F (6) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `name` | `run` | `in` | `out` | `actual_in` | `actual_out` |
| `clip_1.mp4` | `1` | `14` | `8` | *(ground truth)* | *(ground truth)* |

- **Header Row**: Row 1 contains headers.
- **Data Rows**: Begin at Row 2.
- **Pending Condition**: Any row where `actual_in` or `actual_out` is empty is queued for review.

---

## 🚀 Getting Started

### 1. Google Cloud Service Account Setup
1. Create a Google Cloud Service Account with access to the **Google Sheets API** and **Google Drive API**.
2. Download the JSON key file and save it as `credentials.json` in the project root.
3. Share your Google Sheet and Google Drive Folder with the service account client email (e.g., `xyz@project.iam.gserviceaccount.com`):
   - **Google Sheet**: Share with **Editor** role.
   - **Google Drive Folder**: Share with **Viewer** role.

### 2. Configuration
Create a `config.json` file in the root directory (or copy `config.example.json`):

```json
{
  "spreadsheet_id": "YOUR_GOOGLE_SPREADSHEET_ID",
  "sheet_name": "Sheet1",
  "drive_folder_id": "YOUR_GOOGLE_DRIVE_FOLDER_ID",
  "credentials_file": "credentials.json",
  "port": "8080"
}
```

*Alternatively, you can configure using environment variables:*
- `CLIPPA_SPREADSHEET_ID`
- `CLIPPA_SHEET_NAME`
- `CLIPPA_DRIVE_FOLDER_ID`
- `CLIPPA_CREDENTIALS_FILE`
- `PORT`

---

## 🐳 Running with Docker

### Using Docker Compose (Recommended)
```bash
docker compose up --build -d
```
Visit `http://localhost:8080`.

To view logs:
```bash
docker compose logs -f
```

To stop:
```bash
docker compose down
```

### Using Plain Docker
```bash
# 1. Build container image
docker build -t clippa:latest .

# 2. Run with configuration mounted
docker run -d \
  --name clippa \
  -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json:ro \
  -v $(pwd)/credentials.json:/app/credentials.json:ro \
  clippa:latest
```

---

## 💻 Running Locally without Docker

```bash
go run .
```

Open your browser and navigate to:
```
http://localhost:8080
```

---

## 🧪 Testing

Run all unit and integration tests:
```bash
go test -v ./...
```

Verify clean build without binary artifact output:
```bash
go build -o /dev/null .
```
