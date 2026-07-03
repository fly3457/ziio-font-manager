package appdirs

import (
	"os"
	"path/filepath"
)

type Dirs struct {
	DataDir      string
	CacheDir     string
	LogDir       string
	DatabasePath string
}

func Resolve() (Dirs, error) {
	dataBase := os.Getenv("APPDATA")
	if dataBase == "" {
		var err error
		dataBase, err = os.UserConfigDir()
		if err != nil {
			return Dirs{}, err
		}
	}
	cacheBase := os.Getenv("LOCALAPPDATA")
	if cacheBase == "" {
		var err error
		cacheBase, err = os.UserCacheDir()
		if err != nil {
			return Dirs{}, err
		}
	}

	dirs := Dirs{
		DataDir:  filepath.Join(dataBase, "Ziio", "FontManager"),
		CacheDir: filepath.Join(cacheBase, "Ziio", "FontManager", "cache"),
	}
	dirs.DatabasePath = filepath.Join(dirs.DataDir, "fontmanager.db")
	dirs.LogDir = filepath.Join(dirs.DataDir, "logs")

	if err := os.MkdirAll(dirs.DataDir, 0o755); err != nil {
		return Dirs{}, err
	}
	if err := os.MkdirAll(dirs.CacheDir, 0o755); err != nil {
		return Dirs{}, err
	}
	if err := os.MkdirAll(dirs.LogDir, 0o755); err != nil {
		return Dirs{}, err
	}
	return dirs, nil
}
