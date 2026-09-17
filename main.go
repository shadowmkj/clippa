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
