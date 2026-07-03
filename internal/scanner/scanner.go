package scanner

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"fontManager/internal/fontmeta"
	"fontManager/internal/models"
	"fontManager/internal/store"
)

const parseTimeout = 12 * time.Second

var errParseTimeout = errors.New("font metadata parse timed out")

type Scanner struct {
	store *store.Store
}

func New(s *store.Store) *Scanner {
	return &Scanner{store: s}
}

func (s *Scanner) ScanRoot(root models.LibraryRoot) (models.ScanResult, error) {
	result := models.ScanResult{RootID: root.ID}
	jobID, err := s.store.BeginScan(root.ID)
	if err != nil {
		return result, err
	}

	var candidates []string
	walkErr := filepath.WalkDir(root.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			result.Failed++
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if fontmeta.IsSupportedPath(path) {
			candidates = append(candidates, path)
		}
		return nil
	})
	if walkErr != nil {
		_ = s.store.UpdateScan(jobID, "error", len(candidates), result.Processed, result.Added, result.Updated, result.Failed, walkErr.Error())
		return result, walkErr
	}

	result.Total = len(candidates)
	seen := make(map[string]struct{}, len(candidates))
	_ = s.store.UpdateScan(jobID, "running", result.Total, result.Processed, result.Added, result.Updated, result.Failed, "Scanning font files")

	for _, path := range candidates {
		result.Processed++
		clean, err := filepath.Abs(path)
		if err == nil {
			path = filepath.Clean(clean)
		}
		seen[path] = struct{}{}

		if info, err := os.Stat(path); err == nil {
			modifiedAt := info.ModTime().Format("2006-01-02T15:04:05-07:00")
			current, err := s.store.FileIsCurrent(path, info.Size(), modifiedAt)
			if err == nil && current {
				if result.Processed%50 == 0 || result.Processed == result.Total {
					_ = s.store.UpdateScan(jobID, "running", result.Total, result.Processed, result.Added, result.Updated, result.Failed, "Skipping unchanged files")
				}
				continue
			}
		}

		parsed, err := parseFileWithTimeout(path, parseTimeout)
		if err != nil {
			result.Failed++
			if fallback, fallbackErr := fontmeta.ErrorFile(path, err); fallbackErr == nil {
				_, _, _ = s.store.UpsertFontFile(store.FileUpsert{
					RootID:           root.ID,
					Path:             path,
					FileName:         fallback.File.FileName,
					Format:           fallback.File.Format,
					Size:             fallback.File.Size,
					ModifiedAt:       fallback.File.ModifiedAt,
					Hash:             fallback.File.Hash,
					Status:           fallback.File.Status,
					Error:            fallback.File.Error,
					PreviewSupported: fallback.File.PreviewSupported,
				}, fallback.Faces)
			}
			_ = s.store.UpdateScan(jobID, "running", result.Total, result.Processed, result.Added, result.Updated, result.Failed, filepath.Base(path)+": "+err.Error())
			continue
		}
		parsed.File.RootID = root.ID
		fileID, added, err := s.store.UpsertFontFile(store.FileUpsert{
			RootID:           root.ID,
			Path:             path,
			FileName:         parsed.File.FileName,
			Format:           parsed.File.Format,
			Size:             parsed.File.Size,
			ModifiedAt:       parsed.File.ModifiedAt,
			Hash:             parsed.File.Hash,
			Status:           parsed.File.Status,
			Error:            parsed.File.Error,
			PreviewSupported: parsed.File.PreviewSupported,
		}, parsed.Faces)
		_ = fileID
		if err != nil {
			result.Failed++
		} else if added {
			result.Added++
		} else {
			result.Updated++
		}
		if result.Processed%25 == 0 || result.Processed == result.Total {
			_ = s.store.UpdateScan(jobID, "running", result.Total, result.Processed, result.Added, result.Updated, result.Failed, parsed.File.FileName)
		}
	}

	if err := s.store.MarkMissingFiles(root.ID, seen); err != nil {
		result.Failed++
		_ = s.store.UpdateScan(jobID, "error", result.Total, result.Processed, result.Added, result.Updated, result.Failed, err.Error())
		return result, err
	}
	if err := s.store.FinishRootScan(root.ID); err != nil {
		result.Failed++
		_ = s.store.UpdateScan(jobID, "error", result.Total, result.Processed, result.Added, result.Updated, result.Failed, err.Error())
		return result, err
	}

	status := "complete"
	if result.Failed > 0 {
		status = "complete_with_errors"
	}
	err = s.store.UpdateScan(jobID, status, result.Total, result.Processed, result.Added, result.Updated, result.Failed, "Scan finished")
	return result, err
}

func parseFileWithTimeout(path string, timeout time.Duration) (fontmeta.ParsedFile, error) {
	type parseResult struct {
		parsed fontmeta.ParsedFile
		err    error
	}
	done := make(chan parseResult, 1)
	go func() {
		parsed, err := fontmeta.ParseFile(path)
		done <- parseResult{parsed: parsed, err: err}
	}()
	select {
	case result := <-done:
		return result.parsed, result.err
	case <-time.After(timeout):
		return fontmeta.ParsedFile{}, errParseTimeout
	}
}
