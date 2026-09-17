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
