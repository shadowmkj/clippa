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
