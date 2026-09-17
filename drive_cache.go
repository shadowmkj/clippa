package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"google.golang.org/api/drive/v3"
)

type DriveCache struct {
	srv      *drive.Service
	folderID string
	mu       sync.RWMutex
	fileMap  map[string]string
}

func NewDriveCache(srv *drive.Service, folderID string) *DriveCache {
	return &DriveCache{
		srv:      srv,
		folderID: folderID,
		fileMap:  make(map[string]string),
	}
}

func (d *DriveCache) Set(name, fileID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fileMap[name] = fileID
}

func (d *DriveCache) Get(name string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.fileMap[name]
}

func (d *DriveCache) Refresh(ctx context.Context) error {
	if d.srv == nil || d.folderID == "" {
		return nil
	}

	query := fmt.Sprintf("'%s' in parents and trashed = false", d.folderID)
	var pageToken string
	newMap := make(map[string]string)

	for {
		call := d.srv.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name)").
			PageSize(100).
			Context(ctx)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		res, err := call.Do()
		if err != nil {
			return fmt.Errorf("failed to list drive files: %w", err)
		}

		for _, file := range res.Files {
			newMap[file.Name] = file.Id
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	d.mu.Lock()
	d.fileMap = newMap
	d.mu.Unlock()

	log.Printf("[Drive] Cached %d files from folder %s", len(newMap), d.folderID)
	return nil
}

func (d *DriveCache) Resolve(name string) string {
	id := d.Get(name)
	if id != "" {
		return id
	}

	if d.srv == nil || d.folderID == "" {
		return ""
	}

	// Fallback single query on cache miss
	query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", name, d.folderID)
	res, err := d.srv.Files.List().Q(query).Fields("files(id, name)").PageSize(1).Do()
	if err == nil && len(res.Files) > 0 {
		d.Set(name, res.Files[0].Id)
		return res.Files[0].Id
	}

	return ""
}
