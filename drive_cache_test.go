package main

import (
	"testing"
)

func TestDriveCache_ManualPopulateAndGet(t *testing.T) {
	cache := &DriveCache{
		folderID: "test-folder",
		fileMap:  make(map[string]string),
	}

	cache.Set("clip_1.mp4", "drive_id_1")
	cache.Set("clip_2.mp4", "drive_id_2")

	if id := cache.Get("clip_1.mp4"); id != "drive_id_1" {
		t.Errorf("expected drive_id_1, got '%s'", id)
	}
	if id := cache.Get("clip_2.mp4"); id != "drive_id_2" {
		t.Errorf("expected drive_id_2, got '%s'", id)
	}
	if id := cache.Get("nonexistent.mp4"); id != "" {
		t.Errorf("expected empty string for missing file, got '%s'", id)
	}
}
