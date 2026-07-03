package store

import (
	"path/filepath"
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
