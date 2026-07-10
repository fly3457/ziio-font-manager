package scanner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fontManager/internal/fontmeta"
	"fontManager/internal/models"
	"fontManager/internal/store"
)

func TestParseFileWithTimeoutRecoversParserPanic(t *testing.T) {
	restore := replaceParser(func(path string) (fontmeta.ParsedFile, error) {
		panic("boom")
	})
	defer restore()

	_, err := parseFileWithTimeout("bad.ttf", time.Second)
	if err == nil {
		t.Fatal("expected panic to be returned as error")
	}
	if !strings.Contains(err.Error(), "font metadata parser panic") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanRootRecoversParserPanicAndWritesLog(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "bad.ttf"), []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "good.ttf"), []byte("good"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	restore := replaceParser(func(path string) (fontmeta.ParsedFile, error) {
		if filepath.Base(path) == "bad.ttf" {
			panic("parser exploded")
		}
		return parsedOK(path), nil
	})
	defer restore()

	logDir := t.TempDir()
	result, err := New(db, logDir).ScanRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || result.Processed != 2 || result.Failed != 1 {
		t.Fatalf("result = %#v, want total=2 processed=2 failed=1", result)
	}
	status, err := db.LatestScanStatus(root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "complete_with_errors" {
		t.Fatalf("status = %q, want complete_with_errors", status.Status)
	}
	items, err := db.QueryFonts(models.FontQuery{RootID: root.ID, Query: "bad", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("bad item query returned %#v", items)
	}
	if !strings.Contains(items[0].Error, "candidate") || !strings.Contains(items[0].Error, "font metadata parser panic") {
		t.Fatalf("bad item error = %q", items[0].Error)
	}
	logText := readLog(t, filepath.Join(logDir, "scan.log"))
	for _, want := range []string{"scan-start", "current", "bad.ttf", "font metadata parser panic", "scan-finish"} {
		if !strings.Contains(logText, want) {
			t.Fatalf("scan log missing %q:\n%s", want, logText)
		}
	}
}

func TestScanRootCountsParsedErrorStatusAsFailed(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "broken.ttf"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	restore := replaceParser(func(path string) (fontmeta.ParsedFile, error) {
		parsed := parsedOK(path)
		parsed.File.Status = "error"
		parsed.File.Error = "bad table"
		parsed.Faces[0].Status = "error"
		parsed.Faces[0].Error = "bad table"
		return parsed, nil
	})
	defer restore()

	result, err := New(db, t.TempDir()).ScanRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 1 {
		t.Fatalf("failed = %d, want 1", result.Failed)
	}
	status, err := db.LatestScanStatus(root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "complete_with_errors" {
		t.Fatalf("status = %q, want complete_with_errors", status.Status)
	}
}

func TestScanRootContinuesAfterParserError(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "broken.ttf"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "good.ttf"), []byte("good"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	restore := replaceParser(func(path string) (fontmeta.ParsedFile, error) {
		if filepath.Base(path) == "broken.ttf" {
			return fontmeta.ParsedFile{}, errors.New("bad font")
		}
		return parsedOK(path), nil
	})
	defer restore()

	result, err := New(db, t.TempDir()).ScanRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Processed != 2 || result.Failed != 1 {
		t.Fatalf("result = %#v, want processed=2 failed=1", result)
	}
	items, err := db.QueryFonts(models.FontQuery{RootID: root.ID, Query: "good", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Family != "good.ttf" {
		t.Fatalf("good item missing after parser error: %#v", items)
	}
}

func TestScanRootCountsUnchangedFiles(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(root.Path, "current.ttf")
	if err := os.WriteFile(filePath, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	indexCurrentTestFont(t, db, root.ID, filePath, "Current")

	parserCalled := false
	restore := replaceParser(func(path string) (fontmeta.ParsedFile, error) {
		parserCalled = true
		return fontmeta.ParsedFile{}, errors.New("parser should not be called for unchanged files")
	})
	defer restore()

	result, err := New(db, t.TempDir()).ScanRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Processed != 1 || result.Unchanged != 1 || result.Failed != 0 {
		t.Fatalf("result = %#v, want total=1 processed=1 unchanged=1 failed=0", result)
	}
	if parserCalled {
		t.Fatal("parser was called for an unchanged file")
	}
	status, err := db.LatestScanStatus(root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Unchanged != 1 || status.Scope != "root" {
		t.Fatalf("status = %#v, want unchanged=1 scope=root", status)
	}
}

func TestScanFolderMarksMissingOnlyWithinFolder(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	serifDir := filepath.Join(root.Path, "Serif")
	sansDir := filepath.Join(root.Path, "Sans")
	if err := os.MkdirAll(serifDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sansDir, 0o755); err != nil {
		t.Fatal(err)
	}
	keepPath := filepath.Join(serifDir, "keep.ttf")
	gonePath := filepath.Join(serifDir, "gone.ttf")
	outsidePath := filepath.Join(sansDir, "outside.ttf")
	for _, path := range []string{keepPath, outsidePath} {
		if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	indexCurrentTestFont(t, db, root.ID, keepPath, "Keep")
	indexMissingTestFont(t, db, root.ID, gonePath, "Gone")
	indexCurrentTestFont(t, db, root.ID, outsidePath, "Outside")

	parserCalled := false
	restore := replaceParser(func(path string) (fontmeta.ParsedFile, error) {
		parserCalled = true
		return fontmeta.ParsedFile{}, errors.New("parser should not be called for unchanged files")
	})
	defer restore()

	result, err := New(db, t.TempDir()).ScanFolder(root, "Serif")
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Processed != 1 || result.Unchanged != 1 || result.Missing != 1 {
		t.Fatalf("result = %#v, want total=1 processed=1 unchanged=1 missing=1", result)
	}
	if parserCalled {
		t.Fatal("parser was called for an unchanged file")
	}
	items, err := db.QueryFonts(models.FontQuery{RootID: root.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	families := map[string]bool{}
	for _, item := range items {
		families[item.Family] = true
	}
	if !families["Keep"] || !families["Outside"] || families["Gone"] {
		t.Fatalf("visible families after folder scan = %#v", families)
	}
}

func TestScanFolderMissingDirectoryMarksSubtreeMissing(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	indexMissingTestFont(t, db, root.ID, filepath.Join(root.Path, "Deleted", "one.ttf"), "One")
	indexMissingTestFont(t, db, root.ID, filepath.Join(root.Path, "Deleted", "two.ttf"), "Two")
	outsidePath := filepath.Join(root.Path, "Outside.ttf")
	if err := os.WriteFile(outsidePath, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	indexCurrentTestFont(t, db, root.ID, outsidePath, "Outside")

	result, err := New(db, t.TempDir()).ScanFolder(root, "Deleted")
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 || result.Missing != 2 || result.Failed != 0 {
		t.Fatalf("result = %#v, want total=0 missing=2 failed=0", result)
	}
	items, err := db.QueryFonts(models.FontQuery{RootID: root.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Family != "Outside" {
		t.Fatalf("visible items after missing folder scan = %#v", items)
	}
}

func replaceParser(parser func(string) (fontmeta.ParsedFile, error)) func() {
	old := parseFontFile
	parseFontFile = parser
	return func() {
		parseFontFile = old
	}
}

func parsedOK(path string) fontmeta.ParsedFile {
	name := filepath.Base(path)
	return fontmeta.ParsedFile{
		File: models.FontFile{
			Path:             path,
			FileName:         name,
			Format:           "TTF",
			Size:             4,
			ModifiedAt:       "2026-07-03T00:00:00+08:00",
			Hash:             "hash-" + name,
			Status:           "ok",
			PreviewSupported: true,
		},
		Faces: []models.FontFace{{
			FaceIndex:        0,
			Family:           name,
			Style:            "Regular",
			FullName:         name,
			PostScriptName:   strings.TrimSuffix(name, filepath.Ext(name)),
			Weight:           400,
			PreviewSupported: true,
			Status:           "ok",
		}},
	}
}

func indexCurrentTestFont(t *testing.T, db *store.Store, rootID int64, path, family string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	indexTestFont(t, db, rootID, path, family, info.Size(), info.ModTime().Format("2006-01-02T15:04:05-07:00"))
}

func indexMissingTestFont(t *testing.T, db *store.Store, rootID int64, path, family string) {
	t.Helper()
	indexTestFont(t, db, rootID, path, family, 123, "2026-06-24T00:00:00+08:00")
}

func indexTestFont(t *testing.T, db *store.Store, rootID int64, path, family string, size int64, modifiedAt string) {
	t.Helper()
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	cleanPath = filepath.Clean(cleanPath)
	_, _, err = db.UpsertFontFile(store.FileUpsert{
		RootID:           rootID,
		Path:             cleanPath,
		FileName:         filepath.Base(cleanPath),
		Format:           "TTF",
		Size:             size,
		ModifiedAt:       modifiedAt,
		Hash:             "hash-" + family,
		Status:           "ok",
		PreviewSupported: true,
	}, []models.FontFace{{
		FaceIndex:        0,
		Family:           family,
		Style:            "Regular",
		FullName:         family + " Regular",
		PostScriptName:   family + "-Regular",
		Weight:           400,
		PreviewSupported: true,
		Status:           "ok",
	}})
	if err != nil {
		t.Fatal(err)
	}
}

func readLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
