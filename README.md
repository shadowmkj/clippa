# Clippa — Traffic Footage Review & Ground Truth Logging

A lightweight, zero-build Go web application for reviewing video traffic footage from Google Drive and recording ground truth counts (`actual_in`, `actual_out`) directly into Google Sheets.

---

## 🌟 Key Features

- **Zero-Build Architecture**: Vanilla HTML5/CSS and HTMX 2.0.4 loaded via CDN. No Node.js, npm, Webpack, or Tailwind required.
- **Embedded Google Drive Player**: Automatically indexes files from a Google Drive folder and embeds responsive 16:9 iframe video previews (`https://drive.google.com/file/d/{FILE_ID}/preview`).
- **Atomic Google Sheets Sync**: Updates columns `E` (`actual_in`) and `F` (`actual_out`) using `USER_ENTERED` formatting.
- **Seamless HTMX Transitions**: Form submissions swap `#task-card` (`hx-swap="outerHTML"`) instantly without full page reloads.
- **Fast Keyboard Workflow**: Automatic autofocus on ground truth numeric inputs for rapid labeling.
- **Robust Error Handling**: Short-row padding, missing Drive file fallbacks, and non-crashing Google API error recovery.

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
Create a `config.json` file in the root directory (or use `config.example.json`):

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

### 3. Run the Application
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
