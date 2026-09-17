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
