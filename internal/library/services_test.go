package library

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fontManager/internal/store"
)

func TestPreviewCacheKeyIncludesSampleHash(t *testing.T) {
	fontHash := "abcdef0123456789"
	first := previewCacheKey(fontHash, "", 0, previewSampleHash("AaBbCc"))
	same := previewCacheKey(fontHash, "", 0, previewSampleHash("AaBbCc"))
	other := previewCacheKey(fontHash, "", 0, previewSampleHash("012345"))

	if !strings.HasPrefix(first, "v2-") {
		t.Fatalf("cache key = %q, want v2 prefix", first)
	}
	if first != same {
		t.Fatalf("same sample text produced different cache keys: %q vs %q", first, same)
	}
	if first == other {
		t.Fatalf("different sample text produced same cache key: %q", first)
	}
}

func TestLibraryServiceRemoveRootRejectsSystemRoot(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root, err := db.AddRootWithKind(t.TempDir(), "system", "系统字库")
	if err != nil {
		t.Fatal(err)
	}
	service := NewLibraryService(db)

	err = service.RemoveRoot(root.ID)
	if err == nil || !strings.Contains(err.Error(), "系统字库") {
		t.Fatalf("RemoveRoot error = %v, want system root rejection", err)
	}
	if _, err := db.RootByID(root.ID); err != nil {
		t.Fatalf("system root should remain after rejected remove: %v", err)
	}
}

func TestLibraryServiceRemoveRootRejectsRunningRoot(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root, err := db.AddRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewLibraryService(db)
	service.mu.Lock()
	service.running[root.ID] = true
	service.mu.Unlock()

	err = service.RemoveRoot(root.ID)
	if err == nil || !strings.Contains(err.Error(), "正在扫描") {
		t.Fatalf("RemoveRoot error = %v, want running root rejection", err)
	}
	if _, err := db.RootByID(root.ID); err != nil {
		t.Fatalf("running root should remain after rejected remove: %v", err)
	}
}

func TestLibraryServiceRemoveRootRejectsRunningScanStatus(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root, err := db.AddRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.BeginScan(root.ID); err != nil {
		t.Fatal(err)
	}
	service := NewLibraryService(db)

	err = service.RemoveRoot(root.ID)
	if err == nil || !strings.Contains(err.Error(), "正在扫描") {
		t.Fatalf("RemoveRoot error = %v, want running scan status rejection", err)
	}
	if _, err := db.RootByID(root.ID); err != nil {
		t.Fatalf("running scan root should remain after rejected remove: %v", err)
	}
}

func TestLibraryServiceRescanAllRootsSkipsRunningRoots(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	runningRoot, err := db.AddRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	idleRoot, err := db.AddRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewLibraryService(db)
	service.mu.Lock()
	service.running[runningRoot.ID] = true
	service.mu.Unlock()
	defer func() {
		service.mu.Lock()
		delete(service.running, runningRoot.ID)
		service.mu.Unlock()
	}()

	queued, err := service.RescanAllRoots()
	if err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("queued = %d, want 1", queued)
	}
	waitForIdle(t, service, idleRoot.ID)
}

func TestLibraryServiceRescanFolderRejectsRunningRootAndTraversal(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	root, err := db.AddRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewLibraryService(db)
	service.mu.Lock()
	service.running[root.ID] = true
	service.mu.Unlock()

	if _, err := service.RescanFolder(root.ID, "Sub"); err == nil || !strings.Contains(err.Error(), "正在扫描") {
		t.Fatalf("RescanFolder running error = %v, want running rejection", err)
	}
	service.mu.Lock()
	delete(service.running, root.ID)
	service.mu.Unlock()

	if _, err := service.RescanFolder(root.ID, ".."); err == nil || !strings.Contains(err.Error(), "不在字体库内") {
		t.Fatalf("RescanFolder traversal error = %v, want path rejection", err)
	}
}

func waitForIdle(t *testing.T, service *LibraryService, rootID int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		service.mu.Lock()
		running := service.running[rootID]
		service.mu.Unlock()
		if !running {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("root %d still running after timeout", rootID)
}
