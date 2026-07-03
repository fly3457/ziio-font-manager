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

func readLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
