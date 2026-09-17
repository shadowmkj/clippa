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
