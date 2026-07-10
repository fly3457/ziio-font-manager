package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fontManager/internal/models"
)

func TestStoreQueryFavoriteAndInstallRecord(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(rootDir, "Demo.ttf")
	fileID, added, err := db.UpsertFontFile(FileUpsert{
		RootID:           root.ID,
		Path:             filePath,
		FileName:         "Demo.ttf",
		Format:           "TTF",
		Size:             123,
		ModifiedAt:       "2026-06-24T00:00:00+08:00",
		Hash:             "abc",
		Status:           "ok",
		PreviewSupported: true,
	}, []models.FontFace{{
		FaceIndex:        0,
		Family:           "Demo",
		Style:            "Regular",
		FullName:         "Demo Regular",
		PostScriptName:   "Demo-Regular",
		Weight:           400,
		SampleText:       "Demo",
		PreviewSupported: true,
		Status:           "ok",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !added || fileID == 0 {
		t.Fatalf("added=%v fileID=%d", added, fileID)
	}

	items, err := db.QueryFonts(models.FontQuery{Query: "demo", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if err := db.SetFavorite(items[0].FaceID, true); err != nil {
		t.Fatal(err)
	}
	favorites, err := db.QueryFonts(models.FontQuery{FavoritesOnly: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(favorites) != 1 || !favorites[0].IsFavorite {
		t.Fatalf("favorite query returned %#v", favorites)
	}

	err = db.AddInstallRecord(models.InstallRecord{
		FileID:            items[0].FileID,
		FaceID:            items[0].FaceID,
		SourcePath:        filePath,
		TargetPath:        filePath,
		Mode:              "link",
		Scope:             "user",
		RegistryKey:       "HKCU",
		RegistryValueName: "Demo Regular (TrueType)",
		RegistryValueData: filePath,
		Status:            "installed",
	})
	if err != nil {
		t.Fatal(err)
	}
	installed, err := db.QueryFonts(models.FontQuery{InstalledOnly: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 || !installed[0].IsInstalled {
		t.Fatalf("installed query returned %#v", installed)
	}
	stats, err := db.FontStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.FavoriteCount != 1 || stats.InstalledCount != 1 {
		t.Fatalf("stats = %#v, want favorite=1 installed=1", stats)
	}
}

func TestStoreFolderListingAndFolderQuery(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	subPath := filepath.Join(root.Path, "Serif", "Demo.ttf")
	_, _, err = db.UpsertFontFile(FileUpsert{
		RootID:           root.ID,
		Path:             subPath,
		FileName:         "Demo.ttf",
		Format:           "TTF",
		Size:             123,
		ModifiedAt:       "2026-06-24T00:00:00+08:00",
		Hash:             "abc",
		Status:           "ok",
		PreviewSupported: true,
	}, []models.FontFace{{
		FaceIndex:        0,
		Family:           "Folder Demo",
		Style:            "Regular",
		FullName:         "Folder Demo Regular",
		Weight:           400,
		PreviewSupported: true,
		Status:           "ok",
	}})
	if err != nil {
		t.Fatal(err)
	}

	folders, err := db.ListFolders(root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) < 2 {
		t.Fatalf("expected root and child folder, got %#v", folders)
	}

	items, err := db.QueryFonts(models.FontQuery{RootID: root.ID, FolderPath: "Serif", FolderRecursive: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Family != "Folder Demo" {
		t.Fatalf("folder query returned %#v", items)
	}
}

func TestStoreMarkMissingFilesInScopeOnlyMarksSubtree(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	keepPath := filepath.Join(root.Path, "Serif", "Keep.ttf")
	missingPath := filepath.Join(root.Path, "Serif", "Gone.ttf")
	outsidePath := filepath.Join(root.Path, "Sans", "Outside.ttf")
	for _, item := range []struct {
		path   string
		family string
	}{
		{keepPath, "Keep"},
		{missingPath, "Gone"},
		{outsidePath, "Outside"},
	} {
		upsertTestFont(t, db, root.ID, item.path, item.family)
	}

	missing, err := db.MarkMissingFilesInScope(root.ID, filepath.Join(root.Path, "Serif"), map[string]struct{}{
		keepPath: {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if missing != 1 {
		t.Fatalf("missing = %d, want 1", missing)
	}

	items, err := db.QueryFonts(models.FontQuery{RootID: root.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	families := map[string]bool{}
	for _, item := range items {
		families[item.Family] = true
	}
	if !families["Keep"] || !families["Outside"] {
		t.Fatalf("expected Keep and Outside to remain visible, got %#v", families)
	}
	if families["Gone"] {
		t.Fatalf("Gone should have been marked missing: %#v", families)
	}
}

func TestStoreRemoveRootDeletesIndexedFonts(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rootDir := t.TempDir()
	root, err := db.AddRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(root.Path, "Demo.ttf")
	_, _, err = db.UpsertFontFile(FileUpsert{
		RootID:           root.ID,
		Path:             filePath,
		FileName:         "Demo.ttf",
		Format:           "TTF",
		Size:             123,
		ModifiedAt:       "2026-06-24T00:00:00+08:00",
		Hash:             "abc",
		Status:           "ok",
		PreviewSupported: true,
	}, []models.FontFace{{
		FaceIndex:        0,
		Family:           "Remove Demo",
		Style:            "Regular",
		FullName:         "Remove Demo Regular",
		Weight:           400,
		PreviewSupported: true,
		Status:           "ok",
	}})
	if err != nil {
		t.Fatal(err)
	}

	items, err := db.QueryFonts(models.FontQuery{RootID: root.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items before remove = %d, want 1", len(items))
	}

	if err := db.RemoveRoot(root.ID); err != nil {
		t.Fatal(err)
	}
	roots, err := db.ListRoots()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range roots {
		if item.ID == root.ID {
			t.Fatalf("removed root still listed: %#v", roots)
		}
	}
	items, err = db.QueryFonts(models.FontQuery{RootID: root.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items after remove = %d, want 0: %#v", len(items), items)
	}
}

func upsertTestFont(t *testing.T, db *Store, rootID int64, path, family string) {
	t.Helper()
	_, _, err := db.UpsertFontFile(FileUpsert{
		RootID:           rootID,
		Path:             path,
		FileName:         filepath.Base(path),
		Format:           "TTF",
		Size:             123,
		ModifiedAt:       "2026-06-24T00:00:00+08:00",
		Hash:             "hash-" + family,
		Status:           "ok",
		PreviewSupported: true,
	}, []models.FontFace{{
		FaceIndex:        0,
		Family:           family,
		Style:            "Regular",
		FullName:         family + " Regular",
		Weight:           400,
		PreviewSupported: true,
		Status:           "ok",
	}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStoreRejectsNestedUserRoots(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "Child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.AddRoot(parent); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddRoot(child); err == nil || !strings.Contains(err.Error(), "覆盖") {
		t.Fatalf("AddRoot child error = %v, want covered-by-parent rejection", err)
	}

	db2, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	if _, err := db2.AddRoot(child); err != nil {
		t.Fatal(err)
	}
	if _, err := db2.AddRoot(parent); err == nil || !strings.Contains(err.Error(), "位于该路径内") {
		t.Fatalf("AddRoot parent error = %v, want contains-existing-root rejection", err)
	}
}
