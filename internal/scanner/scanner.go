package scanner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"fontManager/internal/diagnostics"
	"fontManager/internal/fontmeta"
	"fontManager/internal/models"
	"fontManager/internal/store"
)

const parseTimeout = 12 * time.Second

var errParseTimeout = errors.New("font metadata parse timed out")

type Scanner struct {
	store  *store.Store
	logDir string
}

var parseFontFile = fontmeta.ParseFile

func New(s *store.Store, logDir ...string) *Scanner {
	scanner := &Scanner{store: s}
	if len(logDir) > 0 {
		scanner.logDir = logDir[0]
	}
	return scanner
}

func (s *Scanner) ScanRoot(root models.LibraryRoot) (models.ScanResult, error) {
	result := models.ScanResult{RootID: root.ID}
	s.logScan("scan-start root=%d path=%q", root.ID, root.Path)
	jobID, err := s.store.BeginScan(root.ID)
	if err != nil {
		s.logScan("scan-begin-error root=%d path=%q error=%q", root.ID, root.Path, err.Error())
		return result, err
	}

	var candidates []string
	walkErr := filepath.WalkDir(root.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			result.Failed++
			s.logScan("walk-error root=%d path=%q error=%q", root.ID, path, err.Error())
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
		s.logScan("scan-walk-fatal root=%d path=%q candidates=%d error=%q", root.ID, root.Path, len(candidates), walkErr.Error())
		return result, walkErr
	}

	result.Total = len(candidates)
	s.logScan("scan-candidates root=%d path=%q total=%d", root.ID, root.Path, result.Total)
	seen := make(map[string]struct{}, len(candidates))
	_ = s.store.UpdateScan(jobID, "running", result.Total, result.Processed, result.Added, result.Updated, result.Failed, "Scanning font files")

	for _, path := range candidates {
		result.Processed++
		clean, err := filepath.Abs(path)
		if err == nil {
			path = filepath.Clean(clean)
		}
		seen[path] = struct{}{}
		s.logScan("current root=%d candidate=%d/%d path=%q", root.ID, result.Processed, result.Total, path)

		if info, err := os.Stat(path); err == nil {
			modifiedAt := info.ModTime().Format("2006-01-02T15:04:05-07:00")
			current, err := s.store.FileIsCurrent(path, info.Size(), modifiedAt)
			if err == nil && current {
				if result.Processed%100 == 0 || result.Processed == result.Total {
					s.logScan("progress root=%d processed=%d/%d added=%d updated=%d failed=%d message=%q", root.ID, result.Processed, result.Total, result.Added, result.Updated, result.Failed, "skipping unchanged files")
				}
				if result.Processed%50 == 0 || result.Processed == result.Total {
					_ = s.store.UpdateScan(jobID, "running", result.Total, result.Processed, result.Added, result.Updated, result.Failed, "Skipping unchanged files")
				}
				continue
			}
		}

		parsed, err := parseFileWithTimeout(path, parseTimeout)
		if err != nil {
			err = scanFileError(path, result.Processed, result.Total, err)
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
			s.logScan("file-error root=%d candidate=%d/%d path=%q error=%q", root.ID, result.Processed, result.Total, path, err.Error())
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
			s.logScan("upsert-error root=%d candidate=%d/%d path=%q error=%q", root.ID, result.Processed, result.Total, path, err.Error())
		} else if added {
			result.Added++
		} else {
			result.Updated++
		}
		if err == nil && parsed.File.Status == "error" {
			result.Failed++
			s.logScan("file-error-status root=%d candidate=%d/%d path=%q error=%q", root.ID, result.Processed, result.Total, path, parsed.File.Error)
		}
		if result.Processed%100 == 0 || result.Processed == result.Total {
			s.logScan("progress root=%d processed=%d/%d added=%d updated=%d failed=%d message=%q", root.ID, result.Processed, result.Total, result.Added, result.Updated, result.Failed, parsed.File.FileName)
		}
		if result.Processed%25 == 0 || result.Processed == result.Total {
			_ = s.store.UpdateScan(jobID, "running", result.Total, result.Processed, result.Added, result.Updated, result.Failed, parsed.File.FileName)
		}
	}

	if err := s.store.MarkMissingFiles(root.ID, seen); err != nil {
		result.Failed++
		_ = s.store.UpdateScan(jobID, "error", result.Total, result.Processed, result.Added, result.Updated, result.Failed, err.Error())
		s.logScan("mark-missing-error root=%d processed=%d/%d added=%d updated=%d failed=%d error=%q", root.ID, result.Processed, result.Total, result.Added, result.Updated, result.Failed, err.Error())
		return result, err
	}
	if err := s.store.FinishRootScan(root.ID); err != nil {
		result.Failed++
		_ = s.store.UpdateScan(jobID, "error", result.Total, result.Processed, result.Added, result.Updated, result.Failed, err.Error())
		s.logScan("finish-root-error root=%d processed=%d/%d added=%d updated=%d failed=%d error=%q", root.ID, result.Processed, result.Total, result.Added, result.Updated, result.Failed, err.Error())
		return result, err
	}

	status := "complete"
	if result.Failed > 0 {
		status = "complete_with_errors"
	}
	err = s.store.UpdateScan(jobID, status, result.Total, result.Processed, result.Added, result.Updated, result.Failed, "Scan finished")
	s.logScan("scan-finish root=%d status=%s total=%d processed=%d added=%d updated=%d failed=%d error=%q", root.ID, status, result.Total, result.Processed, result.Added, result.Updated, result.Failed, errorText(err))
	return result, err
}

func parseFileWithTimeout(path string, timeout time.Duration) (fontmeta.ParsedFile, error) {
	type parseResult struct {
		parsed fontmeta.ParsedFile
		err    error
	}
	done := make(chan parseResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- parseResult{err: fmt.Errorf("font metadata parser panic: %v", r)}
			}
		}()
		parsed, err := parseFontFile(path)
		done <- parseResult{parsed: parsed, err: err}
	}()
	select {
	case result := <-done:
		return result.parsed, result.err
	case <-time.After(timeout):
		return fontmeta.ParsedFile{}, errParseTimeout
	}
}

func scanFileError(path string, index, total int, err error) error {
	return fmt.Errorf("candidate %d/%d path=%q: %w", index, total, path, err)
}

func (s *Scanner) logScan(format string, args ...any) {
	diagnostics.Appendf(s.logDir, "scan.log", format, args...)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
