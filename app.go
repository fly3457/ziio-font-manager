package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fontManager/internal/appdirs"
	"fontManager/internal/diagnostics"
	"fontManager/internal/library"
	"fontManager/internal/models"
	"fontManager/internal/store"
)

// App struct
type App struct {
	ctx context.Context

	Dirs           appdirs.Dirs
	Store          *store.Store
	LibraryService *library.LibraryService
	FontService    *library.FontService
	InstallService *library.InstallService
}

// NewApp creates a new App application struct
func NewApp() (*App, error) {
	dirs, err := appdirs.Resolve()
	if err != nil {
		return nil, err
	}
	diagnostics.Appendf(dirs.LogDir, "app.log", "app-start dataDir=%q cacheDir=%q logDir=%q db=%q", dirs.DataDir, dirs.CacheDir, dirs.LogDir, dirs.DatabasePath)
	db, err := store.Open(dirs.DatabasePath)
	if err != nil {
		diagnostics.Appendf(dirs.LogDir, "app.log", "store-open-error db=%q error=%q", dirs.DatabasePath, err.Error())
		return nil, err
	}
	if err := db.InterruptRunningScans("应用重启后重置未完成扫描"); err != nil {
		_ = db.Close()
		diagnostics.Appendf(dirs.LogDir, "app.log", "interrupt-running-scans-error error=%q", err.Error())
		return nil, err
	}
	diagnostics.Append(dirs.LogDir, "app.log", "running scans reset after startup")

	app := &App{
		Dirs:  dirs,
		Store: db,
	}
	app.LibraryService = library.NewLibraryService(db, dirs.LogDir)
	app.FontService = library.NewFontService(db, dirs.CacheDir)
	app.InstallService = library.NewInstallService(db)
	return app, nil
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.LibraryService.SetContext(ctx)
	a.FontService.SetContext(ctx)
	a.InstallService.SetContext(ctx)
	diagnostics.Append(a.Dirs.LogDir, "app.log", "wails-startup-complete")
}

func (a *App) GetAppInfo() models.AppInfo {
	return models.AppInfo{
		Name:         "Ziio Font Manager",
		Version:      "0.1.0",
		DataDir:      a.Dirs.DataDir,
		CacheDir:     a.Dirs.CacheDir,
		LogDir:       a.Dirs.LogDir,
		DatabasePath: a.Dirs.DatabasePath,
	}
}

func (a *App) previewFontHandler() http.Handler {
	previewDir := filepath.Join(a.Dirs.CacheDir, "previews")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		const prefix = "/preview-fonts/"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}

		name := strings.TrimPrefix(r.URL.Path, prefix)
		if name == "" || name != filepath.Base(name) || filepath.Ext(name) != ".otf" {
			http.NotFound(w, r)
			return
		}
		for _, ch := range strings.TrimSuffix(name, ".otf") {
			if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
				http.NotFound(w, r)
				return
			}
		}

		path := filepath.Join(previewDir, name)
		if _, err := os.Stat(path); err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "font/otf")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, path)
	})
}
